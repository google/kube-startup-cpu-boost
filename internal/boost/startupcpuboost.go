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

	"github.com/go-logr/logr"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StartupCPUBoost is an implementation of a StartupCPUBoost CRD
type StartupCPUBoost interface {
	// Name returns startup-cpu-boost name
	Name() string
	// Namespace returns startup-cpu-boost namespace
	Namespace() string
	// ApplyResourcePolicy applies resource policy on a given POD and returns the boosted container names
	ApplyResourcePolicy(ctx context.Context, pod *corev1.Pod, containerNames bpod.ContainerNameSet) ([]string, error)
	// DurationPolicies returns configured duration policies
	DurationPolicies() map[string]duration.Policy
	// Pod returns a POD if tracked by startup-cpu-boost
	Pod(name string) (*corev1.Pod, bool)
	// HandlePodEvent handles the POD event.
	HandlePodEvent(ctx context.Context, event *bpod.PodEvent) error
	// ValidatePolicy validates policy with a given name on all startup-cpu-boost PODs.
	ValidatePolicy(ctx context.Context, name string) []*corev1.Pod
	// RevertResources updates POD's container resource requests and limits to their original
	// values using the data from StartupCPUBoost annotation
	RevertResources(ctx context.Context, pod *corev1.Pod) error
	// Matches verifies if a boost selector matches the given POD
	Matches(pod *corev1.Pod) bool
	// Stats returns the StartupCPUBoost usage statistics
	Stats() StartupCPUBoostStats
	// UpdateFromSpec updates the StartupCPUBoost from the API spec
	UpdateFromSpec(ctx context.Context, boost *autoscaling.StartupCPUBoost) error
}

const (
	StartupCPUBoostStatsPodCreateEvent             = 1
	StartupCPUBoostStatsPodUpdateEvent             = 2
	StartupCPUBoostStatsPodDeleteEvent             = 3
	StartupCPUBoostStatsContainerRestartBoostEvent = 4
	ResizeSubResourceName                          = "resize"
)

var (
	ErrNilBoost  = errors.New("boost spec cannot be nil")
	ErrNilConfig = errors.New("config cannot be nil")
	ErrNilClient = errors.New("k8s client cannot be nil")
)

type StartupCPUBoostStatsEventType int32

type StartupCPUBoostStatsEvent struct {
	Type                   StartupCPUBoostStatsEventType
	Object                 any
	BoostedContainersCount int
}

// StartupCPUBoostStats holds the StartupCPUBoost usage statistics
type StartupCPUBoostStats struct {
	// activeContainerBoosts is a number of a containers which CPU resources
	// were increased (boosted) and not yet reverted to their original values
	ActiveContainerBoosts int
	// totalContainerBoosts is a number of a containers which CPU resources
	// were increased (boosted)
	TotalContainerBoosts int
}

type containerPolicyEntry struct {
	matcher resource.ContainerMatcher
	policy  resource.ContainerPolicy
}

// StartupCPUBoostImpl is an implementation of a StartupCPUBoost CRD
type StartupCPUBoostImpl struct {
	sync.RWMutex
	name                     string
	namespace                string
	selector                 labels.Selector
	durationPolicies         map[string]duration.Policy
	resourcePolicies         []containerPolicyEntry
	pods                     map[string]*corev1.Pod
	client                   client.Client
	stats                    StartupCPUBoostStats
	legacyRevertMode         bool
	boostOnRestartEnabled    bool
	podLevelResourcesEnabled bool
	removeLimitsEnabled      bool
}

// StartupCPUBoostConfig holds configuration for a boost
type StartupCPUBoostConfig struct {
	// Client is a k8s client
	Client client.Client
	// LegacyRevertMode controls if pre k8s resource reversion mode should be used
	LegacyRevertMode bool
	// BoostOnRestart controls if POD resources should be boosted on container restarts
	BoostOnRestart bool
	// PodLevelResourcesEnabled controls if pod level resources should be used
	PodLevelResourcesEnabled bool
	// RemoveLimitsEnabled controls if cpu limits should be removed when boosting
	RemoveLimitsEnabled bool
}

// Validate validates the configuration
func (c *StartupCPUBoostConfig) Validate() error {
	if c == nil {
		return ErrNilConfig
	}
	if c.Client == nil {
		return ErrNilClient
	}
	return nil
}

