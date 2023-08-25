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

// Package boost contains logic for managing startup resource boosts
package boost

import (
	"context"
	"errors"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

var (
	errStartupCPUBoostAlreadyExists = errors.New("startupCPUBoost already exists")
	errInvalidStartupCPUBoostSpec   = errors.New("invalid startupCPUBoost spec")
)

const (
	DefaultManagerCheckInterval     = time.Duration(5 * time.Second)
	StartupCPUBoostPodLabelKey      = "autoscaling.x-k8s.io/startup-cpu-boost"
	StartupCPUBoostPodAnnotationKey = "autoscaling.x-k8s.io/startup-cpu-boost"
)

type Manager interface {
	AddStartupCPUBoost(ctx context.Context, boost *autoscaling.StartupCPUBoost) error
	DeleteStartupCPUBoost(boost *autoscaling.StartupCPUBoost)
	GetStartupCPUBoostForPod(pod *corev1.Pod) (*startupCPUBoost, bool)
	GetStartupCPUBoost(namespace string, name string) (*startupCPUBoost, bool)
	Start(ctx context.Context) error
}

type managerImpl struct {
	sync.RWMutex
	client           client.Client
	startupCPUBoosts map[string]map[string]*startupCPUBoost
	checkInterval    time.Duration
}

func NewManager(client client.Client) Manager {
	return &managerImpl{
		client:           client,
		startupCPUBoosts: make(map[string]map[string]*startupCPUBoost),
		checkInterval:    DefaultManagerCheckInterval,
	}
}

func (m *managerImpl) AddStartupCPUBoost(ctx context.Context, boost *autoscaling.StartupCPUBoost) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.getStartupCPUBoostWithNS(boost.Namespace, boost.Name); ok {
		return errStartupCPUBoostAlreadyExists
	}
	boostImpl, err := newStartupCPUBoost(boost)
	if err != nil {
		return errInvalidStartupCPUBoostSpec
	}
	m.addStartupCPUBoostWithNS(boostImpl)
	return nil
}

func (m *managerImpl) DeleteStartupCPUBoost(boost *autoscaling.StartupCPUBoost) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.getStartupCPUBoostWithNS(boost.Namespace, boost.Name); !ok {
		return
	}
	m.deleteStartupCPUBoostWithNS(boost.Namespace, boost.Name)
}

func (m *managerImpl) GetStartupCPUBoostForPod(pod *corev1.Pod) (*startupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	nsBoosts, ok := m.startupCPUBoosts[pod.Namespace]
	if !ok {
		return nil, false
	}
	for _, boost := range nsBoosts {
		if boost.selector.Matches(labels.Set(pod.Labels)) {
			return boost, true
		}
	}
	return nil, false
}

func (m *managerImpl) GetStartupCPUBoost(namespace string, name string) (*startupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.getStartupCPUBoostWithNS(namespace, name)
}

func (m *managerImpl) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithName("boost-manager")
	t := time.NewTicker(m.checkInterval)
	defer t.Stop()
	log.V(2).Info("Boost manager starting")
	for {
		select {
		case <-t.C:
			log.V(5).Info("tick...")
			m.updateStartupCPUBoostPods(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *managerImpl) addStartupCPUBoostWithNS(boostImpl *startupCPUBoost) {
	nsBoosts, ok := m.startupCPUBoosts[boostImpl.namespace]
	if !ok {
		nsBoosts = make(map[string]*startupCPUBoost)
		m.startupCPUBoosts[boostImpl.namespace] = nsBoosts
	}
	nsBoosts[boostImpl.name] = boostImpl
}

func (m *managerImpl) getStartupCPUBoostWithNS(ns string, name string) (*startupCPUBoost, bool) {
	if nsboosts, ok := m.startupCPUBoosts[ns]; ok {
		boost, ok := nsboosts[name]
		return boost, ok
	}
	return nil, false
}

func (m *managerImpl) deleteStartupCPUBoostWithNS(ns string, name string) {
	if nsBoosts, ok := m.startupCPUBoosts[ns]; ok {
		delete(nsBoosts, name)
	}
}

func (m *managerImpl) getAllStartupCPUBoosts() []*startupCPUBoost {
	result := make([]*startupCPUBoost, 0)
	for _, nsMap := range m.startupCPUBoosts {
		for _, boost := range nsMap {
			result = append(result, boost)
		}
	}
	return result
}

func (m *managerImpl) updateStartupCPUBoostPods(ctx context.Context) {
	m.RLock()
	defer m.RUnlock()
	now := time.Now()
	for _, boost := range m.getAllStartupCPUBoosts() {
		for _, pod := range boost.pods {
			if pod.boostTimestamp.Add(boost.time).Before(now) {
				log := ctrl.LoggerFrom(ctx).WithName("boost-manager").WithValues("pod.boostTimestamp", pod.boostTimestamp,
					"boost.time", boost.time, "time.now", now, "pod", pod.name, "namespace", pod.namespace)
				log.V(2).Info("Reverting startup CPU boost for pod")
				if err := m.podCleanup(ctx, pod); err != nil {
					log.Error(err, "unable to update pod")
				}
				delete(boost.pods, pod.name)
			}
		}
	}
}

func (m *managerImpl) podCleanup(ctx context.Context, pod *startupCPUBoostPod) error {
	podObj := &corev1.Pod{}
	if err := m.client.Get(ctx, types.NamespacedName{Namespace: pod.namespace, Name: pod.name}, podObj); err != nil {
		return err
	}
	for _, container := range podObj.Spec.Containers {
		if request, ok := pod.initCPURequests[container.Name]; ok {
			if reqQuantity, err := apiResource.ParseQuantity(request); err == nil {
				container.Resources.Requests[corev1.ResourceCPU] = reqQuantity
			} else {
				return errors.New("unparsable init CPU request: " + err.Error())
			}
		}
		if limit, ok := pod.initCPULimits[container.Name]; ok {
			if limitQuantity, err := apiResource.ParseQuantity(limit); err == nil {
				container.Resources.Limits[corev1.ResourceCPU] = limitQuantity
			} else {
				return errors.New("unparsable init CPU limit: " + err.Error())
			}
		}
	}
	delete(podObj.Labels, StartupCPUBoostPodLabelKey)
	delete(podObj.Annotations, StartupCPUBoostPodAnnotationKey)
	if err := m.client.Update(ctx, podObj); err != nil {
		return err
	}
	return nil
}
