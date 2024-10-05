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
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"

	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	errStartupCPUBoostAlreadyExists = errors.New("startupCPUBoost already exists")
)

const (
	DefaultManagerCheckInterval = time.Duration(5 * time.Second)
	DefaultMaxGoroutines        = 10
)

type Manager interface {
	// AddStartupCPUBoost registers a new startup-cpu-boost is a manager.
	AddStartupCPUBoost(ctx context.Context, boost StartupCPUBoost) error
	// RemoveStartupCPUBoost removes a startup-cpu-boost from a manager
	RemoveStartupCPUBoost(ctx context.Context, namespace, name string)
	// StartupCPUBoost returns a startup-cpu-boost with a given name and namespace
	StartupCPUBoostForPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, bool)
	// StartupCPUBoostForPod returns a startup-cpu-boost that matches a given pod
	StartupCPUBoost(namespace, name string) (StartupCPUBoost, bool)
	SetStartupCPUBoostReconciler(reconciler reconcile.Reconciler)
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
	reconciler       reconcile.Reconciler
	ticker           TimeTicker
	checkInterval    time.Duration
	startupCPUBoosts map[string]map[string]StartupCPUBoost
	timePolicyBoosts map[boostKey]StartupCPUBoost
	maxGoroutines    int
	log              logr.Logger
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
		maxGoroutines:    DefaultMaxGoroutines,
		log:              ctrl.Log.WithName("boost-manager"),
	}
}

// AddStartupCPUBoost registers a new startup-cpu-boost is a manager.
// If a boost with a given name and namespace already exists, it returns an error.
func (m *managerImpl) AddStartupCPUBoost(ctx context.Context, boost StartupCPUBoost) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.getStartupCPUBoost(boost.Namespace(), boost.Name()); ok {
		return errStartupCPUBoostAlreadyExists
	}
	log := m.log.WithValues("boost", boost.Name(), "namespace", boost.Namespace())
	log.V(5).Info("handling boost registration")
	m.addStartupCPUBoost(boost)
	metrics.NewBoostConfiguration(boost.Namespace())
	log.Info("boost registered successfully")
	return nil
}

// RemoveStartupCPUBoost removes a startup-cpu-boost from a manager if registered.
func (m *managerImpl) RemoveStartupCPUBoost(ctx context.Context, namespace, name string) {
	m.Lock()
	defer m.Unlock()
	log := m.log.WithValues("boost", name, "namespace", namespace)
	log.V(5).Info("handling boost deletion")
	if boosts, ok := m.startupCPUBoosts[namespace]; ok {
		delete(boosts, name)
	}
	key := boostKey{name: name, namespace: namespace}
	delete(m.timePolicyBoosts, key)
	metrics.DeleteBoostConfiguration(namespace)
	log.Info("boost deleted successfully")
}

// StartupCPUBoost returns a startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) StartupCPUBoost(namespace string, name string) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.getStartupCPUBoost(namespace, name)
}

// StartupCPUBoostForPod returns a startup-cpu-boost that matches a given pod if such is registered
// in a manager.
func (m *managerImpl) StartupCPUBoostForPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	m.log.V(5).Info("handling boost pod lookup")
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

func (m *managerImpl) SetStartupCPUBoostReconciler(reconciler reconcile.Reconciler) {
	m.reconciler = reconciler
}

func (m *managerImpl) Start(ctx context.Context) error {
	defer m.ticker.Stop()
	m.log.Info("starting")
	for {
		select {
		case <-m.ticker.Tick():
			m.log.V(5).Info("tick...")
			m.validateTimePolicyBoosts(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

// addStartupCPUBoost registers a new startup-cpu-boost in a manager.
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

// getStartupCPUBoost returns the startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) getStartupCPUBoost(namespace string, name string) (StartupCPUBoost, bool) {
	if boosts, ok := m.startupCPUBoosts[namespace]; ok {
		boost, ok := boosts[name]
		return boost, ok
	}
	return nil, false
}

type podRevertTask struct {
	boost StartupCPUBoost
	pod   *corev1.Pod
}

// validateTimePolicyBoosts validates all time policy boosts in a manager
// and reverts the resources for violated pods.
func (m *managerImpl) validateTimePolicyBoosts(ctx context.Context) {
	m.RLock()
	defer m.RUnlock()
	revertTasks := make(chan *podRevertTask, m.maxGoroutines)
	reconcileTasks := make(chan *reconcile.Request, m.maxGoroutines)
	errors := make(chan error, m.maxGoroutines)

	go func() {
		for _, boost := range m.timePolicyBoosts {
			for _, pod := range boost.ValidatePolicy(ctx, duration.FixedDurationPolicyName) {
				revertTasks <- &podRevertTask{
					boost: boost,
					pod:   pod,
				}
			}
			for _, pod := range boost.ValidatePolicy(ctx, duration.AutoDurationPolicyName) {
				revertTasks <- &podRevertTask{
					boost: boost,
					pod:   pod,
				}
			}
		}
		close(revertTasks)
	}()

	go func() {
		var wg sync.WaitGroup
		for i := 0; i < m.maxGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for task := range revertTasks {
					log := m.log.WithValues("boost", task.boost.Name(), "namespace", task.boost.Namespace(), "pod", task.pod.Name)
					log.V(5).Info("reverting pod resources")
					if err := task.boost.RevertResources(ctx, task.pod); err != nil {
						errors <- fmt.Errorf("pod %s/%s: %w", task.pod.Namespace, task.pod.Name, err)
					} else {
						if autoPolicy, ok := task.boost.DurationPolicies()[duration.AutoDurationPolicyName]; ok {
							log.Info("notifying about pod resource reversion under auto policy")
							if autoPolicy, ok := autoPolicy.(*duration.AutoDurationPolicy); ok {
								autoPolicy.NotifyReversion(task.pod)
							} else {
								log.Info("auto policy not found")
							}
						}

						log.Info("pod resources reverted successfully")
						reconcileTasks <- &reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      task.boost.Name(),
								Namespace: task.boost.Namespace(),
							},
						}
					}
				}
			}()
		}
		wg.Wait()
		close(reconcileTasks)
		close(errors)
	}()

	go func() {
		for err := range errors {
			m.log.Error(err, "pod resources reversion failed")
		}
	}()

	reconcileRequests := dedupeReconcileRequests(reconcileTasks)
	if m.reconciler != nil {
		for _, req := range reconcileRequests {
			m.reconciler.Reconcile(ctx, req)
		}
	}
}

func dedupeReconcileRequests(reconcileTasks chan *reconcile.Request) []reconcile.Request {
	result := make([]reconcile.Request, 0, len(reconcileTasks))
	requests := make(map[reconcile.Request]bool)
	for task := range reconcileTasks {
		requests[*task] = true
	}
	for k := range requests {
		result = append(result, k)
	}
	return result
}