// NewStartupCPUBoost constructs startup-cpu-boost implementation from a given API spec
func NewStartupCPUBoost(boost *autoscaling.StartupCPUBoost, cfg *StartupCPUBoostConfig) (StartupCPUBoost, error) {
	if boost == nil {
		return nil, ErrNilBoost
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&boost.Selector)
	if err != nil {
		return nil, err
	}
	resourcePolicies, err := mapResourcePolicies(boost.Spec.ResourcePolicy)
	if err != nil {
		return nil, err
	}
	return &StartupCPUBoostImpl{
		name:                     boost.Name,
		namespace:                boost.Namespace,
		selector:                 selector,
		durationPolicies:         mapDurationPolicy(boost.Spec.DurationPolicy),
		resourcePolicies:         resourcePolicies,
		pods:                     make(map[string]*corev1.Pod),
		client:                   cfg.Client,
		stats:                    StartupCPUBoostStats{},
		legacyRevertMode:         cfg.LegacyRevertMode,
		boostOnRestartEnabled:    cfg.BoostOnRestart,
		podLevelResourcesEnabled: cfg.PodLevelResourcesEnabled,
		removeLimitsEnabled:      cfg.RemoveLimitsEnabled,
	}, nil
}

// Name returns startup-cpu-boost name
func (b *StartupCPUBoostImpl) Name() string {
	return b.name
}

// Namespace returns startup-cpu-boost namespace
func (b *StartupCPUBoostImpl) Namespace() string {
	return b.namespace
}

// ApplyResourcePolicy applies resource policy on a given POD
func (b *StartupCPUBoostImpl) ApplyResourcePolicy(ctx context.Context, pod *corev1.Pod, containerNames bpod.ContainerNameSet) ([]string, error) {
	log := b.loggerFromContext(ctx)
	originalQosClass := bpod.ComputePodQOS(pod, b.podLevelResourcesEnabled)
	annotation := bpod.NewBoostAnnotation()
	if _, ok := pod.Annotations[bpod.BoostAnnotationKey]; ok {
		var err error
		annotation, err = bpod.BoostAnnotationFromPod(pod)
		if err != nil {
			return nil, err
		}
		if annotation.State != bpod.BoostStateReverted {
			log.WithValues("state", annotation.State).Info("skipping boost as it's not in reverted state")
			return nil, nil
		}
	}
	var boostedContainers []string
	for i, container := range pod.Spec.Containers {
		if containerNames != nil && !containerNames.Has(container.Name) {
			continue
		}
		policy, found := b.resourcePolicy(ctx, &container)
		if !found {
			continue
		}
		log = log.WithValues("container", container.Name,
			"cpuRequests", container.Resources.Requests.Cpu().String(),
			"cpuLimits", container.Resources.Limits.Cpu().String(),
		)
		if bpod.ResourceResizeRequiresRestart(container, corev1.ResourceCPU) {
			log.Info("skipping container due to restart policy")
			continue
		}
		if !bpod.HasCPUResourcesToIncrease(container) {
			log.Info("skipping container due to lack of CPU resources to increase")
			continue
		}
		resources := policy.NewResources(ctx, &container)
		tmpUpdatedPod := pod.DeepCopy()
		tmpUpdatedPod.Spec.Containers[i].Resources = *resources
		tmpNewQosClass := bpod.ComputePodQOS(tmpUpdatedPod, b.podLevelResourcesEnabled)
		if tmpNewQosClass != originalQosClass {
			log.Info("skipping container due to QOS class change after boost")
			continue
		}
		if !resources.Requests.Cpu().IsZero() {
			log = log.WithValues(
				"newCpuRequests", resources.Requests.Cpu().String(),
			)
		}
		if !resources.Limits.Cpu().IsZero() {
			if b.canRemoveLimit(pod, originalQosClass, log) {
				delete(resources.Limits, corev1.ResourceCPU)
				log = log.WithValues("newCpuLimits", "<removed>")
			} else {
				log = log.WithValues("newCpuLimits", resources.Limits.Cpu().String())
			}
		}
		annotation.UpdateInitResources(container.Name, container.Resources)
		pod.Spec.Containers[i].Resources = *resources
		boostedContainers = append(boostedContainers, container.Name)
		log.Info("calculated increased container resources")
	}
	// checks if any container CPU resources were boosted
	if len(boostedContainers) > 0 {
		annotation.State = bpod.BoostStateActive
		annotation.BoostTimestamp = time.Now()
		annotation.Apply(pod)
		label := &bpod.BoostPodLabel{BoostName: b.Name()}
		label.Apply(pod)
		return boostedContainers, nil
	}
	return nil, nil
}

