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
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
)

type FixedPolicy struct {
	cpuRequests apiResource.Quantity
	cpuLimits   apiResource.Quantity
}

func NewFixedPolicy(requests apiResource.Quantity, limits apiResource.Quantity) ContainerPolicy {
	return &FixedPolicy{
		cpuRequests: requests,
		cpuLimits:   limits,
	}
}

func (p *FixedPolicy) Requests() apiResource.Quantity {
	return p.cpuRequests
}

func (p *FixedPolicy) Limits() apiResource.Quantity {
	return p.cpuLimits
}

func (p *FixedPolicy) NewResources(ctx context.Context, container *corev1.Container) *corev1.ResourceRequirements {
	log := ctrl.LoggerFrom(ctx).WithName("fixed-cpu-policy").
		WithValues("newCPURequsts", p.cpuRequests.String()).
		WithValues("newCPULimits", p.cpuLimits.String())
	result := container.Resources.DeepCopy()
	p.setResource(corev1.ResourceCPU, result.Requests, p.cpuRequests, log)
	p.setResource(corev1.ResourceCPU, result.Limits, p.cpuLimits, log)
	return result
}

func (p *FixedPolicy) setResource(resource corev1.ResourceName, resources corev1.ResourceList, target apiResource.Quantity, log logr.Logger) {
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
