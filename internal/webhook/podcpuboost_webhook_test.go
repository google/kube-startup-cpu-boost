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
		Describe("Webhook Admission Behavior", func() {
			BeforeEach(func() {
				pod = oneContainerBurstablePodTemplate.DeepCopy()
			})
			When("there is no matching Startup CPU Boost", func() {
				BeforeEach(func() {
					managerCall.Return(nil, false)
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
					boost                   *mock.MockStartupCPUBoost
					applyResourcePolicyCall *gomock.Call
				)
				BeforeEach(func() {
					boost = mock.NewMockStartupCPUBoost(mockCtrl)
					boost.EXPECT().Name().AnyTimes().Return("boost-one")
					managerCall.Return(boost, true)
					applyResourcePolicyCall = boost.EXPECT().ApplyResourcePolicy(
						gomock.Any(),
						gomock.Cond(func(x any) bool {
							p, ok := x.(*corev1.Pod)
							if !ok {
								return false
							}
							return p.Name == pod.Name && p.Namespace == pod.Namespace
						}),
						gomock.Cond(func(x any) bool {
							set, ok := x.(bpod.ContainerNameSet)
							if !ok || set == nil {
								return false
							}
							return set.Has(pod.Spec.Containers[0].Name)
						}),
					)
				})
				When("ApplyResourcePolicy makes no changes", func() {
					BeforeEach(func() {
						applyResourcePolicyCall.Return(false, nil)
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns zero patches", func() {
						Expect(response.Patches).To(HaveLen(0))
					})
				})
				When("ApplyResourcePolicy mutates the pod", func() {
					BeforeEach(func() {
						applyResourcePolicyCall.DoAndReturn(func(ctx context.Context, p *corev1.Pod, _ bpod.ContainerNameSet) (bool, error) {
							p.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = apiResource.MustParse("2")
							if p.Annotations == nil {
								p.Annotations = make(map[string]string)
							}
							p.Annotations[bpod.BoostAnnotationKey] = `{"state": "Active"}`

							if p.Labels == nil {
								p.Labels = make(map[string]string)
							}
							p.Labels[bpod.BoostLabelKey] = "boost-one"
							return true, nil
						})
					})
					It("allows the admission", func() {
						Expect(response.Allowed).To(BeTrue())
					})
					It("returns valid JSON patches", func() {
						Expect(response.Patches).To(ConsistOf(
							jsonpatch.Operation{
								Operation: "add",
								Path:      "/metadata/annotations",
								Value: map[string]interface{}{
									bpod.BoostAnnotationKey: `{"state": "Active"}`,
								},
							},
							jsonpatch.Operation{
								Operation: "add",
								Path:      "/metadata/labels",
								Value: map[string]interface{}{
									bpod.BoostLabelKey: "boost-one",
								},
							},
							jsonpatch.Operation{
								Operation: "replace",
								Path:      "/spec/containers/0/resources/requests/cpu",
								Value:     "2",
							},
						))
					})
				})
				When("ApplyResourcePolicy returns an error", func() {
					BeforeEach(func() {
						applyResourcePolicyCall.Return(false, fmt.Errorf("internal policy error"))
					})
					It("denies the admission with the error", func() {
						Expect(response.Allowed).To(BeFalse())
						Expect(response.Result.Message).To(ContainSubstring("internal policy error"))
					})
				})
			})
		})
	})
})
