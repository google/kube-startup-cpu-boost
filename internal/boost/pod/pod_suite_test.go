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

package pod_test

import (
	"testing"

	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod Suite")
}

var (
	podTemplate      *corev1.Pod
	containerOneName string
	containerTwoName string
)

var _ = BeforeSuite(func() {
	containerOneName = "container-one"
	containerTwoName = "container-two"
	reqQuantity, err := apiResource.ParseQuantity("1")
	Expect(err).ShouldNot(HaveOccurred())
	limitQuantity, err := apiResource.ParseQuantity("2")
	Expect(err).ShouldNot(HaveOccurred())
	podTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				bpod.BoostLabelKey: "boost-001",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: containerOneName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: reqQuantity,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: limitQuantity,
						},
					},
				},
				{
					Name: containerTwoName,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: reqQuantity,
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: limitQuantity,
						},
					},
				},
			},
		},
	}
})