// DurationPolicies returns configured duration policies
func (b *StartupCPUBoostImpl) DurationPolicies() map[string]duration.Policy {
	return b.durationPolicies
}

// Pod returns a POD if tracked by startup-cpu-boost.
func (b *StartupCPUBoostImpl) Pod(name string) (*corev1.Pod, bool) {
	b.RLock()
	defer b.RUnlock()
	pod, ok := b.pods[name]
	return pod, ok
}

// HandlePodEvent handles the POD event.
func (b *StartupCPUBoostImpl) HandlePodEvent(ctx context.Context, event *bpod.PodEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	switch event.Type {
	case bpod.PodEventTypePodCreated:
		existing := b.upsertPod(ctx, event.Pod)
		statsType := StartupCPUBoostStatsEventType(StartupCPUBoostStatsPodCreateEvent)
		if existing {
			statsType = StartupCPUBoostStatsPodUpdateEvent
		}
		b.updateStats(StartupCPUBoostStatsEvent{Type: statsType, Object: event.Pod})
		return b.validateDurationPolicies(ctx, event.Pod)
	case bpod.PodEventTypePodDeleted:
		return b.deletePod(ctx, event.Pod)
	case bpod.PodEventTypeConditionChanged:
		b.inspectResizeConditions(ctx, event.Pod)
		b.upsertPod(ctx, event.Pod)
		b.updateStats(StartupCPUBoostStatsEvent{Type: StartupCPUBoostStatsPodUpdateEvent, Object: event.Pod})
		return b.validateDurationPolicies(ctx, event.Pod)
	case bpod.PodEventTypeContainerRestarting:
		return b.boostOnRestart(ctx, event.Pod, event.RestartingContainerNames)
	default:
		log := b.loggerFromContext(ctx).WithValues("event_type", event.Type, "pod", event.Pod.Name)
		log.Info("unknown event type, skipping")
		return nil
	}
}

// ValidatePolicy validates policy with a given name on all startup-cpu-boost PODs.
// The function returns slice of PODs that violated the policy.
func (b *StartupCPUBoostImpl) ValidatePolicy(ctx context.Context, name string) (violated []*corev1.Pod) {
	b.RLock()
	defer b.RUnlock()
	violated = make([]*corev1.Pod, 0)
	policy, ok := b.durationPolicies[name]
	if !ok {
		return
	}
	for _, pod := range b.pods {
		if !b.validatePolicyOnPod(ctx, policy, pod) {
			violated = append(violated, pod)
		}
	}
	return
}

// RevertResources updates POD's container resource requests and limits to their original
// values using the data from StartupCPUBoost annotation
func (b *StartupCPUBoostImpl) RevertResources(ctx context.Context, pod *corev1.Pod) error {
	return b.revertResources(ctx, pod)
}

// Matches verifies if a boost selector matches the given POD
func (b *StartupCPUBoostImpl) Matches(pod *corev1.Pod) bool {
	return b.selector.Matches(labels.Set(pod.Labels))
}

// Stats returns the StartupCPUBoost usage statistics
func (b *StartupCPUBoostImpl) Stats() StartupCPUBoostStats {
	b.RLock()
	defer b.RUnlock()
	return b.stats
}

// UpdateFromSpec updates the StartupCPUBoost from the API spec
func (b *StartupCPUBoostImpl) UpdateFromSpec(ctx context.Context, boost *autoscaling.StartupCPUBoost) error {
	b.Lock()
	defer b.Unlock()
	log := b.loggerFromContext(ctx)
	log.V(5).Info("handling boost update from API spec")
	selector, err := metav1.LabelSelectorAsSelector(&boost.Selector)
	if err != nil {
		return err
	}
	resourcePolicies, err := mapResourcePolicies(boost.Spec.ResourcePolicy)
	if err != nil {
		return err
	}
	b.selector = selector
	b.resourcePolicies = resourcePolicies
	b.durationPolicies = mapDurationPolicy(boost.Spec.DurationPolicy)
	return nil
}

// resourcePolicy returns the resource policy for a given container
func (b *StartupCPUBoostImpl) resourcePolicy(ctx context.Context, container *corev1.Container) (resource.ContainerPolicy, bool) {
	b.RLock()
	defer b.RUnlock()
	for _, entry := range b.resourcePolicies {
		if entry.matcher != nil && entry.matcher.Matches(ctx, container) {
			return entry.policy, true
		}
	}
	return nil, false
}

