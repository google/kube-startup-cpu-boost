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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type BoostPodHandler interface {
	Create(context.Context, event.CreateEvent,
		workqueue.TypedRateLimitingInterface[reconcile.Request])
	Update(context.Context, event.UpdateEvent,
		workqueue.TypedRateLimitingInterface[reconcile.Request])
	Delete(context.Context, event.DeleteEvent,
		workqueue.TypedRateLimitingInterface[reconcile.Request])
	Generic(context.Context, event.GenericEvent,
		workqueue.TypedRateLimitingInterface[reconcile.Request])
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

func (h *boostPodHandler) Create(ctx context.Context, e event.CreateEvent,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod create")
	boost, err := h.manager.UpsertPod(ctx, pod)
	if err != nil {
		log.Error(err, "failed to handle pod create")
		return
	}
	if boost != nil {
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      boost.Name(),
				Namespace: boost.Namespace(),
			},
		})
	}
}

func (h *boostPodHandler) Delete(ctx context.Context, e event.DeleteEvent,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod delete")
	boost, err := h.manager.DeletePod(ctx, pod)
	if err != nil {
		log.Error(err, "failed to handle pod delete")
		return
	}
	if boost != nil {
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      boost.Name(),
				Namespace: boost.Namespace(),
			},
		})
	}
}

func (h *boostPodHandler) Update(ctx context.Context, e event.UpdateEvent,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	pod, ok := e.ObjectNew.(*corev1.Pod)
	oldPod, ok_ := e.ObjectOld.(*corev1.Pod)
	if !ok || !ok_ {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod update")
	if equality.Semantic.DeepEqual(pod.Status.Conditions, oldPod.Status.Conditions) {
		log.V(5).Info("pod update skipped: conditions did not change")
		return
	}
	boost, err := h.manager.UpsertPod(ctx, pod)
	if err != nil {
		log.Error(err, "failed to handle pod update")
		return
	}
	if boost != nil {
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      boost.Name(),
				Namespace: boost.Namespace(),
			},
		})
	}
}

func (h *boostPodHandler) Generic(ctx context.Context, e event.GenericEvent,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		return
	}
	log := h.log.WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling pod generic event")
}

func (h *boostPodHandler) GetPodLabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      bpod.BoostLabelKey,
				Operator: metav1.LabelSelectorOpExists,
				Values:   []string{},
			},
		},
	}
}
