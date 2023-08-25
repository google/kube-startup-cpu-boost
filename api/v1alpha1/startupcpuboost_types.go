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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StartupCPUBoostSpec defines the desired state of StartupCPUBoost
type StartupCPUBoostSpec struct {
	// TimePeriod defines the period of time, in seconds, that POD will be affected
	// by the CPU Boost after the initialization
	TimePeriod int64 `json:"timePeriod,omitempty"`

	// BootPercent defines the percent of CPU request increase that POD will get
	// during the CPU boost time period
	BoostPercent int64 `json:"boostPercent,omitempty"`
}

// StartupCPUBoostStatus defines the observed state of StartupCPUBoost
type StartupCPUBoostStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