// upsertPod inserts new or updates existing POD in startup-cpu-boost tracking.
// It returns true if the pod already existed in tracking, or false if it was newly inserted.
func (b *StartupCPUBoostImpl) upsertPod(ctx context.Context, pod *corev1.Pod) (existing bool) {
	b.Lock()
	defer b.Unlock()
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("handling pod upsert")
	_, existing = b.pods[pod.Name]
	b.pods[pod.Name] = pod
	log.V(5).Info("pod upserted successfully")
	return existing
}

func (b *StartupCPUBoostImpl) validateDurationPolicies(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	b.RLock()
	condPolicy, ok := b.durationPolicies[duration.PodConditionPolicyName]
	b.RUnlock()
	if !ok {
		log.V(5).Info("pod duration policy not found, skipping resource reversion")
		return nil
	}
	if valid := b.validatePolicyOnPod(ctx, condPolicy, pod); !valid {
		log.V(5).Info("reverting pod resources")
		if err := b.RevertResources(ctx, pod); err != nil {
			return fmt.Errorf("pod resources reversion failed: %s", err)
		}
		log.Info("pod resources reverted successfully")
	}
	return nil
}

// deletePod removes the POD from the startup-cpu-boost tracking
func (b *StartupCPUBoostImpl) deletePod(ctx context.Context, pod *corev1.Pod) error {
	b.Lock()
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("handling pod delete")
	delete(b.pods, pod.Name)
	b.Unlock()
	b.updateStats(StartupCPUBoostStatsEvent{Type: StartupCPUBoostStatsPodDeleteEvent, Object: pod})
	return nil
}

func (b *StartupCPUBoostImpl) boostOnRestart(ctx context.Context, pod *corev1.Pod, containerNames []string) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name, "container_names", containerNames)
	if !b.boostOnRestartEnabled {
		log.V(5).Info("boost on restart not enabled, skipping")
		return nil
	}
	log.Info("boosting pod resources on container restart")
	boostedPod := pod.DeepCopy()
	boostedContainers, err := b.ApplyResourcePolicy(ctx, boostedPod, bpod.ContainerNameSetFromSlice(containerNames))
	if err != nil {
		return err
	}
	if len(boostedContainers) == 0 {
		return nil
	}
	if err := b.updatePod(ctx, pod, boostedPod); err != nil {
		return err
	}
	b.upsertPod(ctx, boostedPod)
	b.updateStats(StartupCPUBoostStatsEvent{
		Type:                   StartupCPUBoostStatsContainerRestartBoostEvent,
		Object:                 boostedPod,
		BoostedContainersCount: len(boostedContainers),
	})
	log.Info("pod resources boosted on restart successfully")
	return nil
}

// loggerFromContext provides Logger from a current context with configured
// values common for startup-cpu-boost like name or namespace
func (b *StartupCPUBoostImpl) loggerFromContext(ctx context.Context) logr.Logger {
	return ctrl.LoggerFrom(ctx).
		WithName("boost").
		WithValues(
			"name", b.name,
			"namespace", b.namespace,
		)
}

// validatePolicyOnPod validates given policy on a given POD.
// The function returns true if policy is valid or false otherwise
func (b *StartupCPUBoostImpl) validatePolicyOnPod(ctx context.Context, p duration.Policy, pod *corev1.Pod) (valid bool) {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	annot, err := bpod.BoostAnnotationFromPod(pod)
	if err != nil || annot.State != bpod.BoostStateActive {
		return true
	}
	if valid = p.Valid(pod); !valid {
		log.WithValues("policy", p.Name()).V(5).Info("policy is not valid")
	}
	return
}

// revertResources updates POD's container resource requests and limits to their original
// values using the data from StartupCPUBoost annotation
func (b *StartupCPUBoostImpl) revertResources(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx)
	originalPod := pod.DeepCopy()
	if b.boostOnRestartEnabled {
		log.Info("reverting pod resources (boost on restart enabled)")
		bpod.RevertResourceBoostWithBoostOnRestart(pod)
	} else {
		log.Info("reverting pod resources")
		bpod.RevertResourceBoost(pod)
	}
	if err := b.updatePod(ctx, originalPod, pod); err != nil {
		return err
	}
	if b.boostOnRestartEnabled {
		b.upsertPod(ctx, pod)
		b.updateStats(StartupCPUBoostStatsEvent{Type: StartupCPUBoostStatsPodUpdateEvent, Object: pod})
		return nil
	}
	return b.deletePod(ctx, pod)
}

