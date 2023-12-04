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

package duration

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	PodConditionPolicyName = "PodCondition"
)

type PodConditionPolicy struct {
	condition corev1.PodConditionType
	status    corev1.ConditionStatus
}

func NewPodConditionPolicy(condition corev1.PodConditionType, status corev1.ConditionStatus) Policy {
	return &PodConditionPolicy{
		condition: condition,
		status:    status,
	}
}

func (*PodConditionPolicy) Name() string {
	return PodConditionPolicyName
}

func (p *PodConditionPolicy) Condition() corev1.PodConditionType {
	return p.condition
}

func (p *PodConditionPolicy) Status() corev1.ConditionStatus {
	return p.status
}

func (p *PodConditionPolicy) Valid(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type != p.condition {
			continue
		}
		if condition.Status == p.status {
			return false
		}
	}
	return true
}
