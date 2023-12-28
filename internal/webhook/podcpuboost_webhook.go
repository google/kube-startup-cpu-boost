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

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	corev1 "k8s.io/api/core/v1"
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
	log := ctrl.LoggerFrom(ctx).WithName("cpuboost-webhook")
	log.V(5).Info("handling Pod")

	boostImpl, ok := h.manager.StartupCPUBoostForPod(ctx, pod)
	if !ok {
		log.V(5).Info("no startupCPUBoost matched")
		return admission.Allowed("no StartupCPUBoost matched")
	}
	log = log.WithValues("boost", boostImpl.Name())
	h.boostContainerResources(ctx, boostImpl, pod, log)
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (h *podCPUBoostHandler) boostContainerResources(ctx context.Context, b boost.StartupCPUBoost, pod *corev1.Pod, log logr.Logger) {
	annotation := bpod.NewBoostAnnotation()
	for i, container := range pod.Spec.Containers {
		policy, found := b.ResourcePolicy(container.Name)
		if !found {
			continue
		}
		log = log.WithValues("container", container.Name,
			"CPURequests", container.Resources.Requests.Cpu().String(),
			"CPULimits", container.Resources.Limits.Cpu().String(),
		)
		if resizeRequiresRestart(container, corev1.ResourceCPU) {
			log.Info("skipping container due to restart policy")
			continue
		}
		updateBoostAnnotation(annotation, container.Name, container.Resources)
		resources := policy.NewResources(ctx, &container)
		log = log.WithValues(
			"newCPURequests", resources.Requests.Cpu().String(),
			"newCPULimits", resources.Limits.Cpu().String(),
		)
		log.Info("increasing resources")
		pod.Spec.Containers[i].Resources = *resources
	}
	if len(annotation.InitCPULimits) > 0 || len(annotation.InitCPURequests) > 0 {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations[bpod.BoostAnnotationKey] = annotation.ToJSON()
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		pod.Labels[bpod.BoostLabelKey] = b.Name()
	}
}

func updateBoostAnnotation(annot *bpod.BoostPodAnnotation, containerName string, resources corev1.ResourceRequirements) {
	if cpuRequests, ok := resources.Requests[corev1.ResourceCPU]; ok {
		annot.InitCPURequests[containerName] = cpuRequests.String()
	}
	if cpuLimits, ok := resources.Limits[corev1.ResourceCPU]; ok {
		annot.InitCPULimits[containerName] = cpuLimits.String()
	}
}

func resizeRequiresRestart(c corev1.Container, r corev1.ResourceName) bool {
	for _, p := range c.ResizePolicy {
		if p.ResourceName != r {
			continue
		}
		return p.RestartPolicy == corev1.RestartContainer
	}
	return false
}
