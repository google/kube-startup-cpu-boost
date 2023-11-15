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

package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type BoostPodHandler interface {
	Create(context.Context, event.CreateEvent, workqueue.RateLimitingInterface)
	Delete(context.Context, event.DeleteEvent, workqueue.RateLimitingInterface)
	Update(context.Context, event.UpdateEvent, workqueue.RateLimitingInterface)
	Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface)
	GetPodLabelSelector() *metav1.LabelSelector
}

type boostPodHandler struct {
	manager boost.Manager
	log     logr.Logger
}

func NewBoostPodHandler(manager boost.Manager, log logr.Logger) BoostPodHandler {
	return &boostPodHandler{
		manager: manager,
		log:     log,
	}
}

func (h *boostPodHandler) Create(ctx context.Context, e event.CreateEvent, wq workqueue.RateLimitingInterface) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod create")
	boost, ok := h.boostForPod(pod)
	if !ok {
		log.V(5).Info("failed to get boost for pod")
		return
	}
	log.WithValues("boost", boost.Name())
	if err := boost.UpsertPod(ctx, pod); err != nil {
		log.Error(err, "failed to handle pod create")
	}
}

func (h *boostPodHandler) Delete(ctx context.Context, e event.DeleteEvent, wq workqueue.RateLimitingInterface) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod delete")
	boost, ok := h.boostForPod(pod)
	if !ok {
		log.V(5).Info("failed to get boost for pod")
		return
	}
	if err := boost.DeletePod(ctx, pod); err != nil {
		log.Error(err, "failed to handle pod delete")
	}
}

func (h *boostPodHandler) Update(ctx context.Context, e event.UpdateEvent, wq workqueue.RateLimitingInterface) {
	pod, ok := e.ObjectNew.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod update")
	//TODO react only on POD or container condition updates
	boost, ok := h.boostForPod(pod)
	if !ok {
		log.V(5).Info("failed to get boost for pod")
		return
	}
	if err := boost.UpsertPod(ctx, pod); err != nil {
		log.Error(err, "failed to handle pod update")
	}
}

func (h *boostPodHandler) Generic(ctx context.Context, e event.GenericEvent, wq workqueue.RateLimitingInterface) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("got pod generic event")
}

func (h *boostPodHandler) GetPodLabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      boost.StartupCPUBoostPodLabelKey,
				Operator: metav1.LabelSelectorOpExists,
				Values:   []string{},
			},
		},
	}
}

func (h *boostPodHandler) boostForPod(pod *corev1.Pod) (boost.StartupCPUBoost, bool) {
	boostName, ok := pod.Labels[bpod.BoostLabelKey]
	if !ok {
		return nil, false
	}
	return h.manager.StartupCPUBoost(pod.Namespace, boostName)
}
