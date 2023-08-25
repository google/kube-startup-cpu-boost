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

package webhook

import (
	"encoding/json"
	"testing"

	"github.com/google/kube-startup-cpu-boost/internal/boost"
	. "github.com/onsi/ginkgo/v2"

	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestBoostContainersCPU(t *testing.T) {
	reqQuantityStr := "1"
	reqQuantity, _ := apiResource.ParseQuantity(reqQuantityStr)
	limitQuantityStr := "2"
	limitQuantity, _ := apiResource.ParseQuantity(limitQuantityStr)
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
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
					Name: "container-two",
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
	var boostPerc int64 = 20
	expReqQuantityStr := "1200m"
	expLimitQuantityStr := "2400m"
	log := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

	handler := &podCPUBoostHandler{}
	result, boosted := handler.boostContainersCPU(pod, boostPerc, log)
	if !boosted {
		t.Fatalf("boosted = %v; want %v", boosted, true)
	}
	if len(result) != len(pod.Spec.Containers) {
		t.Errorf("len(result) = %v; want %v", len(result), len(pod.Spec.Containers))
	}
	for i := range result {
		cpuReq := result[i].Resources.Requests[corev1.ResourceCPU]
		cpuLimit := result[i].Resources.Limits[corev1.ResourceCPU]
		if cpuReq.String() != expReqQuantityStr {
			t.Errorf("container %d: cpu requests = %v; want %v", i, cpuReq.String(), expReqQuantityStr)
		}
		if cpuLimit.String() != expLimitQuantityStr {
			t.Errorf("container %d: cpu limits = %v; want %v", i, cpuLimit.String(), expLimitQuantityStr)
		}
	}
	annotStr, ok := pod.Annotations[boost.StartupCPUBoostPodAnnotationKey]
	if !ok {
		t.Fatalf("POD is missing startup CPU boost annotation")
	}
	annot := &boost.StartupCPUBoostPodAnnotation{}
	if err := json.Unmarshal([]byte(annotStr), annot); err != nil {
		t.Fatalf("can't unmarshal boost annotation due to %s", err)
	}
	if len(annot.InitCPURequests) != len(pod.Spec.Containers) {
		t.Fatalf("CPU boost annotation: len(initCPURequests) = %v; want %v", len(annot.InitCPURequests), len(pod.Spec.Containers))
	}
	if len(annot.InitCPULimits) != len(pod.Spec.Containers) {
		t.Fatalf("CPU boost annotation: len(initCPULimits) = %v; want %v", len(annot.InitCPULimits), len(pod.Spec.Containers))
	}
	for _, container := range pod.Spec.Containers {
		initReq := annot.InitCPURequests[container.Name]
		initLimit := annot.InitCPULimits[container.Name]
		if initReq != reqQuantityStr {
			t.Errorf("CPU boost annotation: InitCPURequests[%v] = %v; want %v", container.Name, initReq, reqQuantityStr)
		}
		if initLimit != limitQuantityStr {
			t.Errorf("CPU boost annotation: InitCPULimits[%v] = %v; want %v", container.Name, initLimit, limitQuantityStr)
		}
	}
}

func TestIncreaseQuantityForResource(t *testing.T) {
	quantityStr := "250m"
	boostPerc := 120
	expectedQuantityStr := "550m"
	quantity, _ := apiResource.ParseQuantity(quantityStr)
	log := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))
	requests := corev1.ResourceList{
		corev1.ResourceCPU: quantity,
	}
	increased, init, new := increaseQuantityForResource(requests, corev1.ResourceCPU, int64(boostPerc), log)
	result := requests[corev1.ResourceCPU]
	if !increased {
		t.Errorf("increased = %v; want %v", increased, true)
	}
	if init.String() != quantityStr {
		t.Errorf("initial quantity = %v; want %v", init.String(), quantityStr)
	}
	if new.String() != expectedQuantityStr {
		t.Errorf("new quantity = %v; want %v", new.String(), expectedQuantityStr)
	}
	if result.String() != expectedQuantityStr {
		t.Errorf("quantity = %v; want %v", result, expectedQuantityStr)
	}
}

func TestIncreaseQuantity(t *testing.T) {
	type input struct {
		quantityStr string
		boostPerc   int64
	}
	inputs := []input{
		{"100m", 20},
		{"1.3", 50},
		{"800m", 100},
		{"4", 80},
		{"101m", 325},
		{"1", 20},
	}
	expected := []string{
		"120m",
		"1950m",
		"1600m",
		"7200m",
		"430m",
		"1200m",
	}

	for i := range inputs {
		quantity, err := apiResource.ParseQuantity(inputs[i].quantityStr)
		if err != nil {
			t.Fatalf("could not parse quantity due to %s", err)
		}
		result := increaseQuantity(quantity, inputs[i].boostPerc)
		if result.String() != expected[i] {
			t.Errorf("input %d, result = %v; want %v", i, result, expected[i])
		}
	}
}
