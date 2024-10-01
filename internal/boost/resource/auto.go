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

package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

type AutoPolicy struct {
	apiEndpoint string
}

type ResourcePrediction struct {
	CPURequests string `json:"cpuRequests"`
	CPULimits   string `json:"cpuLimits"`
}

func NewAutoPolicy(apiEndpoint string) ContainerPolicy {
	return &AutoPolicy{
		apiEndpoint: apiEndpoint,
	}
}

func (p *AutoPolicy) Requests(ctx context.Context) (apiResource.Quantity, error) {
	prediction, err := p.getPrediction(ctx)
	if err != nil {
		return apiResource.Quantity{}, err
	}
	return apiResource.ParseQuantity(prediction.CPURequests)
}

func (p *AutoPolicy) Limits(ctx context.Context) (apiResource.Quantity, error) {
	prediction, err := p.getPrediction(ctx)
	if err != nil {
		return apiResource.Quantity{}, err
	}
	return apiResource.ParseQuantity(prediction.CPULimits)
}

func (p *AutoPolicy) NewResources(ctx context.Context, container *corev1.Container) *corev1.ResourceRequirements {
	log := ctrl.LoggerFrom(ctx).WithName("auto-cpu-policy")
	prediction, err := p.getPrediction(ctx)
	if err != nil {
		return nil
	}

	cpuRequests, err := apiResource.ParseQuantity(prediction.CPURequests)
	if err != nil {
		log.Error(err, "failed to parse CPU requests")
		return nil
	}
	cpuLimits, err := apiResource.ParseQuantity(prediction.CPULimits)
	if err != nil {
		log.Error(err, "failed to parse CPU limits")
		return nil
	}

	log = log.WithValues("newCPURequests", cpuRequests.String(), "newCPULimits", cpuLimits.String())
	result := container.Resources.DeepCopy()
	p.setResource(corev1.ResourceCPU, result.Requests, cpuRequests, log)
	p.setResource(corev1.ResourceCPU, result.Limits, cpuLimits, log)
	return result
}

func (p *AutoPolicy) setResource(resource corev1.ResourceName, resources corev1.ResourceList, target apiResource.Quantity, log logr.Logger) {
	if target.IsZero() {
		return
	}
	current, ok := resources[resource]
	if !ok {
		return
	}
	if target.Cmp(current) < 0 {
		log.V(2).Info("container has higher CPU requests than policy")
		return
	}
	resources[resource] = target
}

func (p *AutoPolicy) getPrediction(ctx context.Context) (*ResourcePrediction, error) {
	// Retrieve the pod information from the context
	pod, ok := ctx.Value("pod").(*corev1.Pod)
	if !ok || pod == nil {
		return nil, errors.New("pod information is missing or invalid in context")
	}

	// Marshal the pod information to JSON
	reqBody, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pod information: %w", err)
	}

	// Create a new HTTP request with the pod information
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiEndpoint+"cpu", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for a successful response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode the response body into a ResourcePrediction struct
	var prediction ResourcePrediction
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &prediction, nil
}
