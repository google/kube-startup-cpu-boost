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
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	AutoDurationPolicyName = "AutoDuration"
)

type AutoDurationPolicy struct {
	apiEndpoint string
}

func (p *AutoDurationPolicy) Name() string {
	return AutoDurationPolicyName
}

// Valid returns true if the pod is still within the duration
func (p *AutoDurationPolicy) Valid(pod *v1.Pod) bool {
	now := time.Now()
	duration, err := p.GetDuration(pod)

	if err != nil {
		return false
	}

	return pod.CreationTimestamp.Add(duration).After(now)
}

type DurationPrediction struct {
	Duration string `json:"duration"`
}

func NewAutoDurationPolicy(apiEndpoint string) *AutoDurationPolicy {
	return &AutoDurationPolicy{
		apiEndpoint: apiEndpoint,
	}
}

func (p *AutoDurationPolicy) GetDuration(pod *v1.Pod) (time.Duration, error) {
	prediction, err := p.getPrediction(pod)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(prediction.Duration)
}

func (p *AutoDurationPolicy) getPrediction(pod *v1.Pod) (*DurationPrediction, error) {
	podData, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(p.apiEndpoint+"/duration", "application/json", bytes.NewBuffer(podData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var prediction DurationPrediction
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return nil, err
	}
	return &prediction, nil
}
