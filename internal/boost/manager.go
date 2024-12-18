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
	// AddRegularBoost registers a new regular startup-cpu-boost is a manager.
	AddRegularBoost(ctx context.Context, boost NamespacedBoost) error
	// GetRegularBoost returns a regular startup-cpu-boost with a given name and namespace.
	GetRegularBoost(ctx context.Context, name, namespace string) (NamespacedBoost, bool)
	// DeleteRegularBoost de-registers regular startup-cpu-boost from a manager.
	DeleteRegularBoost(ctx context.Context, name, namespace string)
	// AddNamespaceBoost registers a new namespace wide startup-cpu-boost is a manager.
	AddNamespaceBoost(ctx context.Context, boost NamespacedBoost) error
	// GetNamespaceBoost returns a namespace wide startup-cpu-boost with a given name and namespace.
	GetNamespaceBoost(ctx context.Context, name, namespace string) (NamespacedBoost, bool)
	// DeleteNamespaceBoost de-registers namespace wide startup-cpu-boost from a manager.
	DeleteNamespaceBoost(ctx context.Context, name, namespace string)
	// AddClusterBoost registers a new cluster wide startup-cpu-boost is a manager.
	AddClusterBoost(ctx context.Context, boost Boost) error
	// GetClusterBoost returns a cluster wide startup-cpu-boost with a given name and namespace.
	GetClusterBoost(ctx context.Context, name string) (Boost, bool)
	// DeleteClusterBoost de-registers cluster wide startup-cpu-boost from a manager.
	DeleteClusterBoost(ctx context.Context, name string)
	// GetBoostForPod returns a boost that first matches a given pod starting with cluster
	// wide boosts, then namespace wide boosts, then regular boosts.
	GetBoostForPod(ctx context.Context, pod *corev1.Pod) (Boost, bool)
	// SetBoostReconciler sets the given reconciler for a given boost type name.
	SetBoostReconciler(typeName string, reconciler reconcile.Reconciler)
	// Start the boost manager loop for time based boost policies
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
	client          client.Client
	ticker          TimeTicker
	reconcilers     map[string]reconcile.Reconciler
	checkInterval   time.Duration
	regularBoosts   map[string]map[string]NamespacedBoost
	namespaceBoosts map[string]map[string]NamespacedBoost
	clusterBoosts   map[string]Boost
	maxGoroutines   int
	log             logr.Logger
}

func NewManager(client client.Client) Manager {
	return NewManagerWithTicker(client, newTimeTickerImpl(DefaultManagerCheckInterval))
}

func NewManagerWithTicker(client client.Client, ticker TimeTicker) Manager {
	return &managerImpl{
		client:          client,
		ticker:          ticker,
		reconcilers:     make(map[string]reconcile.Reconciler),
		checkInterval:   DefaultManagerCheckInterval,
		regularBoosts:   make(map[string]map[string]NamespacedBoost),
		namespaceBoosts: make(map[string]map[string]NamespacedBoost),
		clusterBoosts:   make(map[string]Boost),
		maxGoroutines:   DefaultMaxGoroutines,
		log:             ctrl.Log.WithName("boost-manager"),
	}
}

// AddRegularBoost registers a new regular startup-cpu-boost is a manager.
// If a boost with a given name and namespace already exists, it returns an error.
func (m *managerImpl) AddRegularBoost(ctx context.Context, boost NamespacedBoost) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.getRegularBoost(boost.Name(), boost.Namespace()); ok {
		return errStartupCPUBoostAlreadyExists
	}
	log := m.log.WithValues("boost", boost.Name(), "namespace", boost.Namespace())
	log.V(5).Info("handling boost registration")
	m.addRegularBoost(boost)
	metrics.NewRegularBoostConfiguration(boost.Namespace())
	log.Info("boost registered successfully")
	return nil
}

// GetRegularBoost returns a regular startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) GetRegularBoost(ctx context.Context, name, namespace string) (NamespacedBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.getRegularBoost(name, namespace)
}

