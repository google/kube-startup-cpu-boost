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

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
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
	// AddRegularCPUBoost registers new regular startup cpu boost in a manager.
	AddRegularCPUBoost(ctx context.Context, boost StartupCPUBoost) error

	// DeleteRegularCPUBoost deletes a regular startup cpu boost from a manager.
	DeleteRegularCPUBoost(ctx context.Context, name, namespace string)

	// UpdateRegularCPUBoost updates a regular startup cpu boost in a manager.
	UpdateRegularCPUBoost(ctx context.Context, spec *autoscaling.StartupCPUBoost) error

	// GetRegularCPUBoost returns a regular startup cpu boost with a given name and namespace
	// if such is registered in a manager.
	GetRegularCPUBoost(ctx context.Context, name, namespace string) (StartupCPUBoost, bool)

	// GetCPUBoostForPod returns a startup cpu boost that matches a given pod if such is registered
	// in a manager. If multiple boost types matches, the most specific is returned.
	GetCPUBoostForPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, bool)

	// UpsertPod adds new or updates existing tracked POD to the manager and boosts.
	// If found, the matching cpu boost is returned.
	UpsertPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, error)

	// DeletePod deletes the tracked POD from the manager and boosts.
	// If found, the matching cpu boost is returned.
	DeletePod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, error)

	// SetStartupCPUBoostReconciler sets the boost object reconciler for the manager.
	SetStartupCPUBoostReconciler(reconciler reconcile.Reconciler)

	// Start starts the manager time based check loop.
	Start(ctx context.Context) error

	// IsRunning returns true if manager has started its time based check loop.
	IsRunning(ctx context.Context) bool
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

type podRevertTask struct {
	boost StartupCPUBoost
	pod   *corev1.Pod
}

type managerImpl struct {
	sync.RWMutex
	isRunning     bool
	client        client.Client
	reconciler    reconcile.Reconciler
	ticker        TimeTicker
	checkInterval time.Duration
	maxGoroutines int
	log           logr.Logger

	// timedBoosts is a collection of boosts of any kind that have time duration policy set
	timedBoosts namespacedObjects[StartupCPUBoost]
	// regularBoost is collection of a regular, namespaced boosts
	regularBoosts namespacedObjects[StartupCPUBoost]
	// orphanedPods is a collection of tracked pods that have no matching boost registered
	orphanedPods namespacedObjects[*corev1.Pod]
}

func NewManager(client client.Client) Manager {
	return NewManagerWithTicker(client, newTimeTickerImpl(DefaultManagerCheckInterval))
}

func NewManagerWithTicker(client client.Client, ticker TimeTicker) Manager {
	return &managerImpl{
		client:        client,
		ticker:        ticker,
		checkInterval: DefaultManagerCheckInterval,
		timedBoosts:   *newNamespacedObjects[StartupCPUBoost](),
		regularBoosts: *newNamespacedObjects[StartupCPUBoost](),
		orphanedPods:  *newNamespacedObjects[*corev1.Pod](),
		maxGoroutines: DefaultMaxGoroutines,
		log:           ctrl.Log.WithName("boost-manager"),
	}
}

// AddRegularCPUBoost registers new regular startup cpu boost in a manager.
// Returns an error if a boost is already registered.
func (m *managerImpl) AddRegularCPUBoost(ctx context.Context, boost StartupCPUBoost) error {
	m.Lock()
	defer m.Unlock()
	log := m.log.WithValues("boost", boost.Name(), "namespace", boost.Namespace())
	if _, ok := m.regularBoosts.Get(boost.Name(), boost.Namespace()); ok {
		return errStartupCPUBoostAlreadyExists
	}
	defer log.Info("regular boost registered successfully")
	defer m.postProcessNewBoost(ctx, boost)
	log.V(5).Info("handling regular boost registration")
	m.regularBoosts.Put(boost.Name(), boost.Namespace(), boost)
	metrics.NewBoostConfiguration(boost.Namespace())
	return nil
}

// DeleteRegularCPUBoost deletes a regular startup cpu boost from a manager.
func (m *managerImpl) DeleteRegularCPUBoost(ctx context.Context, namespace, name string) {
	m.Lock()
	defer m.Unlock()
	log := m.log.WithValues("boost", name, "namespace", namespace)
	log.V(5).Info("handling regular boost deletion")
	defer log.Info("boost deleted successfully")
	m.regularBoosts.Delete(name, namespace)
	m.timedBoosts.Delete(name, namespace)
	metrics.DeleteBoostConfiguration(namespace)
}

// UpdateRegularCPUBoost updates a regular startup cpu boost in a manager.
func (m *managerImpl) UpdateRegularCPUBoost(ctx context.Context,
	spec *autoscaling.StartupCPUBoost) error {
	m.Lock()
	defer m.Unlock()
	log := m.log.WithValues("boost", spec.ObjectMeta.Name, "namespace", spec.ObjectMeta.Namespace)
	log.V(5).Info("handling boost update")
	defer log.Info("boost updated successfully")
	boost, ok := m.regularBoosts.Get(spec.ObjectMeta.Name, spec.ObjectMeta.Namespace)
	if !ok {
		log.V(5).Info("boost not found")
		return nil
	}
	return boost.UpdateFromSpec(ctx, spec)
}

