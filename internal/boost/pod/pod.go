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

// Package pod contains implementation of startup-cpu-boost POD manipulation
// functions
package pod

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

const (
	BoostLabelKey      = "autoscaling.x-k8s.io/startup-cpu-boost"
	BoostAnnotationKey = "autoscaling.x-k8s.io/startup-cpu-boost"
)

type BoostPodAnnotation struct {
	BoostTimestamp  time.Time         `json:"timestamp,omitempty"`
	InitCPURequests map[string]string `json:"initCPURequests,omitempty"`
	InitCPULimits   map[string]string `json:"initCPULimits,omitempty"`
}

func NewBoostAnnotation() *BoostPodAnnotation {
	return &BoostPodAnnotation{
		BoostTimestamp:  time.Now(),
		InitCPURequests: make(map[string]string),
		InitCPULimits:   make(map[string]string),
	}
}

func (a BoostPodAnnotation) ToJSON() string {
	result, err := json.Marshal(a)
	if err != nil {
		panic("failed to marshall to JSON: " + err.Error())
	}
	return string(result)
}

func BoostAnnotationFromPod(pod *corev1.Pod) (*BoostPodAnnotation, error) {
	annotation := &BoostPodAnnotation{}
	data, ok := pod.Annotations[BoostAnnotationKey]
	if !ok {
		return nil, errors.New("boost annotation not found")
	}
	if err := json.Unmarshal([]byte(data), annotation); err != nil {
		return nil, err
	}
	return annotation, nil
}

func RevertResourceBoost(pod *corev1.Pod) error {
	annotation, err := BoostAnnotationFromPod(pod)
	if err != nil {
		return fmt.Errorf("failed to get boost annotation from pod: %s", err)
	}
	delete(pod.Labels, BoostLabelKey)
	delete(pod.Annotations, BoostAnnotationKey)
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if request, ok := annotation.InitCPURequests[container.Name]; ok {
			if reqQuantity, err := apiResource.ParseQuantity(request); err == nil {
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceCPU] = reqQuantity
			} else {
				return fmt.Errorf("failed to parse CPU request: %s", err)
			}
		}
		if limit, ok := annotation.InitCPULimits[container.Name]; ok {
			if limitQuantity, err := apiResource.ParseQuantity(limit); err == nil {
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Limits[corev1.ResourceCPU] = limitQuantity
			} else {
				return fmt.Errorf("failed to parse CPU limit: %s", err)
			}
		}
	}
	return nil
}
