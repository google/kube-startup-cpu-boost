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

// Package duration contains implementation of resource boost duration policies
package duration

import corev1 "k8s.io/api/core/v1"

const (
	PolicyTypeFixed        = "Fixed"
	PolicyTypePodCondition = "PodCondition"
)

type Policy interface {
	Valid(pod *corev1.Pod) bool
	Name() string
}
