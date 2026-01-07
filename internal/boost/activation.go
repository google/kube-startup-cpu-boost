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
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// BoostActivation represents a single boost activation instance
// It tracks when a boost was activated and what triggered it
type BoostActivation struct {
	// TriggerType is the type of trigger that caused this activation
	TriggerType autoscaling.BoostTriggerType
	// StartTime is when the boost was activated
	StartTime time.Time
	// ExpiryCondition defines when this activation should expire
	// This is derived from the DurationPolicy
	ExpiryCondition ExpiryCondition
}

// ExpiryCondition defines when a boost activation should expire
type ExpiryCondition struct {
	// Type indicates the expiry condition type
	Type ExpiryConditionType
	// FixedDuration is set when Type is ExpiryConditionTypeFixedDuration
	FixedDuration *time.Duration
	// PodCondition is set when Type is ExpiryConditionTypePodCondition
	PodCondition *PodConditionExpiry
}

// ExpiryConditionType indicates how a boost activation expires
type ExpiryConditionType string

const (
	// ExpiryConditionTypeFixedDuration means the boost expires after a fixed duration
	ExpiryConditionTypeFixedDuration ExpiryConditionType = "FixedDuration"
	// ExpiryConditionTypePodCondition means the boost expires when a pod condition is met
	ExpiryConditionTypePodCondition ExpiryConditionType = "PodCondition"
)

// PodConditionExpiry defines pod condition-based expiry
type PodConditionExpiry struct {
	Type   string
	Status corev1.ConditionStatus
}

// NewBoostActivation creates a new BoostActivation from a trigger and duration policy
func NewBoostActivation(trigger autoscaling.BoostTrigger, durationPolicy autoscaling.DurationPolicy) BoostActivation {
	activation := BoostActivation{
		TriggerType: trigger.Type,
		StartTime:   time.Now(),
	}

	// Map duration policy to expiry condition
	if fixed := durationPolicy.Fixed; fixed != nil {
		var duration time.Duration
		switch fixed.Unit {
		case autoscaling.FixedDurationPolicyUnitMin:
			duration = time.Duration(fixed.Value) * time.Minute
		default:
			duration = time.Duration(fixed.Value) * time.Second
		}
		activation.ExpiryCondition = ExpiryCondition{
			Type:          ExpiryConditionTypeFixedDuration,
			FixedDuration: &duration,
		}
	} else if cond := durationPolicy.PodCondition; cond != nil {
		activation.ExpiryCondition = ExpiryCondition{
			Type: ExpiryConditionTypePodCondition,
			PodCondition: &PodConditionExpiry{
				Type:   string(cond.Type),
				Status: cond.Status,
			},
		}
	}

	return activation
}

// IsExpired checks if the activation has expired based on its expiry condition
func (ba *BoostActivation) IsExpired(pod *corev1.Pod) bool {
	switch ba.ExpiryCondition.Type {
	case ExpiryConditionTypeFixedDuration:
		if ba.ExpiryCondition.FixedDuration == nil {
			return false
		}
		// Fixed duration is based on pod scheduled time
		if pod.Status.StartTime == nil {
			return false
		}
		elapsed := time.Since(pod.Status.StartTime.Time)
		return elapsed >= *ba.ExpiryCondition.FixedDuration
	case ExpiryConditionTypePodCondition:
		if ba.ExpiryCondition.PodCondition == nil {
			return false
		}
		// Check if pod condition matches
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodConditionType(ba.ExpiryCondition.PodCondition.Type) {
				return condition.Status == ba.ExpiryCondition.PodCondition.Status
			}
		}
		return false
	default:
		return false
	}
}

// ShouldActivateForPodCreate checks if a boost should be activated for a PodCreate trigger
// This maintains backward compatibility - if no triggers are specified, default to PodCreate
func ShouldActivateForPodCreate(boostSpec autoscaling.StartupCPUBoostSpec) bool {
	// If no triggers specified, default to PodCreate (backward compatibility)
	if len(boostSpec.Triggers) == 0 {
		return true
	}

	// Check if any trigger is PodCreate
	for _, trigger := range boostSpec.Triggers {
		if trigger.Type == autoscaling.BoostTriggerTypePodCreate {
			return true
		}
	}

	return false
}
