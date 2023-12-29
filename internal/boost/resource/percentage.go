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

	"gopkg.in/inf.v0"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

type PercentageContainerPolicy struct {
	percentage int64
}

func NewPercentageContainerPolicy(percentage int64) ContainerPolicy {
	return &PercentageContainerPolicy{
		percentage: percentage,
	}
}

func (p *PercentageContainerPolicy) Percentage() int64 {
	return p.percentage
}

func (p *PercentageContainerPolicy) NewResources(ctx context.Context, container *corev1.Container) *corev1.ResourceRequirements {
	result := container.Resources.DeepCopy()
	p.increaseResource(corev1.ResourceCPU, result.Requests)
	p.increaseResource(corev1.ResourceCPU, result.Limits)
	return result
}

func (p *PercentageContainerPolicy) increaseResource(resource corev1.ResourceName, resources corev1.ResourceList) {
	if quantity, ok := resources[resource]; ok {
		resources[resource] = *increaseQuantity(quantity, p.percentage)
	}
}

func increaseQuantity(quantity apiResource.Quantity, incPerc int64) *apiResource.Quantity {
	quantityDec := quantity.AsDec()
	decPerc := inf.NewDec(100+incPerc, 2)
	decResult := &inf.Dec{}
	decResult.Mul(quantityDec, decPerc)
	decRoundedResult := inf.Dec{}
	decRoundedResult.Round(decResult, 2, inf.RoundCeil)
	return apiResource.NewDecimalQuantity(decRoundedResult, quantity.Format)
}
