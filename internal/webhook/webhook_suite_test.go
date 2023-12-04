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

package webhook_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var (
	podTemplate          *corev1.Pod
	containerOneName     string
	containerTwoName     string
	containerOneCPUReq   string
	containerOneCPULimit string
	containerTwoCPUReq   string
	containerTwoCPULimit string
)

var _ = BeforeSuite(func() {
	containerOneName = "container-one"
	containerOneCPUReq = "500m"
	containerOneCPULimit = "1000m"
	containerTwoName = "container-two"
	containerTwoCPUReq = "1"
	containerTwoCPULimit = "2"

	containerOneCPUReqObj, err := apiResource.ParseQuantity(containerOneCPUReq)
	Expect(err).NotTo(HaveOccurred())
	containerOneCPULimitObj, err := apiResource.ParseQuantity(containerOneCPULimit)
	Expect(err).NotTo(HaveOccurred())
	containerTwoCPUReqObj, err := apiResource.ParseQuantity(containerTwoCPUReq)
	Expect(err).NotTo(HaveOccurred())
	containerTwoCPULimitObj, err := apiResource.ParseQuantity(containerTwoCPULimit)
	Expect(err).NotTo(HaveOccurred())
	podTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-1235",
			Namespace: "demo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: containerOneName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: containerOneCPUReqObj,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: containerOneCPULimitObj,
						},
					},
				},
				{
					Name: containerTwoName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: containerTwoCPUReqObj,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: containerTwoCPULimitObj,
						},
					},
				},
			},
		},
	}
})
