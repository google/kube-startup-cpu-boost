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

package resource_test

import (
	"context"

	"github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Fixed Resource Policy", func() {
	var (
		policy                 resource.ContainerPolicy
		newResources           *corev1.ResourceRequirements
		container              *corev1.Container
		cpuRequests, cpuLimits apiResource.Quantity
	)

	BeforeEach(func() {
		cpuRequests = apiResource.MustParse("1")
		cpuLimits = apiResource.MustParse("2")
	})
	JustBeforeEach(func() {
		policy = resource.NewFixedPolicy(cpuRequests, cpuLimits)
		newResources = policy.NewResources(context.TODO(), container)
	})
	When("There are resources and limits defined", func() {
		BeforeEach(func() {
			container = containerTemplate.DeepCopy()
			cpuReq := apiResource.MustParse("500m")
			cpuLim := apiResource.MustParse("1")
			container.Resources.Requests[corev1.ResourceCPU] = cpuReq
			container.Resources.Limits[corev1.ResourceCPU] = cpuLim
		})
		It("returns resources with a valid CPU requests", func() {
			Expect(newResources.Requests).To(HaveKey(corev1.ResourceCPU))
			qty := newResources.Requests[corev1.ResourceCPU]
			Expect(qty.String()).To(Equal(cpuRequests.String()))
		})
		It("returns resources with a valid CPU limits", func() {
			Expect(newResources.Limits).To(HaveKey(corev1.ResourceCPU))
			qty := newResources.Limits[corev1.ResourceCPU]
			Expect(qty.String()).To(Equal(cpuLimits.String()))
		})
	})
	When("There are no requests and limits defined", func() {
		BeforeEach(func() {
			container = containerTemplate.DeepCopy()
			container.Resources.Requests = nil
			container.Resources.Limits = nil
		})
		It("returns empty new resources", func() {
			Expect(newResources.Requests).To(HaveLen(0))
			Expect(newResources.Limits).To(HaveLen(0))
		})
	})
	When("There are lower requests and limits defined", func() {
		var (
			containerCPUReq, containerCPULim apiResource.Quantity
		)
		BeforeEach(func() {
			containerCPUReq = cpuRequests.DeepCopy()
			containerCPUReq.Add(apiResource.MustParse("1"))
			containerCPULim = cpuLimits.DeepCopy()
			containerCPULim.Add(apiResource.MustParse("1"))
			container = containerTemplate.DeepCopy()
			container.Resources.Requests[corev1.ResourceCPU] = containerCPUReq
			container.Resources.Limits[corev1.ResourceCPU] = containerCPULim
		})
		It("returns resources with a valid CPU requests", func() {
			Expect(newResources.Requests).To(HaveKey(corev1.ResourceCPU))
			qty := newResources.Requests[corev1.ResourceCPU]
			Expect(qty.String()).To(Equal(containerCPUReq.String()))
		})
		It("returns resources with a valid CPU limits", func() {
			Expect(newResources.Limits).To(HaveKey(corev1.ResourceCPU))
			qty := newResources.Limits[corev1.ResourceCPU]
			Expect(qty.String()).To(Equal(containerCPULim.String()))
		})
	})
})
