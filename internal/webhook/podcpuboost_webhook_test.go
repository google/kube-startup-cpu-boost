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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	bwebhook "github.com/google/kube-startup-cpu-boost/internal/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("Pod CPU Boost Webhook", func() {
	Describe("Handles admission requests", func() {
		var (
			mockCtrl    *gomock.Controller
			manager     *mock.MockManager
			managerCall *gomock.Call
			pod         *corev1.Pod
			response    webhook.AdmissionResponse
		)
		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			mockCtrl = gomock.NewController(GinkgoT())
			manager = mock.NewMockManager(mockCtrl)
			managerCall = manager.EXPECT().StartupCPUBoostForPod(
				gomock.Any(),
				gomock.Cond(func(x any) bool {
					p, ok := x.(*corev1.Pod)
					if !ok {
						return false
					}
					return p.Name == pod.Name && p.Namespace == pod.Namespace
				}),
			)
		})
		JustBeforeEach(func() {
			podJSON, err := json.Marshal(pod)
			Expect(err).NotTo(HaveOccurred())
			admissionReq := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: podJSON,
					},
				},
			}
			hook := bwebhook.NewPodCPUBoostWebHook(manager, scheme.Scheme)
			response = hook.Handle(context.TODO(), admissionReq)
		})
		When("there is no matching Startup CPU Boost", func() {
			BeforeEach(func() {
				managerCall.Return(nil, false)
			})
			It("calls the Startup CPU Boost manager", func() {
				managerCall.Times(1)
			})
			It("allows the admission", func() {
				Expect(response.Allowed).To(BeTrue())
			})
			It("returns zero patches", func() {
				Expect(response.Patches).To(HaveLen(0))
			})
		})
		When("there is a matching Startup CPU Boost", func() {
			When("there is no policy for any container", func() {
				var (
					boost            *mock.MockStartupCPUBoost
					resPolicyCallOne *gomock.Call
					resPolicyCallTwo *gomock.Call
				)
				BeforeEach(func() {
					boost = mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					resPolicyCallOne = boost.EXPECT().ResourcePolicy(gomock.Eq(containerOneName)).Return(nil, false)
					resPolicyCallTwo = boost.EXPECT().ResourcePolicy(gomock.Eq(containerTwoName)).Return(nil, false)
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for containers", func() {
					resPolicyCallOne.Times(1)
					resPolicyCallTwo.Times(1)
				})
				It("allows the admission", func() {
					Expect(response.Allowed).To(BeTrue())
				})
				It("returns zero patches", func() {
					Expect(response.Patches).To(HaveLen(0))
				})
			})
			When("there is a policy for one container", func() {
				var (
					boostName        string
					boost            *mock.MockStartupCPUBoost
					resPolicy        resource.ContainerPolicy
					resPolicyCallOne *gomock.Call
					resPolicyCallTwo *gomock.Call
				)
				BeforeEach(func() {
					boost = mock.NewMockStartupCPUBoost(mockCtrl)
					boostName = "boost-one"
					boost.EXPECT().Name().AnyTimes().Return(boostName)
					resPolicy = resource.NewPercentageContainerPolicy(120)
					resPolicyCallOne = boost.EXPECT().ResourcePolicy(gomock.Eq(containerOneName)).Return(resPolicy, true)
					resPolicyCallTwo = boost.EXPECT().ResourcePolicy(gomock.Eq(containerTwoName)).Return(nil, false)
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for containers", func() {
					resPolicyCallOne.Times(1)
					resPolicyCallTwo.Times(1)
				})
				It("allows the admission", func() {
					Expect(response.Allowed).To(BeTrue())
				})
				It("returns admission with four patches", func() {
					Expect(response.Patches).To(HaveLen(4))
				})
				It("returns admission with boost label patch", func() {
					Expect(response.Patches).To(ContainElement(boostLabelPatch(boostName)))
				})
				It("returns admission with boost annotation patch", func() {
					annotPatch, found := boostAnnotationPatch(response.Patches)
					Expect(found).To(BeTrue())
					annot, err := boostAnnotationFromPatch(annotPatch)
					Expect(err).NotTo(HaveOccurred())
					Expect(annot.InitCPURequests).To(HaveKeyWithValue(
						containerOneName,
						pod.Spec.Containers[0].Resources.Requests.Cpu().String(),
					))
					Expect(annot.InitCPULimits).To(HaveKeyWithValue(
						containerOneName,
						pod.Spec.Containers[0].Resources.Limits.Cpu().String(),
					))
				})
				It("returns admission with container-one requests patch", func() {
					patch := containerResourcePatch(pod, resPolicy, "requests", 0)
					Expect(response.Patches).To(ContainElement(patch))
				})
				It("returns admission with container-one limits patch", func() {
					patch := containerResourcePatch(pod, resPolicy, "limits", 0)
					Expect(response.Patches).To(ContainElement(patch))
				})
				When("container has no request and no limits set", func() {
					BeforeEach(func() {
						pod.Spec.Containers[0].Resources.Requests = nil
						pod.Spec.Containers[0].Resources.Limits = nil
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns admission with zero patches", func() {
						Expect(response.Patches).To(HaveLen(0))
					})
				})
				When("container has only requests set", func() {
					BeforeEach(func() {
						pod.Spec.Containers[0].Resources.Limits = nil
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns admission with three patches", func() {
						Expect(response.Patches).To(HaveLen(3))
					})
				})
				When("container has restart container resize policy", func() {
					BeforeEach(func() {
						pod.Spec.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{
							{
								ResourceName:  corev1.ResourceCPU,
								RestartPolicy: corev1.RestartContainer,
							},
						}
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns admission with zero patches", func() {
						Expect(response.Patches).To(HaveLen(0))
					})
				})
			})
			When("there is a policy for two containers", func() {
				var (
					resPolicyCallOne *gomock.Call
					resPolicyCallTwo *gomock.Call
				)
				BeforeEach(func() {
					boost := mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					resPolicy := resource.NewPercentageContainerPolicy(120)
					resPolicyCallOne = boost.EXPECT().ResourcePolicy(gomock.Eq(containerOneName)).Return(resPolicy, true)
					resPolicyCallTwo = boost.EXPECT().ResourcePolicy(gomock.Eq(containerTwoName)).Return(resPolicy, true)
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for containers", func() {
					resPolicyCallOne.Times(1)
					resPolicyCallTwo.Times(1)
				})
				It("allows the admission", func() {
					Expect(response.Allowed).To(BeTrue())
				})
				It("returns admission with six patches", func() {
					Expect(response.Patches).To(HaveLen(6))
				})
			})
		})
	})
})