// DeleteRegularBoost de-registers regular startup-cpu-boost from a manager.
func (m *managerImpl) DeleteRegularBoost(ctx context.Context, name, namespace string) {
	m.Lock()
	defer m.Unlock()
	log := m.log.WithValues("boost", name, "namespace", namespace)
	log.V(5).Info("handling boost deletion")
	m.deleteRegularBoost(name, namespace)
	metrics.DeleteRegularBoostConfiguration(namespace)
	log.Info("boost deleted successfully")
}

// AddNamespaceBoost registers a new namespace wide startup-cpu-boost is a manager.
// If a boost with a given name and namespace already exists, it returns an error.
func (m *managerImpl) AddNamespaceBoost(ctx context.Context, boost NamespacedBoost) error {
	panic("unimplemented")
}

// GetNamespaceBoost returns a namespace wide startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) GetNamespaceBoost(ctx context.Context, name, namespace string) (NamespacedBoost, bool) {
	panic("unimplemented")
}

// DeleteNamespaceBoost de-registers namespace wide startup-cpu-boost from a manager.
func (m *managerImpl) DeleteNamespaceBoost(ctx context.Context, name, namespace string) {
	panic("unimplemented")
}

// AddClusterBoost registers a new cluster wide startup-cpu-boost is a manager.
// If a boost with a given name and namespace already exists, it returns an error.
func (m *managerImpl) AddClusterBoost(ctx context.Context, boost Boost) error {
	panic("unimplemented")
}

// GetClusterBoost returns a cluster wide startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) GetClusterBoost(ctx context.Context, name string) (Boost, bool) {
	panic("unimplemented")
}

// DeleteClusterBoost de-registers cluster wide startup-cpu-boost from a manager.
func (m *managerImpl) DeleteClusterBoost(ctx context.Context, name string) {
	panic("unimplemented")
}

// GetBoostForPod returns a boost that first matches a given pod starting with cluster
// wide boosts, then namespace wide boosts, then regular boosts.
func (m *managerImpl) GetBoostForPod(ctx context.Context, pod *corev1.Pod) (Boost, bool) {
	m.RLock()
	defer m.RUnlock()
	m.log.V(5).Info("handling boost pod lookup")
	for _, boost := range m.clusterBoosts {
		if boost.Matches(pod) {
			return boost, true
		}
	}
	for ns := range m.namespaceBoosts {
		for _, boost := range m.namespaceBoosts[ns] {
			if boost.Matches(pod) {
				return boost, true
			}
		}
	}
	for ns := range m.regularBoosts {
		for _, boost := range m.regularBoosts[ns] {
			if boost.Matches(pod) {
				return boost, true
			}
		}
	}
	return nil, false
}

// SetBoostReconciler sets the given reconciler for a given boost type name.
func (m *managerImpl) SetBoostReconciler(typeName string, reconciler reconcile.Reconciler) {
	m.reconcilers[typeName] = reconciler
}

// Start the boost manager loop for time based boost policies
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
func (m *managerImpl) addRegularBoost(boost NamespacedBoost) {
	boosts, ok := m.regularBoosts[boost.Namespace()]
	if !ok {
		boosts = make(map[string]NamespacedBoost)
		m.regularBoosts[boost.Namespace()] = boosts
	}
	boosts[boost.Name()] = boost
}

// getRegularBoost returns the regular startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) getRegularBoost(name, namespace string) (NamespacedBoost, bool) {
	if boosts, ok := m.regularBoosts[namespace]; ok {
		boost, ok := boosts[name]
		return boost, ok
	}
	return nil, false
}

// deleteRegularBoost removes the regular startup-cpu-boost with a given name and namespace
// if registered in a manager.
func (m *managerImpl) deleteRegularBoost(name, namespace string) bool {
	if boosts, ok := m.regularBoosts[namespace]; ok {
		_, ok := boosts[name]
		delete(boosts, name)
		return ok
	}
	return false
}

