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
	"time"

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	inf "gopkg.in/inf.v0"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create,versions=v1,name=cpuboost.autoscaling.x-k8s.io,admissionReviewVersions=v1

type podCPUBoostHandler struct {
	decoder *admission.Decoder
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
	log := ctrl.LoggerFrom(ctx).WithName("cpuboost-webhook").WithValues("pod.Name", pod.Name, "pod.Namespace", pod.Namespace)
	log.V(5).Info("Handling Pod")

	boostImpl, ok := h.manager.GetStartupCPUBoostForPod(pod)
	if !ok {
		log.V(5).Info("StartupCPUBoost was not found")
		return admission.Allowed("no StartupCPUBoost matched")
	}
	containers, ok := h.boostContainersCPU(pod, boostImpl.BoostPercent(), log)
	if !ok {
		log.V(5).Info("no suitable CPU requests were found")
		return admission.Allowed("no CPU request found")
	}
	pod.Spec.Containers = containers
	pod.ObjectMeta.Labels[boost.StartupCPUBoostPodLabelKey] = boostImpl.Name()
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

/*
func (h *podCPUBoostHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}
*/

func (h *podCPUBoostHandler) boostContainersCPU(pod *corev1.Pod, boostPerc int64, log logr.Logger) (result []corev1.Container, boosted bool) {
	result = pod.Spec.Containers
	now := time.Now()
	boostAnnot := *boost.NewStartupCPUBoostPodAnnotation(&now)
	for _, container := range pod.Spec.Containers {
		log = log.WithValues("container.Name", container.Name)
		if boostedReq, initReq, _ := increaseQuantityForResource(container.Resources.Requests, corev1.ResourceCPU, boostPerc, log.WithValues("resourceRequirement", "request")); boostedReq {
			boosted = true
			boostAnnot.InitCPURequests[container.Name] = initReq.String()
		}
		if boostedLimit, initLimit, _ := increaseQuantityForResource(container.Resources.Limits, corev1.ResourceCPU, boostPerc, log.WithValues("resourceRequirement", "limit")); boostedLimit {
			boostAnnot.InitCPULimits[container.Name] = initLimit.String()
		}
	}
	if boosted {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[boost.StartupCPUBoostPodAnnotationKey] = boostAnnot.MustMarshalToJSON()
	}
	return
}

func increaseQuantityForResource(resources corev1.ResourceList, resName corev1.ResourceName, incPerc int64, log logr.Logger) (increased bool, init, new *apiResource.Quantity) {
	if quantity, ok := resources[resName]; ok {
		newQuantity := increaseQuantity(quantity, incPerc)
		log = log.WithValues(resName.String(), quantity.String(), "incPercent", incPerc,
			resName.String()+"New", newQuantity.String())
		log.V(2).Info("increasing container resource quantity")
		resources[corev1.ResourceCPU] = *newQuantity
		init = &quantity
		new = newQuantity
		increased = true
	}
	return
}

func increaseQuantity(quantity apiResource.Quantity, incPerc int64) *apiResource.Quantity {
	quantityDec := quantity.AsDec()
	decPerc := inf.NewDec(100+incPerc, 2)
	decResult := &inf.Dec{}
	decResult.Mul(quantityDec, decPerc)
	decRoundedResult := inf.Dec{}
	decRoundedResult.Round(decResult, 2, inf.RoundCeil)
	return apiResource.NewDecimalQuantity(decRoundedResult, quantity.Format)
}
