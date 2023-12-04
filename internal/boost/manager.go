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

package boost

import (
	"context"
	"errors"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	errStartupCPUBoostAlreadyExists = errors.New("startupCPUBoost already exists")
)

const (
	DefaultManagerCheckInterval = time.Duration(5 * time.Second)
)

type Manager interface {
	AddStartupCPUBoost(ctx context.Context, boost StartupCPUBoost) error
	RemoveStartupCPUBoost(ctx context.Context, namespace, name string)
	StartupCPUBoostForPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, bool)
	StartupCPUBoost(namespace, name string) (StartupCPUBoost, bool)
	Start(ctx context.Context) error
}

type TimeTicker interface {
	Tick() <-chan time.Time
	Stop()
}

type timeTickerImpl struct {
	t time.Ticker
}

func (t *timeTickerImpl) Tick() <-chan time.Time {
	return t.t.C
}

func (t *timeTickerImpl) Stop() {
	t.t.Stop()
}

func newTimeTickerImpl(d time.Duration) TimeTicker {
	return &timeTickerImpl{
		t: *time.NewTicker(d),
	}
}

type managerImpl struct {
	sync.RWMutex
	client           client.Client
	ticker           TimeTicker
	checkInterval    time.Duration
	startupCPUBoosts map[string]map[string]StartupCPUBoost
	timePolicyBoosts map[boostKey]StartupCPUBoost
}

type boostKey struct {
	name      string
	namespace string
}

func NewManager(client client.Client) Manager {
	return NewManagerWithTicker(client, newTimeTickerImpl(DefaultManagerCheckInterval))
}

func NewManagerWithTicker(client client.Client, ticker TimeTicker) Manager {
	return &managerImpl{
		client:           client,
		ticker:           ticker,
		checkInterval:    DefaultManagerCheckInterval,
		startupCPUBoosts: make(map[string]map[string]StartupCPUBoost),
		timePolicyBoosts: make(map[boostKey]StartupCPUBoost),
	}
}

func (m *managerImpl) AddStartupCPUBoost(ctx context.Context, boost StartupCPUBoost) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.getStartupCPUBoost(boost.Namespace(), boost.Name()); ok {
		return errStartupCPUBoostAlreadyExists
	}
	log := m.loggerFromContext(ctx).WithValues("boost", boost.Name, "namespace", boost.Namespace)
	log.V(5).Info("handling startup-cpu-boost create")
	m.addStartupCPUBoost(boost)
	return nil
}

func (m *managerImpl) RemoveStartupCPUBoost(ctx context.Context, namespace, name string) {
	m.Lock()
	defer m.Unlock()
	log := m.loggerFromContext(ctx).WithValues("boost", name, "namespace", namespace)
	log.V(5).Info("handling startup-cpu-boost delete")
	if boosts, ok := m.startupCPUBoosts[namespace]; ok {
		delete(boosts, name)
	}
	key := boostKey{name: name, namespace: namespace}
	delete(m.timePolicyBoosts, key)
}

func (m *managerImpl) StartupCPUBoost(namespace string, name string) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.getStartupCPUBoost(namespace, name)
}

func (m *managerImpl) StartupCPUBoostForPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	log := m.loggerFromContext(ctx).WithValues("pod", pod.Name, "namespace", pod.Namespace)
	log.V(5).Info("handling startup-cpu-boost pod lookup")
	nsBoosts, ok := m.startupCPUBoosts[pod.Namespace]
	if !ok {
		return nil, false
	}
	for _, boost := range nsBoosts {
		if boost.Matches(pod) {
			return boost, true
		}
	}
	return nil, false
}

func (m *managerImpl) Start(ctx context.Context) error {
	log := m.loggerFromContext(ctx)
	//t := time.NewTicker(m.checkInterval)
	//defer t.Stop()
	defer m.ticker.Stop()
	log.V(2).Info("Starting")
	for {
		select {
		case <-m.ticker.Tick():
			log.V(5).Info("tick...")
			m.updateTimePolicyBoosts(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *managerImpl) addStartupCPUBoost(boost StartupCPUBoost) {
	boosts, ok := m.startupCPUBoosts[boost.Namespace()]
	if !ok {
		boosts = make(map[string]StartupCPUBoost)
		m.startupCPUBoosts[boost.Namespace()] = boosts
	}
	boosts[boost.Name()] = boost
	if _, ok := boost.DurationPolicies()[duration.FixedDurationPolicyName]; ok {
		key := boostKey{name: boost.Name(), namespace: boost.Namespace()}
		m.timePolicyBoosts[key] = boost
	}
}

func (m *managerImpl) getStartupCPUBoost(namespace string, name string) (StartupCPUBoost, bool) {
	if boosts, ok := m.startupCPUBoosts[namespace]; ok {
		boost, ok := boosts[name]
		return boost, ok
	}
	return nil, false
}

func (m *managerImpl) updateTimePolicyBoosts(ctx context.Context) {
	m.RLock()
	defer m.RUnlock()
	log := m.loggerFromContext(ctx)
	for _, boost := range m.timePolicyBoosts {
		for _, pod := range boost.ValidatePolicy(ctx, duration.FixedDurationPolicyName) {
			log = log.WithValues("boost", boost.Name(), "namespace", boost.Namespace(), "pod", pod.Name)
			log.V(5).Info("updating pod with initial resources")
			if err := boost.RevertResources(ctx, pod); err != nil {
				log.Error(err, "failed to revert resources for pod")
			}
		}
	}
}

func (m *managerImpl) loggerFromContext(ctx context.Context) logr.Logger {
	return ctrl.LoggerFrom(ctx).
		WithName("boost-manager")
}
