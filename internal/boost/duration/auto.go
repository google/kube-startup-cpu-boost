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
	"context"
	"encoding/json"
	"net/http"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

type AutoDurationPolicy struct {
	apiEndpoint string
}

type DurationPrediction struct {
	Duration string `json:"duration"`
}

func NewAutoDurationPolicy(apiEndpoint string) *AutoDurationPolicy {
	return &AutoDurationPolicy{
		apiEndpoint: apiEndpoint,
	}
}

func (p *AutoDurationPolicy) GetDuration(ctx context.Context) (time.Duration, error) {
	prediction, err := p.getPrediction(ctx)
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(prediction.Duration)
}

func (p *AutoDurationPolicy) getPrediction(ctx context.Context) (*DurationPrediction, error) {
	log := ctrl.LoggerFrom(ctx).WithName("auto-duration-policy").WithValues("apiEndpoint", p.apiEndpoint)
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.apiEndpoint, nil)
	if err != nil {
		log.Error(err, "failed to create request")
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "failed to call API")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error(err, "unexpected status code from API", "statusCode", resp.StatusCode)
		return nil, err
	}

	var prediction DurationPrediction
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		log.Error(err, "failed to decode API response")
		return nil, err
	}

	return &prediction, nil
}
