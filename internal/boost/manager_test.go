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

package boost

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPodCleanup(t *testing.T) {
	containerOneInitCPUReq := "100m"
	containerOneInitCPULimit := "1000m"
	containerTwoInitCPUReq := "200m"
	containerTwoInitCPULimit := "2000m"
	client := fake.NewClientBuilder().
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				podObj, ok := obj.(*corev1.Pod)
				if !ok {
					t.Fatalf("client get received non pod object")
				}
				containerOne := corev1.Container{
					Name: "container-001",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: mustParseQuantity("800m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: mustParseQuantity("1800m"),
						},
					},
				}
				containerTwo := corev1.Container{
					Name: "container-002",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: mustParseQuantity("1000m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU: mustParseQuantity("4000m"),
						},
					},
				}
				podObj.Spec.Containers = append(podObj.Spec.Containers, containerOne)
				podObj.Spec.Containers = append(podObj.Spec.Containers, containerTwo)
				return nil
			},
			Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				_, ok := obj.(*corev1.Pod)
				if !ok {
					t.Fatalf("client update received non pod object")
				}
				return nil
			},
		}).
		Build()
	manager := &managerImpl{
		client: client,
	}
	pod := &startupCPUBoostPod{
		name:           "pod-001",
		namespace:      "demo",
		boostName:      "boost-001",
		boostTimestamp: time.Now(),
		initCPURequests: map[string]string{
			"container-001": containerOneInitCPUReq,
			"container-002": containerTwoInitCPUReq,
		},
		initCPULimits: map[string]string{
			"container-001": containerOneInitCPULimit,
			"container-002": containerTwoInitCPULimit,
		},
	}
	err := manager.podCleanup(context.Background(), pod)
	if err != nil {
		t.Fatalf("err = %v; want nil", err)
	}
}

func mustParseQuantity(s string) apiResource.Quantity {
	q, err := apiResource.ParseQuantity(s)
	if err != nil {
		panic("unparsable quantity: " + err.Error())
	}
	return q
}