func (b *StartupCPUBoostImpl) updatePod(ctx context.Context, originalPod, updatedPod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", updatedPod.Name)
	if b.legacyRevertMode {
		log.V(5).Info("updating POD using update only (legacy)")
		if err := b.client.Update(ctx, updatedPod); err != nil {
			return fmt.Errorf("failed to update pod spec: %s", err)
		}
		return nil
	}
	log.V(5).Info("updating POD")
	resourcePatch := bpod.NewApplyBoostResourcesPatch(originalPod, updatedPod)
	metadataPatch := bpod.NewApplyBoostMetadataPatch(originalPod, updatedPod)

	if err := b.client.SubResource(ResizeSubResourceName).Patch(ctx, updatedPod, resourcePatch); err != nil {
		return fmt.Errorf("failed to patch pod resize: %s", err)
	}
	if err := b.client.Patch(ctx, updatedPod, metadataPatch); err != nil {
		return fmt.Errorf("failed to patch pod metadata: %s", err)
	}
	return nil
}

// updateStats updates the StartupCPUBoost usage statistics based on the
// received update event
func (b *StartupCPUBoostImpl) updateStats(e StartupCPUBoostStatsEvent) {
	b.Lock()
	defer b.Unlock()
	var activeCnt int
	for _, pod := range b.pods {
		activeCnt += boostContainersLen(pod)
	}
	b.stats.ActiveContainerBoosts = activeCnt
	metrics.SetBoostContainersActive(b.namespace, b.name, float64(activeCnt))
	switch e.Type {
	case StartupCPUBoostStatsPodCreateEvent:
		pod := e.Object.(*corev1.Pod)
		boostContainersLen := boostContainersLen(pod)
		b.stats.TotalContainerBoosts += boostContainersLen
		metrics.AddBoostContainersTotal(b.namespace, b.name, float64(boostContainersLen))
	case StartupCPUBoostStatsContainerRestartBoostEvent:
		b.stats.TotalContainerBoosts += e.BoostedContainersCount
		metrics.AddBoostContainersTotal(b.namespace, b.name, float64(e.BoostedContainersCount))
	}
}

func (b *StartupCPUBoostImpl) canRemoveLimit(pod *corev1.Pod, qosClass corev1.PodQOSClass, log logr.Logger) bool {
	if !b.removeLimitsEnabled {
		return false
	}
	if pod.UID != "" {
		log.V(5).Info("skipping CPU limits removal as pod is already created and in-place resize forbids removing limits")
		return false
	}
	if qosClass != corev1.PodQOSBurstable && qosClass != corev1.PodQOSBestEffort {
		log.Info("skipping CPU limits removal as pod is not burstable nor besteffort")
		return false
	}
	return true
}

// boostContainersLen returns the number of containers that were boosted
// by StartupCPUBoost in a given Pod
func boostContainersLen(pod *corev1.Pod) (cnt int) {
	if annot, err := bpod.BoostAnnotationFromPod(pod); err == nil && annot.State == bpod.BoostStateActive {
		return len(annot.InitCPURequests)
	}
	return
}

// mapDurationPolicy maps the Duration Policy from the API spec to the map of policy
// implementations with policy name keys
func mapDurationPolicy(policiesSpec autoscaling.DurationPolicy) map[string]duration.Policy {
	policies := make(map[string]duration.Policy)
	if fixedPolicy := policiesSpec.Fixed; fixedPolicy != nil {
		d := fixedPolicyToDuration(*fixedPolicy)
		policies[duration.FixedDurationPolicyName] = duration.NewFixedDurationPolicy(d)
	}
	if condPolicy := policiesSpec.PodCondition; condPolicy != nil {
		policies[duration.PodConditionPolicyName] = duration.NewPodConditionPolicy(condPolicy.Type, condPolicy.Status)
	}
	return policies
}

func containerPolicyMatcher(policySpec autoscaling.ContainerPolicy) resource.ContainerMatcher {
	//lint:ignore SA1019 backwards-compatible support for deprecated ContainerName
	if name := policySpec.ContainerName; name != "" {
		return resource.FixedNameContainerMatcher{
			Name: name,
		}
	}
	if policySpec.MatchContainers != nil {
		return mapMatchContainersPolicy(policySpec.MatchContainers)
	}
	return nil
}

