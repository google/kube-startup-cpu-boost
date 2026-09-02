// Copyright 2026 Google LLC
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

package pod

import corev1 "k8s.io/api/core/v1"

// ResourceResizeRequiresRestart determines if a in-place resize of a resource with a given
// name requires container restart.
func ResourceResizeRequiresRestart(c corev1.Container, r corev1.ResourceName) bool {
	for _, p := range c.ResizePolicy {
		if p.ResourceName != r {
			continue
		}
		return p.RestartPolicy == corev1.RestartContainer
	}
	return false
}

// HasCPUResourcesToIncrease determines if a container has any CPU resources to increase.
func HasCPUResourcesToIncrease(c corev1.Container) bool {
	return !c.Resources.Requests.Cpu().IsZero() || !c.Resources.Limits.Cpu().IsZero()
}

// ContainerNameSet is a set of container names.
type ContainerNameSet map[string]bool

// Has returns true if the set contains the specified container name.
func (s ContainerNameSet) Has(name string) bool {
	return s[name]
}

// ContainerNameSetFromPod returns a set containing all container and init container names from the Pod.
func ContainerNameSetFromPod(pod *corev1.Pod) ContainerNameSet {
	s := make(ContainerNameSet)
	for _, c := range pod.Spec.Containers {
		s[c.Name] = true
	}
	for _, c := range pod.Spec.InitContainers {
		s[c.Name] = true
	}
	return s
}

// ContainerNameSetFromSlice returns a set populated from the given slice of names.
func ContainerNameSetFromSlice(names []string) ContainerNameSet {
	s := make(ContainerNameSet)
	for _, name := range names {
		s[name] = true
	}
	return s
}
