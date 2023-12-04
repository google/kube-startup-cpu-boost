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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

var containerTemplate *corev1.Container

func TestResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Suite")
}

var _ = BeforeSuite(func() {
	cpuRequestsQty, err := apiResource.ParseQuantity("500m")
	Expect(err).NotTo(HaveOccurred())
	cpuLimitsQty, err := apiResource.ParseQuantity("1")
	Expect(err).NotTo(HaveOccurred())
	containerTemplate = &corev1.Container{
		Name: "container-one",
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: cpuRequestsQty,
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: cpuLimitsQty,
			},
		},
	}
})
