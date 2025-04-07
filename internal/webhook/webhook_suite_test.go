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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var (
	oneContainerGuaranteedPodTemplate *corev1.Pod
	oneContainerBurstablePodTemplate  *corev1.Pod
	twoContainerGuaranteedPodTemplate *corev1.Pod
	twoContainerBurstablePodTemplate  *corev1.Pod
)

var _ = BeforeSuite(func() {
	oneContainerBurstablePodTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-1235",
			Namespace: "demo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("500m"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	oneContainerGuaranteedPodTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-1235",
			Namespace: "demo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	twoContainerBurstablePodTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-1235",
			Namespace: "demo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("500m"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "container-two",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("500m"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
	twoContainerGuaranteedPodTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-1235",
			Namespace: "demo",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "container-two",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}
})

type boostAnnotationPatchMatcher struct {
	containers []corev1.Container
}

func (b *boostAnnotationPatchMatcher) Match(actual any) (bool, error) {
	patchOperation, ok := actual.(jsonpatch.Operation)
	if !ok {
		return false,
			fmt.Errorf("HaveBoostAnnotationPatch matcher expects an jsonpatch.Operation. Got:\n%s",
				format.Object(actual, 1))
	}
	if patchOperation.Operation != "add" {
		return false, nil
	}
	if patchOperation.Path != "/metadata/annotations" {
		return false, nil
	}
	valueMap, ok := patchOperation.Value.(map[string]interface{})
	if !ok {
		return false, nil
	}
	boostAnnotationInt, ok := valueMap[bpod.BoostAnnotationKey]
	if !ok {
		return false, nil
	}
	boostAnnotationStr, ok := boostAnnotationInt.(string)
	if !ok {
		return false, nil
	}
	var boostAnnotation bpod.BoostPodAnnotation
	err := json.Unmarshal([]byte(boostAnnotationStr), &boostAnnotation)
	if err != nil {
		return false, nil
	}
	initCPURequests := make(map[string]string)
	initCPULimits := make(map[string]string)
	for _, container := range b.containers {
		initCPURequests[container.Name] = container.Resources.Requests.Cpu().String()
		initCPULimits[container.Name] = container.Resources.Limits.Cpu().String()
	}
	if !reflect.DeepEqual(boostAnnotation.InitCPURequests, initCPURequests) {
		return false, nil
	}
	if !reflect.DeepEqual(boostAnnotation.InitCPULimits, initCPULimits) {
		return false, nil
	}
	return true, nil
}

// FailureMessage returns a suitable failure message.
func (b *boostAnnotationPatchMatcher) FailureMessage(actual any) (message string) {
	return format.Message(actual, "to have boost annotation",
		fmt.Sprintf("containers: %s", b.containersToString()))
}

// NegatedFailureMessage returns a suitable negated failure message.
func (b *boostAnnotationPatchMatcher) NegatedFailureMessage(actual any) (message string) {
	return format.Message(actual, "to not have boost annotation",
		fmt.Sprintf("containers: %s", b.containersToString()))
}

func (b *boostAnnotationPatchMatcher) containersToString() string {
	stringValue := ""
	for _, container := range b.containers {
		stringValue += fmt.Sprintf("Container: %s\n", container.Name)
		stringValue += fmt.Sprintf("Resource requests: %+v\n", &container.Resources.Requests)
		stringValue += fmt.Sprintf("Resource limits: %+v\n\n ", &container.Resources.Limits)
	}
	return stringValue
}

func HaveBoostAnnotationPatch(containers []corev1.Container) types.GomegaMatcher {
	return &boostAnnotationPatchMatcher{
		containers: containers,
	}
}
