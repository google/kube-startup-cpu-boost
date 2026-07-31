// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/kube-startup-cpu-boost/internal/boost"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,timeoutSeconds=2,groups="",resources=pods,verbs=create,versions=v1,name=cpuboost.autoscaling.x-k8s.io,admissionReviewVersions=v1

type podCPUBoostHandler struct {
	decoder admission.Decoder
	manager boost.Manager
}

func NewPodCPUBoostWebHook(mgr boost.Manager, scheme *runtime.Scheme) *webhook.Admission {
	return &webhook.Admission{
		Handler: &podCPUBoostHandler{
			manager: mgr,
			decoder: admission.NewDecoder(scheme),
		},
	}
}

func (h *podCPUBoostHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := h.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	log := ctrl.LoggerFrom(ctx).WithName("boost-pod-webhook")
	log.V(5).Info("handling pod")

	boostImpl, ok := h.manager.GetCPUBoostForPod(ctx, pod)
	if !ok {
		log.V(5).Info("no boost matched")
		return admission.Allowed("no boost matched")
	}
	log = log.WithValues("boost", boostImpl.Name())

	err = boostImpl.ApplyResourcePolicy(ctx, pod)
	if err != nil {
		log.Error(err, "failed to apply resource policy")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