func boostAnnotationFromPatch(patch jsonpatch.Operation) (*bpod.BoostPodAnnotation, error) {
	valueMap, ok := patch.Value.(map[string]interface{})
	if !ok {
		return nil, errors.New("patch value is not map[string]interface{}")
	}
	annotValue, ok := valueMap[bpod.BoostAnnotationKey]
	if !ok {
		return nil, errors.New("patch value map has no boost annotation key")
	}
	annotStr, err := strconv.Unquote(fmt.Sprintf("`%s`", annotValue))
	if err != nil {
		return nil, errors.New("cannot unquote boost annotation JSON")
	}
	var annot bpod.BoostPodAnnotation
	if err := json.Unmarshal([]byte(annotStr), &annot); err != nil {
		return nil, err
	}
	return &annot, nil
}

func boostAnnotationPatch(patches []jsonpatch.Operation) (jsonpatch.Operation, bool) {
	for _, patch := range patches {
		if patch.Path == "/metadata/annotations" && patch.Operation == "add" {
			return patch, true
		}
	}
	return jsonpatch.Operation{}, false
}

func boostLabelPatch(boostName string) jsonpatch.Operation {
	return jsonpatch.Operation{
		Operation: "add",
		Path:      "/metadata/labels",
		Value: map[string]interface{}{
			bpod.BoostLabelKey: boostName,
		},
	}
}

func containerResourcePatch(pod *corev1.Pod, policy resource.ContainerPolicy, requirement string, containerIdx int) jsonpatch.Operation {
	path := fmt.Sprintf("/spec/containers/%d/resources/%s/cpu", containerIdx, requirement)
	var newQuantity apiResource.Quantity
	switch requirement {
	case "requests":
		newQuantity = policy.NewResources(&pod.Spec.Containers[containerIdx]).Requests[corev1.ResourceCPU]
	case "limits":
		newQuantity = policy.NewResources(&pod.Spec.Containers[containerIdx]).Limits[corev1.ResourceCPU]
	default:
		panic("unsupported resource requirement")
	}
	return jsonpatch.Operation{
		Operation: "replace",
		Path:      path,
		Value:     newQuantity.String(),
	}
}