// getNamespaceBoost returns the namespace wide startup-cpu-boost with a given name and namespace
// if registered in a manager.
// func (m *managerImpl) getNamespaceBoost(name, namespace string) (NamespacedBoost, bool) {
// 	if boosts, ok := m.namespaceBoosts[namespace]; ok {
// 		boost, ok := boosts[name]
// 		return boost, ok
// 	}
// 	return nil, false
// }

// getClusterBoost returns the cluster wide startup-cpu-boost with a given name and namespace
// if registered in a manager.
// func (m *managerImpl) getClusterBoost(name string) (Boost, bool) {
// 	boost, ok := m.clusterBoosts[name]
// 	return boost, ok
// }

// getAllFixedDurationBoosts returns all fixed duration boosts registered in a manager
func (m *managerImpl) getAllFixedDurationBoosts() []Boost {
	totalBoosts := len(m.regularBoosts) + len(m.namespaceBoosts) + len(m.clusterBoosts)
	boosts := make([]Boost, 0, totalBoosts)
	for _, boost := range m.clusterBoosts {
		if _, ok := boost.DurationPolicies()[duration.FixedDurationPolicyName]; ok {
			boosts = append(boosts, boost)
		}
	}
	for ns := range m.namespaceBoosts {
		for _, boost := range m.namespaceBoosts[ns] {
			if _, ok := boost.DurationPolicies()[duration.FixedDurationPolicyName]; ok {
				boosts = append(boosts, boost)
			}
		}
	}
	for ns := range m.regularBoosts {
		for _, boost := range m.regularBoosts[ns] {
			if _, ok := boost.DurationPolicies()[duration.FixedDurationPolicyName]; ok {
				boosts = append(boosts, boost)
			}
		}
	}
	return boosts
}

type podRevertTask struct {
	boost Boost
	pod   *corev1.Pod
}

type reconcileTask struct {
	typeName string
	request  *reconcile.Request
}

// validateTimePolicyBoosts validates all time policy boosts in a manager
// and reverts the resources for violated pods.
func (m *managerImpl) validateTimePolicyBoosts(ctx context.Context) {
	m.RLock()
	defer m.RUnlock()
	revertTasks := make(chan *podRevertTask, m.maxGoroutines)
	reconcileTasks := make(chan *reconcileTask, m.maxGoroutines)
	errors := make(chan error, m.maxGoroutines)

	go func() {
		boosts := m.getAllFixedDurationBoosts()
		for _, boost := range boosts {
			for _, pod := range boost.ValidatePolicy(ctx, duration.FixedDurationPolicyName) {
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
					log := m.log.WithValues("boost", task.boost.Name(), "pod", task.pod.Name)
					if nsBoost, ok := task.boost.(NamespacedBoost); ok {
						log = log.WithValues("namespace", nsBoost.Namespace())
					}
					log.V(5).Info("reverting pod resources")
					if err := task.boost.RevertResources(ctx, task.pod); err != nil {
						errors <- fmt.Errorf("pod %s/%s: %w", task.pod.Namespace, task.pod.Name, err)
					} else {
						log.Info("pod resources reverted successfully")
						namespace := ""
						if nsBoost, ok := task.boost.(NamespacedBoost); ok {
							namespace = nsBoost.Namespace()
						}
						reconcileTasks <- &reconcileTask{
							typeName: task.boost.Type(),
							request: &reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      task.boost.Name(),
									Namespace: namespace,
								},
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

	dedupedReconcileTasks := dedupeReconcileTasks(reconcileTasks)
	for _, task := range dedupedReconcileTasks {
		if reconciler, ok := m.reconcilers[task.typeName]; ok {
			reconciler.Reconcile(ctx, *task.request)
		}
	}
}

func dedupeReconcileTasks(reconcileTasks chan *reconcileTask) []reconcileTask {
	result := make([]reconcileTask, 0, len(reconcileTasks))
	requests := make(map[reconcileTask]bool)
	for task := range reconcileTasks {
		requests[*task] = true
	}
	for k := range requests {
		result = append(result, k)
	}
	return result
}
