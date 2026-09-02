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

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
)

// PodEventType defines the type of domain event that occurred on a tracked Pod.
type PodEventType string

const (
	PodEventTypePodCreated          PodEventType = "PodCreated"
	PodEventTypePodDeleted          PodEventType = "PodDeleted"
	PodEventTypeConditionChanged    PodEventType = "ConditionChanged"
	PodEventTypeContainerRestarting PodEventType = "ContainerRestarting"
)

var (
	ErrNilPodEvent                   = errors.New("pod event cannot be nil")
	ErrEmptyType                     = errors.New("pod event type cannot be empty")
	ErrNilPod                        = errors.New("pod cannot be nil")
	ErrEmptyRestartingContainerNames = errors.New("restarting container names cannot be empty on container restarting event")
)

// PodEvent encapsulates pod event.
type PodEvent struct {
	Type                     PodEventType
	Pod                      *corev1.Pod
	RestartingContainerNames []string
}

// Validate validates pod event
func (e *PodEvent) Validate() error {
	if e == nil {
		return ErrNilPodEvent
	}
	if e.Type == "" {
		return ErrEmptyType
	}
	if e.Pod == nil {
		return ErrNilPod
	}
	if e.Type == PodEventTypeContainerRestarting && len(e.RestartingContainerNames) == 0 {
		return ErrEmptyRestartingContainerNames
	}
	return nil
}
