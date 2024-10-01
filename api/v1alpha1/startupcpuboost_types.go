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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FixedDurationPolicyUnit defines the unit of time for a fixed
// time duration policy
// +kubebuilder:validation:Enum=Seconds;Minutes
type FixedDurationPolicyUnit string

const (
	FixedDurationPolicyUnitSec FixedDurationPolicyUnit = "Seconds"
	FixedDurationPolicyUnitMin FixedDurationPolicyUnit = "Minutes"
)

// FixedDurationPolicy defines the fixed time duration policy
type FixedDurationPolicy struct {
	// unit of time for a fixed time policy
	// +kubebuilder:validation:Required
	Unit FixedDurationPolicyUnit `json:"unit,omitempty"`
	// duration value for a fixed time policy
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=1
	Value int64 `json:"value,omitempty"`
}

// PodConditionDurationPolicy defines the PodCondition based
// duration policy
type PodConditionDurationPolicy struct {
	// type of a PODCondition to check in a policy
	Type corev1.PodConditionType `json:"type,omitempty"`
	// status of a PODCondition to match in a policy
	Status corev1.ConditionStatus `json:"status,omitempty"`
}

// AutoDurationPolicy defines the autoPolicy based duration policy
type AutoDurationPolicy struct {
	// Metric specifies the metric to be used for automatic adjustment
	ApiEndpoint string `json:"apiEndpoint,omitempty"`
}

// DurationPolicy defines the policy used to determine the duration
// time of a resource boost
type DurationPolicy struct {
	// fixed time duration policy
	// +kubebuilder:validation:Optional
	Fixed *FixedDurationPolicy `json:"fixedDuration,omitempty"`
	// podCondition based duration policy
	// +kubebuilder:validation:Optional
	PodCondition *PodConditionDurationPolicy `json:"podCondition,omitempty"`
	// autoPolicy based duration policy
	// +kubebuilder:validation:Optional
	AutoPolicy *AutoDurationPolicy `json:"autoPolicy,omitempty"`
}

// FixedResources defines the CPU resource policy that sets CPU resources
// to the given values
type FixedResources struct {
	// Requests specifies the CPU requests
	// +kubebuilder:validation:Required
	Requests resource.Quantity `json:"requests,omitempty"`
	// Limits specifies the CPU requests
	// +kubebuilder:validation:Optional
	Limits resource.Quantity `json:"limits,omitempty"`
}

// PercentageIncrease defines the CPU resource policy that increases
// CPU resources by the given percentage value
type PercentageIncrease struct {
	// Value specifies the percentage value
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum:=1
	Value int64 `json:"value,omitempty"`
}

// AutoPolicy defines the CPU resource policy that automatically adjusts
// CPU resources based on certain metrics or conditions
type AutoResourcePolicy struct {
	// Metric specifies the metric to be used for automatic adjustment
	ApiEndpoint string `json:"apiEndpoint,omitempty"`
}

// ContainerPolicy defines the policy used to determine the target
// resources for a container
type ContainerPolicy struct {
	// ContainerName specifies the name of container for a given policy
	// +kubebuilder:validation:Required
	ContainerName string `json:"containerName,omitempty"`
	// PercentageIncrease specifies the CPU resource policy that increases
	// CPU resources by the given percentage value
	// +kubebuilder:validation:Optional
	PercentageIncrease *PercentageIncrease `json:"percentageIncrease,omitempty"`
	// FixedResources specifies the CPU resource policy that sets the CPU
	// resources to the given values
	// +kubebuilder:validation:Optional
	FixedResources *FixedResources `json:"fixedResources,omitempty"`
	// AutoPolicy specifies the CPU resource policy that automatically adjusts
	// CPU resources based on certain metrics or conditions
	// +kubebuilder:validation:Optional
	AutoPolicy *AutoResourcePolicy `json:"autoPolicy,omitempty"`
}

// ResourcePolicy defines the policy used to determine the target
// resources for a POD
type ResourcePolicy struct {
	// ContainerPolicies specifies resource policies for the containers
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems:=1
	ContainerPolicies []ContainerPolicy `json:"containerPolicies,omitempty"`
}

// StartupCPUBoostSpec defines the desired state of StartupCPUBoost
type StartupCPUBoostSpec struct {
	// ResourcePolicy specifies policies for container resource increase
	ResourcePolicy ResourcePolicy `json:"resourcePolicy,omitempty"`
	// DurationPolicy specifies policies for resource boost duration
	// +kubebuilder:validation:Required
	DurationPolicy DurationPolicy `json:"durationPolicy,omitempty"`
}

// StartupCPUBoostStatus defines the observed state of StartupCPUBoost
type StartupCPUBoostStatus struct {
	// activeContainerBoosts is the number of containers which CPU
	// resources were increased by the StartupCPUBoost and not yet
	// reverted back to the original values
	// +kubebuilder:validation:Optional
	ActiveContainerBoosts int32 `json:"activeContainerBoosts,omitempty"`
	// totalContainerBoosts is the number of containers which CPU
	// resources were increased by the StartupCPUBoost
	// +kubebuilder:validation:Optional
	TotalContainerBoosts int32 `json:"totalContainerBoosts,omitempty"`
	// Conditions hold the latest available observations of the StartupCPUBoost
	// current state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// StartupCPUBoost is the Schema for the startupcpuboosts API
type StartupCPUBoost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Selector metav1.LabelSelector  `json:"selector,omitempty"`
	Spec     StartupCPUBoostSpec   `json:"spec,omitempty"`
	Status   StartupCPUBoostStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StartupCPUBoostList contains a list of StartupCPUBoost
type StartupCPUBoostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StartupCPUBoost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StartupCPUBoost{}, &StartupCPUBoostList{})
}
