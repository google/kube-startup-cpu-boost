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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StartupCPUBoost is an implementation of a StartupCPUBoost CRD
type StartupCPUBoost interface {
	Name() string
	Namespace() string
	ResourcePolicy(containerName string) (resource.ContainerPolicy, bool)
	DurationPolicies() map[string]duration.Policy
	Pod(name string) (*corev1.Pod, bool)
	UpsertPod(ctx context.Context, pod *corev1.Pod) error
	DeletePod(ctx context.Context, pod *corev1.Pod) error
	ValidatePolicy(ctx context.Context, name string) []*corev1.Pod
	RevertResources(ctx context.Context, pod *corev1.Pod) error
	Matches(pod *corev1.Pod) bool
}

// StartupCPUBoostImpl is an implementation of a StartupCPUBoost CRD
type StartupCPUBoostImpl struct {
	sync.RWMutex
	name             string
	namespace        string
	selector         labels.Selector
	durationPolicies map[string]duration.Policy
	resourcePolicies map[string]resource.ContainerPolicy
	pods             sync.Map
	client           client.Client
}

// NewStartupCPUBoost constructs startup-cpu-boost implementation from a given API spec
func NewStartupCPUBoost(client client.Client, boost *autoscaling.StartupCPUBoost) (StartupCPUBoost, error) {
	selector, err := metav1.LabelSelectorAsSelector(&boost.Selector)
	if err != nil {
		return nil, err
	}
	resourcePolicies, err := mapResourcePolicy(boost.Spec.ResourcePolicy)
	if err != nil {
		return nil, err
	}
	return &StartupCPUBoostImpl{
		name:             boost.Name,
		namespace:        boost.Namespace,
		selector:         selector,
		durationPolicies: mapDurationPolicy(boost.Spec.DurationPolicy),
		resourcePolicies: resourcePolicies,
		client:           client,
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

// ResourcePolicy returns the resource policy for a given container
func (b *StartupCPUBoostImpl) ResourcePolicy(containerName string) (resource.ContainerPolicy, bool) {
	policy, ok := b.resourcePolicies[containerName]
	return policy, ok
}

// DurationPolicies returns configured duration policies
func (b *StartupCPUBoostImpl) DurationPolicies() map[string]duration.Policy {
	return b.durationPolicies
}

// Pod returns a POD if tracked by startup-cpu-boost.
func (b *StartupCPUBoostImpl) Pod(name string) (*corev1.Pod, bool) {
	if v, ok := b.pods.Load(name); ok {
		return v.(*corev1.Pod), ok
	}
	return nil, false
}

// UpsertPod inserts new or updates existing POD to startup-cpu-boost tracking
// The update of existing POD triggers validation logic and may result in POD update
func (b *StartupCPUBoostImpl) UpsertPod(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("upserting a pod")
	if _, loaded := b.pods.Swap(pod.Name, pod); !loaded {
		log.V(5).Info("inserted non-existing pod")
		return nil
	}
	log.V(5).Info("updating existing pod")
	condPolicy, ok := b.durationPolicies[duration.PodConditionPolicyName]
	if !ok {
		log.V(5).Info("skipping pod update as podCondition policy is missing")
		return nil
	}
	if valid := b.validatePolicyOnPod(ctx, condPolicy, pod); !valid {
		log.V(5).Info("updating pod with initial resources")
		if err := b.RevertResources(ctx, pod); err != nil {
			return fmt.Errorf("failed to update pod: %s", err)
		}
	}
	return nil
}

// DeletePod removes the POD from the startup-cpu-boost tracking
func (b *StartupCPUBoostImpl) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("handling pod delete")
	if _, loaded := b.pods.LoadAndDelete(pod.Name); loaded {
		log.Info("deletion of untracked pod")
	}
	return nil
}

// ValidatePolicy validates policy with a given name on all startup-cpu-boost PODs.
// The function returns slice of PODs that violated the policy.
func (b *StartupCPUBoostImpl) ValidatePolicy(ctx context.Context, name string) (violated []*corev1.Pod) {
	violated = make([]*corev1.Pod, 0)
	policy, ok := b.durationPolicies[name]
	if !ok {
		return
	}
	b.pods.Range(func(key, value any) bool {
		pod := value.(*corev1.Pod)
		if !b.validatePolicyOnPod(ctx, policy, pod) {
			violated = append(violated, pod)
		}
		return true
	})
	return
}

// RevertResources updates POD's container resource requests and limits to their original
// values using the data from StartupCPUBoost annotation
func (b *StartupCPUBoostImpl) RevertResources(ctx context.Context, pod *corev1.Pod) error {
	if err := bpod.RevertResourceBoost(pod); err != nil {
		return fmt.Errorf("failed to update pod spec: %s", err)
	}
	if err := b.client.Update(ctx, pod); err != nil {
		return err
	}
	b.pods.Delete(pod.Name)
	return nil
}

// Matches verifies if a boost selector matches the given POD
func (b *StartupCPUBoostImpl) Matches(pod *corev1.Pod) bool {
	return b.selector.Matches(labels.Set(pod.Labels))
}

// loggerFromContext provides Logger from a current context with configured
// values common for startup-cpu-boost like name or namespace
func (b *StartupCPUBoostImpl) loggerFromContext(ctx context.Context) logr.Logger {
	return ctrl.LoggerFrom(ctx).
		WithName("startup-cpu-boost").
		WithValues(
			"name", b.name,
			"namespace", b.namespace,
		)
}

// validatePolicyOnPod validates given policy on a given POD.
// The function returns true if policy is valid or false otherwise
func (b *StartupCPUBoostImpl) validatePolicyOnPod(ctx context.Context, p duration.Policy, pod *corev1.Pod) (valid bool) {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	if valid = p.Valid(pod); !valid {
		log.WithValues("policy", p.Name()).V(5).Info("policy is not valid")
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

// mapResourcePolicy maps the Resource Policy from the API spec to the map of policy
// implementations with container name keys
func mapResourcePolicy(spec autoscaling.ResourcePolicy) (map[string]resource.ContainerPolicy, error) {
	var errs []error
	policies := make(map[string]resource.ContainerPolicy)
	for _, policySpec := range spec.ContainerPolicies {
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
			errs = append(errs, fmt.Errorf("invalid number of resource policies fo container %s; must be one", policySpec.ContainerName))
			continue
		}
		policies[policySpec.ContainerName] = policy
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return policies, nil
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