func mapMatchContainersPolicy(policySpec *autoscaling.MatchContainers) resource.ContainerMatcher {
	switch policySpec.Type {
	case autoscaling.MatchContainersTypeExactName:
		return resource.FixedNameContainerMatcher{
			Name: policySpec.Value,
		}
	case autoscaling.MatchContainersTypeRegexName:
		return resource.RegexNameContainerMatcher{
			Expr: policySpec.Value,
		}
	default:
		return nil
	}
}

// mapResourcePolicies maps the Resource Policy from the API spec to a slice of container policy entries
func mapResourcePolicies(spec autoscaling.ResourcePolicy) ([]containerPolicyEntry, error) {
	var errs []error
	var entries []containerPolicyEntry
	for _, policySpec := range spec.ContainerPolicies {
		matcher := containerPolicyMatcher(policySpec)
		if matcher == nil {
			errs = append(errs,
				fmt.Errorf("container policy must specify either containerName or matchContainers"))
			continue
		}

		var policy resource.ContainerPolicy
		var cnt int
		if fixedResources := policySpec.FixedResources; fixedResources != nil {
			policy = resource.NewFixedPolicy(fixedResources.Requests, fixedResources.Limits)
			cnt++
		}
		if percIncrease := policySpec.PercentageIncrease; percIncrease != nil {
			policy = resource.NewPercentageContainerPolicy(percIncrease.Value)
			cnt++
		}
		if cnt != 1 {
			//lint:ignore SA1019 backwards-compatible support for deprecated ContainerName
			name := policySpec.ContainerName
			if name == "" && policySpec.MatchContainers != nil {
				name = policySpec.MatchContainers.Value
			}
			errs = append(
				errs,
				fmt.Errorf("invalid number of resource policies for container %s; must be one",
					name))
			continue
		}
		entries = append(entries, containerPolicyEntry{
			matcher: matcher,
			policy:  policy,
		})
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return entries, nil
}

// fixedPolicyToDuration maps the attributes from FixedDurationPolicy API spec to the
// time duration
func fixedPolicyToDuration(policy autoscaling.FixedDurationPolicy) time.Duration {
	switch policy.Unit {
	case autoscaling.FixedDurationPolicyUnitMin:
		return time.Duration(policy.Value) * time.Minute
	default:
		return time.Duration(policy.Value) * time.Second
	}
}

const (
	PodConditionPodResizePending    corev1.PodConditionType = "PodResizePending"
	PodConditionPodResizeInProgress corev1.PodConditionType = "PodResizeInProgress"
)

func findPodCondition(pod *corev1.Pod, condType corev1.PodConditionType) *corev1.PodCondition {
	if pod == nil {
		return nil
	}
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == condType {
			return &pod.Status.Conditions[i]
		}
	}
	return nil
}

func isConditionTrue(cond *corev1.PodCondition) bool {
	return cond != nil && cond.Status == corev1.ConditionTrue
}

func conditionChanged(oldCond, newCond *corev1.PodCondition) bool {
	if !isConditionTrue(newCond) {
		return false
	}
	if !isConditionTrue(oldCond) {
		return true
	}
	return oldCond.Reason != newCond.Reason || oldCond.Message != newCond.Message
}

func (b *StartupCPUBoostImpl) inspectResizeConditions(ctx context.Context, pod *corev1.Pod) {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)

	b.RLock()
	oldPod := b.pods[pod.Name]
	b.RUnlock()

	b.logResizePending(log, oldPod, pod)
	b.logResizeInProgress(log, oldPod, pod)
}

func (b *StartupCPUBoostImpl) logResizePending(log logr.Logger, oldPod, newPod *corev1.Pod) {
	newCond := findPodCondition(newPod, PodConditionPodResizePending)
	oldCond := findPodCondition(oldPod, PodConditionPodResizePending)

	if conditionChanged(oldCond, newCond) {
		log.Info("pod resize is pending",
			"reason", newCond.Reason,
			"message", newCond.Message,
		)
	}
}

func (b *StartupCPUBoostImpl) logResizeInProgress(log logr.Logger, oldPod, newPod *corev1.Pod) {
	newCond := findPodCondition(newPod, PodConditionPodResizeInProgress)
	oldCond := findPodCondition(oldPod, PodConditionPodResizeInProgress)

	if isConditionTrue(newCond) && !isConditionTrue(oldCond) {
		log.V(5).Info("pod resize in progress", "message", newCond.Message)
	}
}
