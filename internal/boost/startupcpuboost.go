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
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/boost/policy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StartupCPUBoost is an implementation of a StartupCPUBoost CRD
type StartupCPUBoost struct {
	sync.RWMutex
	name      string
	namespace string
	percent   int64
	selector  labels.Selector
	policies  map[string]policy.DurationPolicy
	pods      sync.Map
	client    client.Client
}

// NewStartupCPUBoost constructs startup-cpu-boost implementation from a given API spec
func NewStartupCPUBoost(client client.Client, boost *autoscaling.StartupCPUBoost) (*StartupCPUBoost, error) {
	selector, err := metav1.LabelSelectorAsSelector(&boost.Selector)
	if err != nil {
		return nil, err
	}
	return &StartupCPUBoost{
		name:      boost.Name,
		namespace: boost.Namespace,
		selector:  selector,
		percent:   boost.Spec.BoostPercent,
		policies:  policiesFromSpec(boost.Spec.DurationPolicy),
		client:    client,
	}, nil
}

// Name returns startup-cpu-boost name
func (b *StartupCPUBoost) Name() string {
	return b.name
}

// Namespace returns startup-cpu-boost namespace
func (b *StartupCPUBoost) Namespace() string {
	return b.namespace
}

// BoostPercent returns startup-cpu-boost boost percentage
func (b *StartupCPUBoost) BoostPercent() int64 {
	return b.percent
}

// DurationPolicies returns configured duration policies
func (b *StartupCPUBoost) DurationPolicies() map[string]policy.DurationPolicy {
	return b.policies
}

// Pod returns a POD if tracked by startup-cpu-boost.
func (b *StartupCPUBoost) Pod(name string) (*corev1.Pod, bool) {
	if v, ok := b.pods.Load(name); ok {
		return v.(*corev1.Pod), ok
	}
	return nil, false
}

// UpsertPod inserts new or updates existing POD to startup-cpu-boost tracking
// The update of existing POD triggers validation logic and may result in POD update
func (b *StartupCPUBoost) UpsertPod(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("upserting a pod")
	if _, loaded := b.pods.Swap(pod.Name, pod); !loaded {
		log.V(5).Info("inserted non-existing pod")
		return nil
	}
	log.V(5).Info("updating existing pod")
	condPolicy, ok := b.policies[policy.PodConditionPolicyName]
	if !ok {
		log.V(5).Info("skipping pod update as podCondition policy is missing")
		return nil
	}
	if valid := b.validatePolicyOnPod(ctx, condPolicy, pod); !valid {
		log.V(5).Info("updating pod with initial resources")
		if err := b.revertResources(ctx, pod); err != nil {
			return fmt.Errorf("failed to update pod: %s", err)
		}
	}
	return nil
}

// DeletePod removes the POD from the startup-cpu-boost tracking
func (b *StartupCPUBoost) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	log.V(5).Info("handling pod delete")
	if _, loaded := b.pods.LoadAndDelete(pod.Name); loaded {
		log.Info("deletion of untracked pod")
	}
	return nil
}

// loggerFromContext provides Logger from a current context with configured
// values common for startup-cpu-boost like name or namespace
func (b *StartupCPUBoost) loggerFromContext(ctx context.Context) logr.Logger {
	return ctrl.LoggerFrom(ctx).
		WithName("startup-cpu-boost").
		WithValues(
			"name", b.name,
			"namespace", b.namespace,
		)
}

// validatePolicy validates policy with a given name on all startup-cpu-boost PODs.
// The function returns slice of PODs that violated the policy.
func (b *StartupCPUBoost) validatePolicy(ctx context.Context, name string) (violated []*corev1.Pod) {
	violated = make([]*corev1.Pod, 0)
	policy, ok := b.policies[name]
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

// validatePolicyOnPod validates given policy on a given POD.
// The function returns true if policy is valid or false otherwise
func (b *StartupCPUBoost) validatePolicyOnPod(ctx context.Context, p policy.DurationPolicy, pod *corev1.Pod) (valid bool) {
	log := b.loggerFromContext(ctx).WithValues("pod", pod.Name)
	if valid = p.Valid(pod); !valid {
		log.WithValues("policy", p.Name()).V(5).Info("policy is not valid")
	}
	return
}

// revertResources updates POD's container resource requests and limits to their original
// values using the data from StartupCPUBoost annotation
func (b *StartupCPUBoost) revertResources(ctx context.Context, pod *corev1.Pod) error {
	if err := bpod.RevertResourceBoost(pod); err != nil {
		return fmt.Errorf("failed to update pod spec: %s", err)
	}
	if err := b.client.Update(ctx, pod); err != nil {
		return err
	}
	b.pods.Delete(pod.Name)
	return nil
}

// policiesFromSpec maps the Duration Policies from the API spec to the map holding policy
// implementations under policy name keys
func policiesFromSpec(policiesSpec autoscaling.DurationPolicy) map[string]policy.DurationPolicy {
	policies := make(map[string]policy.DurationPolicy)
	if fixedPolicy := policiesSpec.Fixed; fixedPolicy != nil {
		duration := fixedPolicyToDuration(*fixedPolicy)
		policies[policy.FixedDurationPolicyName] = policy.NewFixedDurationPolicy(duration)
	}
	if condPolicy := policiesSpec.PodCondition; condPolicy != nil {
		policies[policy.PodConditionPolicyName] = policy.NewPodConditionPolicy(condPolicy.Type, condPolicy.Status)
	}
	return policies
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