// GetRegularCPUBoost returns a regular startup cpu boost with a given name and namespace
// if such is registered in a manager.
func (m *managerImpl) GetRegularCPUBoost(ctx context.Context, name string,
	namespace string) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.regularBoosts.Get(name, namespace)
}

// GetCPUBoostForPod returns a startup cpu boost that matches a given pod if such is registered
// in a manager. If multiple boost types matches, the most specific is returned.
func (m *managerImpl) GetCPUBoostForPod(ctx context.Context,
	pod *corev1.Pod) (StartupCPUBoost, bool) {
	m.RLock()
	defer m.RUnlock()
	return m.getMatchingBoost(pod)
}

// UpsertPod adds new or updates existing tracked POD to the manager and boosts.
// If found, the matching cpu boost is returned.
func (m *managerImpl) UpsertPod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, error) {
	m.Lock()
	defer m.Unlock()
	m.log.V(5).Info("handling pod upsert")
	if boost, ok := m.getMatchingBoost(pod); ok {
		err := boost.UpsertPod(ctx, pod)
		if err == nil {
			m.orphanedPods.Delete(pod.Name, pod.Namespace)
		}
		return boost, err
	}
	m.log.V(5).Info("boost not found, registering orphaned pod")
	m.orphanedPods.Put(pod.Name, pod.Namespace, pod)
	return nil, nil
}

// DeletePod deletes the tracked POD from the manager and boosts.
// If found, the matching cpu boost is returned.
func (m *managerImpl) DeletePod(ctx context.Context, pod *corev1.Pod) (StartupCPUBoost, error) {
	m.Lock()
	defer m.Unlock()
	m.log.V(5).Info("handling pod delete")
	if boost, ok := m.getMatchingBoost(pod); ok {
		return boost, boost.DeletePod(ctx, pod)
	}
	m.log.V(5).Info("boost not found, removing orphaned pod if exists")
	m.orphanedPods.Delete(pod.Name, pod.Namespace)
	return nil, nil
}

// SetStartupCPUBoostReconciler sets the boost object reconciler for the manager.
func (m *managerImpl) SetStartupCPUBoostReconciler(reconciler reconcile.Reconciler) {
	m.reconciler = reconciler
}

// Start starts the manager time based check loop.
func (m *managerImpl) Start(ctx context.Context) error {
	defer m.ticker.Stop()
	defer m.setRunning(false)
	m.log.Info("starting")
	m.setRunning(true)
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

func (m *managerImpl) IsRunning(ctx context.Context) bool {
	return m.isRunning
}

// PRIVATE FUNCS START below

func (m *managerImpl) setRunning(isRunning bool) {
	m.isRunning = isRunning
}

// getMatchingBoost finds the most specific matching boost for a given pod.
func (m *managerImpl) getMatchingBoost(pod *corev1.Pod) (StartupCPUBoost, bool) {
	namespaceBoosts := m.regularBoosts.List(pod.Namespace)
	for _, boost := range namespaceBoosts {
		if boost.Matches(pod) {
			return boost, true
		}
	}
	return nil, false
}

// postProcessNewBoost performs additional post processing of a newly registered boost
func (m *managerImpl) postProcessNewBoost(ctx context.Context, boost StartupCPUBoost) {
	log := m.log.WithValues("boost", boost.Name(), "namespace", boost.Namespace())
	if _, ok := boost.DurationPolicies()[duration.FixedDurationPolicyName]; ok {
		log.V(5).Info("adding boost to timedBoosts collection")
		m.timedBoosts.Put(boost.Name(), boost.Namespace(), boost)
	}
	if err := m.mapOrphanedPods(ctx, boost); err != nil {
		log.Error(err, "failed to map orphaned pods")
	}
}

// mapOrphanedPods maps orphaned pods to the given boost if they match.
// Matched pods are registered in a boost and removed from orphanedPods collection.
func (m *managerImpl) mapOrphanedPods(ctx context.Context, boost StartupCPUBoost) error {
	log := m.log.WithValues("boost", boost.Name(), "namespace", boost.Namespace())
	errs := make([]error, 0)
	namespaceOrphanedPods := m.orphanedPods.List(boost.Namespace())
	mappedOrphanedPods := make([]*corev1.Pod, 0, len(namespaceOrphanedPods))
	for _, orphanedPod := range namespaceOrphanedPods {
		if boost.Matches(orphanedPod) {
			log := log.WithValues("pod", orphanedPod.Name)
			log.V(5).Info("matched orphaned pod")
			if err := boost.UpsertPod(ctx, orphanedPod); err != nil {
				errs = append(errs, err)
			} else {
				mappedOrphanedPods = append(mappedOrphanedPods, orphanedPod)
			}
		}
	}
	for _, orphanedPod := range mappedOrphanedPods {
		m.orphanedPods.Delete(orphanedPod.Name, orphanedPod.Namespace)
	}
	return errors.Join(errs...)
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
		for _, boost := range m.timedBoosts.ListAll() {
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
					log := m.log.WithValues("boost", task.boost.Name(), "namespace", task.boost.Namespace(), "pod", task.pod.Name)
					log.V(5).Info("reverting pod resources")
					if err := task.boost.RevertResources(ctx, task.pod); err != nil {
						errors <- fmt.Errorf("pod %s/%s: %w", task.pod.Namespace, task.pod.Name, err)
					} else {
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
