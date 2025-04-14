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
	"fmt"

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
			mockCtrl                 *gomock.Controller
			manager                  *mock.MockManager
			managerCall              *gomock.Call
			pod                      *corev1.Pod
			response                 webhook.AdmissionResponse
			removeLimits             bool
			podLevelResourcesEnabled bool
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			manager = mock.NewMockManager(mockCtrl)
			managerCall = manager.EXPECT().GetCPUBoostForPod(
				gomock.Any(),
				gomock.Cond(func(x any) bool {
					p, ok := x.(*corev1.Pod)
					if !ok {
						return false
					}
					return p.Name == pod.Name && p.Namespace == pod.Namespace
				}),
			)
			podLevelResourcesEnabled = false
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
			hook := bwebhook.NewPodCPUBoostWebHook(manager, scheme.Scheme, removeLimits,
				podLevelResourcesEnabled)
			response = hook.Handle(context.TODO(), admissionReq)
		})
		Describe("for burstable POD with one container", func() {
			BeforeEach(func() {
				pod = oneContainerBurstablePodTemplate.DeepCopy()
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
				var (
					boostPolicyCall *gomock.Call
				)
				BeforeEach(func() {
					boost := mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					boostPolicyCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-one"))
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for a container", func() {
					boostPolicyCall.Times(1)
				})
				When("there is no policy for a container", func() {
					BeforeEach(func() {
						boostPolicyCall.Return(nil, false)
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns zero patches", func() {
						Expect(response.Patches).To(HaveLen(0))
					})
				})
				When("there is a policy for a container", func() {
					When("policy does not change QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicy := resource.NewPercentageContainerPolicy(120)
							boostPolicyCall.Return(resPolicy, true)
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns valid patches ", func() {
							Expect(response.Patches).To(ConsistOf(
								buildBoostLabelPatch("boost-one"),
								HaveBoostAnnotationPatch(pod.Spec.Containers),
								buildContainerResourcePatch(0, "requests", "replace", "1100m"),
								buildContainerResourcePatch(0, "limits", "replace", "2200m"),
							))
						})
						When("remove limits is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches ", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "1100m"),
									buildContainerResourcePatch(0, "limits", "remove", ""),
								))
							})
						})
					})
					When("policy changes QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicy := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("2"))
							boostPolicyCall.Return(resPolicy, true)
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns zero patches", func() {
							Expect(response.Patches).To(HaveLen(0))
						})
						When("remove limits is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns zero patches", func() {
								Expect(response.Patches).To(HaveLen(0))
							})
						})
					})
				})
			})
		})
		Describe("for guaranteed POD with one container", func() {
			BeforeEach(func() {
				pod = oneContainerGuaranteedPodTemplate.DeepCopy()
			})
			When("there is a matching Startup CPU Boost", func() {
				var (
					boostPolicyCall *gomock.Call
				)
				BeforeEach(func() {
					boost := mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					boostPolicyCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-one"))
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for a container", func() {
					boostPolicyCall.Times(1)
				})
				When("there is a policy for a container", func() {
					When("policy does not change QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicy := resource.NewPercentageContainerPolicy(120)
							boostPolicyCall.Return(resPolicy, true)
						})
						When("remove limits is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "2200m"),
									buildContainerResourcePatch(0, "limits", "replace", "2200m"),
								))
							})
						})
					})
					When("policy changes QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicy := resource.NewFixedPolicy(apiResource.MustParse("1"),
								apiResource.MustParse("2"))
							boostPolicyCall.Return(resPolicy, true)
						})
						When("remove limits is disabled", func() {
							BeforeEach(func() {
								removeLimits = false
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns zero patches", func() {
								Expect(response.Patches).To(HaveLen(0))
							})
						})
						When("remove limits is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns zero patches", func() {
								Expect(response.Patches).To(HaveLen(0))
							})
						})
					})
				})
			})
		})
		Describe("for burstable POD with two containers", func() {
			BeforeEach(func() {
				pod = twoContainerBurstablePodTemplate.DeepCopy()
			})
			When("there is a matching Startup CPU Boost", func() {
				var (
					boostPolicyOneCall *gomock.Call
					boostPolicyTwoCall *gomock.Call
				)
				BeforeEach(func() {
					boost := mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					boostPolicyOneCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-one"))
					boostPolicyTwoCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-two"))
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for a container", func() {
					boostPolicyOneCall.Times(1)
					boostPolicyTwoCall.Times(1)
				})
				When("there is no policy for any container", func() {
					BeforeEach(func() {
						boostPolicyOneCall.Return(nil, false)
						boostPolicyTwoCall.Return(nil, false)
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns zero patches", func() {
						Expect(response.Patches).To(HaveLen(0))
					})
				})
				When("there is a policy for one container", func() {
					BeforeEach(func() {
						resPolicy := resource.NewPercentageContainerPolicy(120)
						boostPolicyOneCall.Return(resPolicy, true)
						boostPolicyTwoCall.Return(nil, false)
					})
					When("remove limits is disabled", func() {
						BeforeEach(func() {
							removeLimits = false
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns valid patches", func() {
							Expect(response.Patches).To(ConsistOf(
								buildBoostLabelPatch("boost-one"),
								HaveBoostAnnotationPatch([]corev1.Container{pod.Spec.Containers[0]}),
								buildContainerResourcePatch(0, "requests", "replace", "1100m"),
								buildContainerResourcePatch(0, "limits", "replace", "2200m"),
							))
						})
					})
					When("remove limits is enabled", func() {
						BeforeEach(func() {
							removeLimits = true
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns valid patches", func() {
							Expect(response.Patches).To(ConsistOf(
								buildBoostLabelPatch("boost-one"),
								HaveBoostAnnotationPatch([]corev1.Container{pod.Spec.Containers[0]}),
								buildContainerResourcePatch(0, "requests", "replace", "1100m"),
								buildContainerResourcePatch(0, "limits", "remove", ""),
							))
						})
					})
				})
				When("there are policies for two containers", func() {
					When("policies do not change POD QoS class", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewPercentageContainerPolicy(120)
							resPolicyTwo := resource.NewPercentageContainerPolicy(120)
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(resPolicyTwo, true)
						})
						When("remove limit is not enabled", func() {
							BeforeEach(func() {
								removeLimits = false
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "1100m"),
									buildContainerResourcePatch(0, "limits", "replace", "2200m"),
									buildContainerResourcePatch(1, "requests", "replace", "1100m"),
									buildContainerResourcePatch(1, "limits", "replace", "2200m"),
								))
							})
						})
						When("remove limit is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "1100m"),
									buildContainerResourcePatch(0, "limits", "remove", ""),
									buildContainerResourcePatch(1, "requests", "replace", "1100m"),
									buildContainerResourcePatch(1, "limits", "remove", ""),
								))
							})
						})
					})
					When("policies change POD QoS class", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("2"))
							resPolicyTwo := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("2"))
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(resPolicyTwo, true)
						})
						When("remove limit is not enabled", func() {
							BeforeEach(func() {
								removeLimits = false
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(
										[]corev1.Container{pod.Spec.Containers[0]}),
									buildContainerResourcePatch(0, "requests", "replace", "2"),
									buildContainerResourcePatch(0, "limits", "replace", "2"),
								))
							})
						})
						When("remove limit is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "2"),
									buildContainerResourcePatch(0, "limits", "remove", ""),
									buildContainerResourcePatch(1, "requests", "replace", "2"),
									buildContainerResourcePatch(1, "limits", "remove", ""),
								))
							})
						})
					})
				})
			})
		})
		Describe("for guaranteed POD with two containers", func() {
			BeforeEach(func() {
				pod = twoContainerGuaranteedPodTemplate.DeepCopy()
			})
			When("there is a matching Startup CPU Boost", func() {
				var (
					boostPolicyOneCall *gomock.Call
					boostPolicyTwoCall *gomock.Call
				)
				BeforeEach(func() {
					boost := mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					boostPolicyOneCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-one"))
					boostPolicyTwoCall = boost.EXPECT().ResourcePolicy(gomock.Eq("container-two"))
					managerCall.Return(boost, true)
				})
				It("retrieves resource policy for a container", func() {
					boostPolicyOneCall.Times(1)
					boostPolicyTwoCall.Times(1)
				})
				When("there is a policy for one container", func() {
					When("policy does not change QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewPercentageContainerPolicy(120)
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(nil, false)
						})
						When("remove limit is not enabled", func() {
							BeforeEach(func() {
								removeLimits = false
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(
										[]corev1.Container{pod.Spec.Containers[0]}),
									buildContainerResourcePatch(0, "requests", "replace", "2200m"),
									buildContainerResourcePatch(0, "limits", "replace", "2200m"),
								))
							})
						})
						When("remove limit is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(
										[]corev1.Container{pod.Spec.Containers[0]}),
									buildContainerResourcePatch(0, "requests", "replace", "2200m"),
									buildContainerResourcePatch(0, "limits", "replace", "2200m"),
								))
							})
						})
					})
					When("policy changes QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("4"))
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(nil, false)
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns zero patches", func() {
							Expect(response.Patches).To(HaveLen(0))
						})
					})
				})
				When("there is a policy for two containers", func() {
					When("policies do not change QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewPercentageContainerPolicy(120)
							resPolicyTwo := resource.NewPercentageContainerPolicy(120)
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(resPolicyTwo, true)
						})
						When("remove limit is not enabled", func() {
							BeforeEach(func() {
								removeLimits = false
							})
						})
						When("remove limit is enabled", func() {
							BeforeEach(func() {
								removeLimits = true
							})
							It("allows the admission", func() {
								Expect(response.Allowed).To(BeTrue())
							})
							It("returns valid patches", func() {
								Expect(response.Patches).To(ConsistOf(
									buildBoostLabelPatch("boost-one"),
									HaveBoostAnnotationPatch(pod.Spec.Containers),
									buildContainerResourcePatch(0, "requests", "replace", "2200m"),
									buildContainerResourcePatch(0, "limits", "replace", "2200m"),
									buildContainerResourcePatch(1, "requests", "replace", "2200m"),
									buildContainerResourcePatch(1, "limits", "replace", "2200m"),
								))
							})
						})
					})
					When("policies change QoS class of a POD", func() {
						BeforeEach(func() {
							resPolicyOne := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("4"))
							resPolicyTwo := resource.NewFixedPolicy(apiResource.MustParse("2"),
								apiResource.MustParse("4"))
							boostPolicyOneCall.Return(resPolicyOne, true)
							boostPolicyTwoCall.Return(resPolicyTwo, true)
						})
						It("allows the admission", func() {
							Expect(response.Allowed).To(BeTrue())
						})
						It("returns zero patches", func() {
							Expect(response.Patches).To(HaveLen(0))
						})
					})
				})
			})
		})
	})
})

func buildBoostLabelPatch(boostName string) jsonpatch.Operation {
	return jsonpatch.Operation{
		Operation: "add",
		Path:      "/metadata/labels",
		Value: map[string]interface{}{
			bpod.BoostLabelKey: boostName,
		},
	}
}

func buildContainerResourcePatch(containerIdx int, requirement string, operation string,
	value string) jsonpatch.Operation {
	var valueToSet interface{}
	valueToSet = value
	if value == "" {
		valueToSet = nil
	}
	return jsonpatch.Operation{
		Operation: operation,
		Path:      fmt.Sprintf("/spec/containers/%d/resources/%s/cpu", containerIdx, requirement),
		Value:     valueToSet,
	}
}
