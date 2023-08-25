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
	"encoding/json"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
)

var (
	errInvalidPodSpecNoLabel               = errors.New("pod is missing startup cpu boost label")
	errInvalidPodSpecNoAnnotation          = errors.New("pod is missing startup cpu boost annotation")
	errInvalidPodSpecAnnotationNoTimestamp = errors.New("pod startup cpu boost annotation has no timestamp")
	errInvalidPodSpecAnnotationNoRequests  = errors.New("pod startup cpu boost annotation has no init cpu requests")
)

type StartupCPUBoostPodAnnotation struct {
	BoostTimestamp  *time.Time        `json:"timestamp,omitempty"`
	InitCPURequests map[string]string `json:"initCPURequests,omitempty"`
	InitCPULimits   map[string]string `json:"initCPULimits,omitempty"`
}

type startupCPUBoostPod struct {
	name            string
	namespace       string
	boostName       string
	boostTimestamp  time.Time
	initCPURequests map[string]string
	initCPULimits   map[string]string
}

func NewStartupCPUBoostPod(pod *corev1.Pod) (*startupCPUBoostPod, error) {
	boostName, ok := pod.Labels[StartupCPUBoostPodLabelKey]
	if !ok {
		return nil, errInvalidPodSpecNoLabel
	}
	podAnnot, ok := pod.Annotations[StartupCPUBoostPodAnnotationKey]
	if !ok {
		return nil, errInvalidPodSpecNoAnnotation
	}
	boostPod, err := podBoostAnnotationToPod(podAnnot)
	if err != nil {
		return nil, err
	}
	boostPod.name = pod.Name
	boostPod.namespace = pod.Namespace
	boostPod.boostName = boostName
	return boostPod, nil
}

func (p *startupCPUBoostPod) GetName() string {
	return p.name
}

func (p *startupCPUBoostPod) GetNamespace() string {
	return p.namespace
}

func (p *startupCPUBoostPod) GetBoostName() string {
	return p.boostName
}

func podBoostAnnotationToPod(annotJSON string) (*startupCPUBoostPod, error) {
	annot := StartupCPUBoostPodAnnotation{}
	if err := json.Unmarshal([]byte(annotJSON), &annot); err != nil {
		return nil, err
	}
	if annot.BoostTimestamp == nil {
		return nil, errInvalidPodSpecAnnotationNoTimestamp
	}
	if len(annot.InitCPURequests) < 1 {
		return nil, errInvalidPodSpecAnnotationNoRequests
	}
	return &startupCPUBoostPod{
		boostTimestamp:  *annot.BoostTimestamp,
		initCPURequests: annot.InitCPURequests,
		initCPULimits:   annot.InitCPULimits,
	}, nil
}

func NewStartupCPUBoostPodAnnotation(timestamp *time.Time) *StartupCPUBoostPodAnnotation {
	return &StartupCPUBoostPodAnnotation{
		BoostTimestamp:  timestamp,
		InitCPURequests: make(map[string]string),
		InitCPULimits:   make(map[string]string),
	}
}

func (a StartupCPUBoostPodAnnotation) MustMarshalToJSON() string {
	result, err := json.Marshal(a)
	if err != nil {
		panic("must marshall to JSON returned an error: " + err.Error())
	}
	return string(result)
}
