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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type boostPodHandler struct {
	manager boost.Manager
	log     logr.Logger
}

func (h *boostPodHandler) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		h.log.V(2).Info("Pod create event contains non-pod object")
		return
	}
	log := h.log.WithValues("pod.Name", pod.Name, "pod.Namespace", pod.Namespace)
	log.V(5).Info("Pod create event")
	boostPod, err := boost.NewStartupCPUBoostPod(pod)
	if err != nil {
		log = log.WithValues("error", err.Error())
		log.V(2).Info("failed to parse startup cpu boost pod")
		return
	}
	boost, ok := h.manager.GetStartupCPUBoost(boostPod.GetNamespace(), boostPod.GetBoostName())
	if !ok {
		log.V(2).Info("failed to get startup cpu boost for a pod")
		return
	}
	log = log.WithValues("boost.Name", boost.Name())
	log.V(5).Info("Found boost in manager")
	if err := boost.AddPod(boostPod); err != nil {
		log.Error(err, "failed to add pod to startup cpu boost")
		return
	}
}

func (h *boostPodHandler) Delete(context.Context, event.DeleteEvent, workqueue.RateLimitingInterface) {
}

func (h *boostPodHandler) Update(context.Context, event.UpdateEvent, workqueue.RateLimitingInterface) {
}

func (h *boostPodHandler) Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface) {
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
