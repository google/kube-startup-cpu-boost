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

package boost_test

import (
	"context"
	"errors"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("StartupCPUBoost", func() {
	var (
		spec       *autoscaling.StartupCPUBoost
		config     *cpuboost.StartupCPUBoostConfig
		boost      cpuboost.StartupCPUBoost
		err        error
		pod        *corev1.Pod
		mockCtrl   *gomock.Controller
		mockClient *mock.MockClient
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		pod = podTemplate.DeepCopy()
		spec = specTemplate.DeepCopy()
		config = &cpuboost.StartupCPUBoostConfig{
			Client: mockClient,
		}
		metrics.ClearBoostMetrics(spec.Namespace, spec.Name)
	})
	Describe("Instantiates from the API specification", func() {
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(spec, config)
		})
		It("does not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
		It("returns valid name", func() {
			Expect(boost.Name()).To(Equal(spec.Name))
		})
		It("returns valid namespace", func() {
			Expect(boost.Namespace()).To(Equal(spec.Namespace))
		})
		When("the spec has resource policy for containers", func() {
			var (
				containerOneName            = "container-one"
				containerTwoName            = "container-two"
				containerOnePercValue int64 = 120
				containerTwoFixedReq        = apiResource.MustParse("1")
				containerTwoFixedLim        = apiResource.MustParse("2")
			)
			Context("with deprecated container name policy", func() {
				BeforeEach(func() {
					spec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
						ContainerPolicies: []autoscaling.ContainerPolicy{
							{
								ContainerName: containerOneName,
								PercentageIncrease: &autoscaling.PercentageIncrease{
									Value: containerOnePercValue,
								},
							},
							{
								ContainerName: containerTwoName,
								FixedResources: &autoscaling.FixedResources{
									Requests: containerTwoFixedReq,
									Limits:   containerTwoFixedLim,
								},
							},
						},
					}
				})
				It("does not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})
			Context("with match containers policy", func() {
				BeforeEach(func() {
					spec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
						ContainerPolicies: []autoscaling.ContainerPolicy{
							{
								MatchContainers: &autoscaling.MatchContainers{
									Type:  autoscaling.MatchContainersTypeExactName,
									Value: containerOneName,
								},
								PercentageIncrease: &autoscaling.PercentageIncrease{
									Value: containerOnePercValue,
								},
							},
							{
								MatchContainers: &autoscaling.MatchContainers{
									Type:  autoscaling.MatchContainersTypeRegexName,
									Value: "^container-two$",
								},
								FixedResources: &autoscaling.FixedResources{
									Requests: containerTwoFixedReq,
									Limits:   containerTwoFixedLim,
								},
							},
						},
					}
				})
				It("does not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
		When("the spec has container policy without resource policy", func() {
			BeforeEach(func() {
				spec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
					ContainerPolicies: []autoscaling.ContainerPolicy{
						{
							MatchContainers: &autoscaling.MatchContainers{
								Type:  autoscaling.MatchContainersTypeExactName,
								Value: "container-one",
							},
						},
					},
				}
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("the spec has container policy with two resource policies", func() {
			BeforeEach(func() {
				spec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
					ContainerPolicies: []autoscaling.ContainerPolicy{
						{
							MatchContainers: &autoscaling.MatchContainers{
								Type:  autoscaling.MatchContainersTypeExactName,
								Value: "container-one",
							},
						},
					},
				}
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("the spec has fixed duration policy", func() {
			BeforeEach(func() {
				spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitSec,
					Value: 123,
				}
			})
			It("returns fixed duration policy implementation", func() {
				Expect(boost.DurationPolicies()).To(HaveKey(duration.FixedDurationPolicyName))
			})
			It("returned fixed duration policy implementation is valid", func() {
				p := boost.DurationPolicies()[duration.FixedDurationPolicyName]
				fixedP, ok := p.(*duration.FixedDurationPolicy)
				Expect(ok).To(BeTrue())
				expDuration := time.Duration(spec.Spec.DurationPolicy.Fixed.Value) * time.Second
				Expect(fixedP.Duration()).To(Equal(expDuration))
			})
		})
		When("the spec has pod condition duration policy", func() {
			BeforeEach(func() {
				spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitSec,
					Value: 123,
				}
				spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				}
			})
			It("returns pod condition duration policy implementation", func() {
				Expect(boost.DurationPolicies()).To(HaveKey(duration.PodConditionPolicyName))
			})
			It("returned pod condition duration policy implementation is valid", func() {
				p := boost.DurationPolicies()[duration.PodConditionPolicyName]
				podCondP, ok := p.(*duration.PodConditionPolicy)
				Expect(ok).To(BeTrue())
				Expect(podCondP.Condition()).To(Equal(spec.Spec.DurationPolicy.PodCondition.Type))
				Expect(podCondP.Status()).To(Equal(spec.Spec.DurationPolicy.PodCondition.Status))
			})
		})
	})
	Describe("Handles POD upsert triggering events", func() {
		Context("when boost spec has no condition policy defined", func() {
			When("POD does not exist", func() {
				DescribeTable("adds POD, updates stats and metrics",
					func(ctx context.Context, eventType bpod.PodEventType) {
						boost, err := cpuboost.NewStartupCPUBoost(spec, config)
						Expect(err).NotTo(HaveOccurred())

						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: eventType,
							Pod:  pod,
						})

						Expect(err).NotTo(HaveOccurred())
						_, found := boost.Pod(pod.Name)
						Expect(found).To(BeTrue())
						stats := boost.Stats()
						Expect(stats.ActiveContainerBoosts).To(Equal(2))
						Expect(stats.TotalContainerBoosts).To(Equal(2))
						Expect(metrics.BoostContainersActive(boost.Namespace(), boost.Name())).To(Equal(float64(2)))
						Expect(metrics.BoostContainersTotal(boost.Namespace(), boost.Name())).To(Equal(float64(2)))
					},
					Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated),
					Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged),
				)
			})
			When("POD already exists", func() {
				DescribeTable("updates POD, stats and metrics",
					func(ctx context.Context, eventType bpod.PodEventType) {
						boost, err := cpuboost.NewStartupCPUBoost(spec, config)
						Expect(err).NotTo(HaveOccurred())
						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypePodCreated,
							Pod:  pod,
						})
						Expect(err).NotTo(HaveOccurred())
						updatedCreationTimestamp := metav1.NewTime(time.Now())
						updatedPod := pod.DeepCopy()
						updatedPod.CreationTimestamp = updatedCreationTimestamp

						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: eventType,
							Pod:  updatedPod,
						})

						Expect(err).NotTo(HaveOccurred())
						storedPod, found := boost.Pod(pod.Name)
						Expect(found).To(BeTrue())
						Expect(storedPod.CreationTimestamp).To(Equal(updatedCreationTimestamp))
						stats := boost.Stats()
						Expect(stats.ActiveContainerBoosts).To(Equal(2))
						Expect(stats.TotalContainerBoosts).To(Equal(2))
						Expect(metrics.BoostContainersActive(boost.Namespace(), boost.Name())).To(Equal(float64(2)))
						Expect(metrics.BoostContainersTotal(boost.Namespace(), boost.Name())).To(Equal(float64(2)))
					},
					Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated),
					Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged),
				)
				When("Pod conditions contain resize conditions", func() {
					It("inspects and handles PodResizePending and PodResizeInProgress", func(ctx context.Context) {
						boost, err := cpuboost.NewStartupCPUBoost(spec, config)
						Expect(err).NotTo(HaveOccurred())

						// First register pod
						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypePodCreated,
							Pod:  pod,
						})
						Expect(err).NotTo(HaveOccurred())

						// Update pod with PodResizePending Deferred
						pendingPod := pod.DeepCopy()
						pendingPod.Status.Conditions = []corev1.PodCondition{
							{
								Type:    cpuboost.PodConditionPodResizePending,
								Status:  corev1.ConditionTrue,
								Reason:  "Deferred",
								Message: "Node didn't have enough resource: cpu",
							},
						}
						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypeConditionChanged,
							Pod:  pendingPod,
						})
						Expect(err).NotTo(HaveOccurred())

						// Update pod with PodResizeInProgress
						inProgressPod := pendingPod.DeepCopy()
						inProgressPod.Status.Conditions = []corev1.PodCondition{
							{
								Type:    cpuboost.PodConditionPodResizeInProgress,
								Status:  corev1.ConditionTrue,
								Message: "Actuating container resize",
							},
						}
						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypeConditionChanged,
							Pod:  inProgressPod,
						})
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})
			Context("when boost spec has condition policy defined", func() {
				var (
					spec *autoscaling.StartupCPUBoost
				)
				BeforeEach(func() {
					spec = specTemplate.DeepCopy()
					spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					}
				})
				When("POD condition doesn't match the policy", func() {
					DescribeTable("registers POD but skips resource reversion",
						func(ctx context.Context, eventType bpod.PodEventType) {
							pod := podTemplate.DeepCopy()
							pod.Status.Conditions = []corev1.PodCondition{{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse,
							}}
							mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
							mockClient := mock.NewMockClient(mockCtrl)
							mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
								gomock.Any()).Return(nil).Times(0)
							mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(0)
							mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
								gomock.Any()).Return(nil).Times(0)
							config.Client = mockClient
							boost, err = cpuboost.NewStartupCPUBoost(spec, config)
							Expect(err).NotTo(HaveOccurred())

							err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
								Type: eventType,
								Pod:  pod,
							})

							Expect(err).NotTo(HaveOccurred())
							_, found := boost.Pod(pod.Name)
							Expect(found).To(BeTrue())
						},
						Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated),
						Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged),
					)
				})
				When("POD condition matches the policy", func() {
					Context("using normal revert mode", func() {
						DescribeTable("reverts resources with resize sub-resource",
							func(ctx context.Context, eventType bpod.PodEventType, boostOnRestart bool) {
								pod := podTemplate.DeepCopy()
								pod.Status.Conditions = []corev1.PodCondition{{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								}}
								config.BoostOnRestart = boostOnRestart

								mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
								mockClient := mock.NewMockClient(mockCtrl)
								mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
									gomock.Any()).Return(nil).Times(1)
								mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(1)

								mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
									gomock.Any()).Return(nil).Times(1)

								config.Client = mockClient
								boost, err = cpuboost.NewStartupCPUBoost(spec, config)
								Expect(err).NotTo(HaveOccurred())

								err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
									Type: eventType,
									Pod:  pod,
								})

								Expect(err).NotTo(HaveOccurred())
							},
							Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated, false),
							Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged, false),
							Entry("via PodCreatedEvent with boostOnRestart", bpod.PodEventTypePodCreated, true),
							Entry("via ConditionChanged event with boostOnRestart", bpod.PodEventTypeConditionChanged, true),
						)
					})
					Context("using legacy revert mode", func() {
						DescribeTable("reverts resources with pod update",
							func(ctx context.Context, eventType bpod.PodEventType, boostOnRestart bool) {
								pod := podTemplate.DeepCopy()
								pod.Status.Conditions = []corev1.PodCondition{{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								}}
								config.BoostOnRestart = boostOnRestart
								config.LegacyRevertMode = true

								mockClient := mock.NewMockClient(mockCtrl)
								mockClient.EXPECT().Update(gomock.Any(), gomock.Eq(pod)).Return(nil).Times(1)
								config.Client = mockClient

								boost, err = cpuboost.NewStartupCPUBoost(spec, config)
								Expect(err).NotTo(HaveOccurred())

								err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
									Type: eventType,
									Pod:  pod,
								})

								Expect(err).NotTo(HaveOccurred())
							},
							Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated, false),
							Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged, false),
							Entry("via PodCreatedEvent with boostOnRestart", bpod.PodEventTypePodCreated, true),
							Entry("via ConditionChanged event with boostOnRestart", bpod.PodEventTypeConditionChanged, true),
						)
					})
				})
			})
		})
	})
	Describe("Handles POD deleted event", func() {
		When("POD exists", func() {
			It("removes POD, updates stats and metrics", func(ctx context.Context) {
				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).NotTo(HaveOccurred())
				err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
					Type: bpod.PodEventTypePodCreated,
					Pod:  pod,
				})
				Expect(err).NotTo(HaveOccurred())

				err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
					Type: bpod.PodEventTypePodDeleted,
					Pod:  pod,
				})

				Expect(err).NotTo(HaveOccurred())
				_, found := boost.Pod(pod.Name)
				Expect(found).To(BeFalse())
				stats := boost.Stats()
				Expect(stats.ActiveContainerBoosts).To(Equal(0))
				Expect(stats.TotalContainerBoosts).To(Equal(2))
				Expect(metrics.BoostContainersActive(boost.Namespace(), boost.Name())).To(Equal(float64(0)))
				Expect(metrics.BoostContainersTotal(boost.Namespace(), boost.Name())).To(Equal(float64(2)))
			})
		})
	})
	Describe("Handles POD container restarting event", func() {
		When("boost on restart is disabled", func() {
			BeforeEach(func() {
				config.BoostOnRestart = false
			})
			It("does not modify the pod", func(ctx context.Context) {
				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).NotTo(HaveOccurred())

				annot, _ := bpod.BoostAnnotationFromPod(pod)
				annot.State = bpod.BoostStateReverted
				annot.Apply(pod)
				err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
					Type:                     bpod.PodEventTypeContainerRestarting,
					Pod:                      pod,
					RestartingContainerNames: []string{"container-one"},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1"))
			})
		})
		When("boost on restart is enabled", func() {
			BeforeEach(func() {
				config.BoostOnRestart = true
				setContainerPercentagePolicy(spec, "container-one", 100)
			})
			Context("with legacy resize", func() {
				BeforeEach(func() {
					config.LegacyRevertMode = true
				})
				It("applies the boost and updates the pod", func(ctx context.Context) {
					mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
							p := obj.(*corev1.Pod)
							Expect(p.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("2"))
							Expect(p.Labels).To(HaveKey(bpod.BoostLabelKey))
							Expect(p.Annotations).To(HaveKey(bpod.BoostAnnotationKey))
							return nil
						},
					)
					config.Client = mockClient

					boost, err := cpuboost.NewStartupCPUBoost(spec, config)
					Expect(err).NotTo(HaveOccurred())

					annot, _ := bpod.BoostAnnotationFromPod(pod)
					annot.State = bpod.BoostStateReverted
					annot.Apply(pod)
					err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
						Type:                     bpod.PodEventTypeContainerRestarting,
						Pod:                      pod,
						RestartingContainerNames: []string{"container-one"},
					})
					Expect(err).NotTo(HaveOccurred())
					trackedPod, ok := boost.Pod(pod.Name)
					Expect(ok).To(BeTrue())
					Expect(trackedPod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("2"))
				})
			})
			Context("with sub-resource resize", func() {
				BeforeEach(func() {
					config.LegacyRevertMode = false
				})
				It("applies the boost and patches the pod", func(ctx context.Context) {
					mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
					mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
							Expect(patch.Type()).To(Equal(types.MergePatchType))
							data, err := patch.Data(obj)
							Expect(err).NotTo(HaveOccurred())
							Expect(string(data)).To(ContainSubstring(`"cpu":"2"`))
							return nil
						},
					)
					mockClient.EXPECT().SubResource(gomock.Eq("resize")).Return(mockSubResourceClient)
					mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
						func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
							Expect(patch.Type()).To(Equal(types.MergePatchType))
							data, err := patch.Data(obj)
							Expect(err).NotTo(HaveOccurred())
							Expect(string(data)).To(ContainSubstring(bpod.BoostLabelKey))
							Expect(string(data)).To(ContainSubstring(bpod.BoostAnnotationKey))
							return nil
						},
					)
					config.Client = mockClient

					boost, err := cpuboost.NewStartupCPUBoost(spec, config)
					Expect(err).NotTo(HaveOccurred())

					annot, _ := bpod.BoostAnnotationFromPod(pod)
					annot.State = bpod.BoostStateReverted
					annot.Apply(pod)
					err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
						Type:                     bpod.PodEventTypeContainerRestarting,
						Pod:                      pod,
						RestartingContainerNames: []string{"container-one"},
					})
					Expect(err).NotTo(HaveOccurred())
					trackedPod, ok := boost.Pod(pod.Name)
					Expect(ok).To(BeTrue())
					Expect(trackedPod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("2"))
				})
				When("patching pod resize fails", func() {
					It("returns error and does not update tracked pod to active", func(ctx context.Context) {
						mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
						mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("node didn't have enough allocatable resources"))
						mockClient.EXPECT().SubResource(gomock.Eq("resize")).Return(mockSubResourceClient)
						config.Client = mockClient

						boost, err := cpuboost.NewStartupCPUBoost(spec, config)
						Expect(err).NotTo(HaveOccurred())

						annot, _ := bpod.BoostAnnotationFromPod(pod)
						annot.State = bpod.BoostStateReverted
						annot.Apply(pod)
						// Initially register pod in boost
						_ = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypePodCreated,
							Pod:  pod,
						})

						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type:                     bpod.PodEventTypeContainerRestarting,
							Pod:                      pod,
							RestartingContainerNames: []string{"container-one"},
						})
						Expect(err).To(HaveOccurred())
						trackedPod, ok := boost.Pod(pod.Name)
						Expect(ok).To(BeTrue())
						trackedAnnot, err := bpod.BoostAnnotationFromPod(trackedPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(trackedAnnot.State).To(Equal(bpod.BoostStateReverted))
					})
				})
				When("all containers are skipped (e.g. QoS change) on restart", func() {
					It("does not patch pod or transition state to active", func(ctx context.Context) {
						// Set policy that would change QoS from Guaranteed to Burstable (e.g. only setting requests)
						spec.Spec.ResourcePolicy.ContainerPolicies = []autoscaling.ContainerPolicy{
							{
								ContainerName: "container-one",
								FixedResources: &autoscaling.FixedResources{
									Requests: apiResource.MustParse("2"),
								},
							},
						}
						// pod has Guaranteed QoS (limits == requests for both cpu and memory)
						pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						}
						pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
							corev1.ResourceCPU:    apiResource.MustParse("1"),
							corev1.ResourceMemory: apiResource.MustParse("1Gi"),
						}
						pod.Spec.Containers = []corev1.Container{pod.Spec.Containers[0]}

						mockClient.EXPECT().SubResource(gomock.Any()).Times(0)
						mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
						config.Client = mockClient

						boost, err := cpuboost.NewStartupCPUBoost(spec, config)
						Expect(err).NotTo(HaveOccurred())

						annot := bpod.NewBoostAnnotation()
						annot.State = bpod.BoostStateReverted
						annot.UpdateInitResources("container-one", pod.Spec.Containers[0].Resources)
						annot.Apply(pod)

						_ = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type: bpod.PodEventTypePodCreated,
							Pod:  pod,
						})

						err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
							Type:                     bpod.PodEventTypeContainerRestarting,
							Pod:                      pod,
							RestartingContainerNames: []string{"container-one"},
						})
						Expect(err).NotTo(HaveOccurred())
						trackedPod, ok := boost.Pod(pod.Name)
						Expect(ok).To(BeTrue())
						trackedAnnot, err := bpod.BoostAnnotationFromPod(trackedPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(trackedAnnot.State).To(Equal(bpod.BoostStateReverted))
					})
				})
			})
		})
	})
	Describe("Updates boost from the spec", func() {
		var (
			updatedSpec *autoscaling.StartupCPUBoost
		)
		BeforeEach(func() {
			spec.Selector = metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			}
			spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
				Unit:  autoscaling.FixedDurationPolicyUnitMin,
				Value: 2,
			}
			spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
				Status: corev1.ConditionTrue,
				Type:   corev1.PodReady,
			}
			updatedSpec = spec.DeepCopy()
		})
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(spec, config)
			Expect(err).ShouldNot(HaveOccurred())
			err = boost.UpdateFromSpec(context.TODO(), updatedSpec)
		})
		When("selector is changed", func() {
			var (
				podToSelect *corev1.Pod
			)
			BeforeEach(func() {
				updatedSpec.Selector = metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "newApp",
					},
				}
				podToSelect = &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: specTemplate.Namespace,
						Labels: map[string]string{
							"app": "newApp",
						}}}
			})
			It("matches pod with new selector", func() {
				Expect(boost.Matches(podToSelect)).To(BeTrue())
			})
		})
		When("duration policy is changed", func() {
			var (
				durationPolicies map[string]duration.Policy
			)
			BeforeEach(func() {
				updatedSpec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
					Unit:  autoscaling.FixedDurationPolicyUnitMin,
					Value: 1000,
				}
				updatedSpec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
					Type:   corev1.PodInitialized,
					Status: corev1.ConditionTrue,
				}
			})
			JustBeforeEach(func() {
				durationPolicies = boost.DurationPolicies()
			})
			It("has valid fixed duration policy", func() {
				durationPolicy := durationPolicies[duration.FixedDurationPolicyName]
				Expect(durationPolicy).To(BeAssignableToTypeOf(&duration.FixedDurationPolicy{}))
				fixedDurationPolicy := durationPolicy.(*duration.FixedDurationPolicy)
				Expect(fixedDurationPolicy.Duration()).To(Equal(1000 * time.Minute))
			})
			It("has valid pod condition policy", func() {
				durationPolicy := durationPolicies[duration.PodConditionPolicyName]
				Expect(durationPolicy).To(BeAssignableToTypeOf(&duration.PodConditionPolicy{}))
				podConditionDurationPolicy := durationPolicy.(*duration.PodConditionPolicy)
				Expect(podConditionDurationPolicy.Condition()).To(Equal(corev1.PodInitialized))
				Expect(podConditionDurationPolicy.Status()).To(Equal(corev1.ConditionTrue))
			})
		})
		When("resource policy is changed", func() {
			BeforeEach(func() {
				updatedSpec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
					ContainerPolicies: []autoscaling.ContainerPolicy{
						{
							MatchContainers: &autoscaling.MatchContainers{
								Type:  autoscaling.MatchContainersTypeExactName,
								Value: "test",
							},
							PercentageIncrease: &autoscaling.PercentageIncrease{
								Value: 1000,
							},
						},
					},
				}

			})
			JustBeforeEach(func() {
				err = boost.UpdateFromSpec(context.TODO(), updatedSpec)
			})
			It("applies updated resource policy", func() {
				Expect(err).NotTo(HaveOccurred())

				pod := podTemplate.DeepCopy()
				delete(pod.Annotations, bpod.BoostAnnotationKey)
				pod.Spec.Containers[0].Name = "test"
				setContainerResource(pod, 0, corev1.ResourceCPU, "1", "2")

				_, err = boost.ApplyResourcePolicy(context.Background(), pod, nil)
				Expect(err).NotTo(HaveOccurred())

				// 1 CPU * 1000% = 11 CPU (10 + 1) -> 11000m
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(11000)))
			})
		})
	})
	Describe("ApplyResourcePolicy", func() {
		When("POD boost status forbids boosting", func() {
			DescribeTable("Does not change POD",
				func(status string) {
					pod := podTemplate.DeepCopy()
					if pod.Annotations == nil {
						pod.Annotations = make(map[string]string)
					}
					annotation := &bpod.BoostPodAnnotation{State: status}
					pod.Annotations[bpod.BoostAnnotationKey] = annotation.ToJSON()
					originalPod := pod.DeepCopy()

					configSpec := specTemplate.DeepCopy()
					setContainerPercentagePolicy(configSpec, "container-one", 100)
					boost, err := cpuboost.NewStartupCPUBoost(configSpec, config)
					Expect(err).NotTo(HaveOccurred())

					boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

					Expect(err).NotTo(HaveOccurred())
					Expect(boosted).To(BeFalse())
					Expect(pod.Spec.Containers).To(Equal(originalPod.Spec.Containers))
				},
				Entry("when status is active", bpod.BoostStateActive),
				Entry("when status is infeasible", bpod.BoostStateInfeasible),
			)
		})
		When("POD has no containers that match policy", func() {
			It("Does not change POD", func() {
				pod := podTemplate.DeepCopy()
				delete(pod.Annotations, bpod.BoostAnnotationKey)
				originalPod := pod.DeepCopy()

				configSpec := specTemplate.DeepCopy()
				setContainerPercentagePolicy(configSpec, "non-existent-container", 100)
				boost, err := cpuboost.NewStartupCPUBoost(configSpec, config)
				Expect(err).NotTo(HaveOccurred())

				boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(boosted).To(BeFalse())
				Expect(pod.Spec.Containers).To(Equal(originalPod.Spec.Containers))
				_, ok := pod.Annotations[bpod.BoostAnnotationKey]
				Expect(ok).To(BeFalse())
			})
		})
		When("POD has containers that match policy", func() {
			When("container has require restart resize policy", func() {
				It("does not increase container CPU resources", func() {
					pod := podTemplate.DeepCopy()
					delete(pod.Annotations, bpod.BoostAnnotationKey)
					pod.Spec.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{
						{
							ResourceName:  corev1.ResourceCPU,
							RestartPolicy: corev1.RestartContainer,
						},
					}
					originalPod := pod.DeepCopy()

					configSpec := specTemplate.DeepCopy()
					setContainerPercentagePolicy(configSpec, "container-one", 100)
					boost, err := cpuboost.NewStartupCPUBoost(configSpec, config)
					Expect(err).NotTo(HaveOccurred())

					boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

					Expect(err).NotTo(HaveOccurred())
					Expect(boosted).To(BeFalse())
					Expect(pod.Spec.Containers).To(Equal(originalPod.Spec.Containers))
					_, ok := pod.Annotations[bpod.BoostAnnotationKey]
					Expect(ok).To(BeFalse())
				})
			})
			When("container has no CPU resources", func() {
				It("does not change container resources", func() {
					pod := podTemplate.DeepCopy()
					delete(pod.Annotations, bpod.BoostAnnotationKey)
					pod.Spec.Containers[0].Resources.Requests = nil
					pod.Spec.Containers[0].Resources.Limits = nil
					originalPod := pod.DeepCopy()

					configSpec := specTemplate.DeepCopy()
					setContainerPercentagePolicy(configSpec, "container-one", 100)
					boost, err := cpuboost.NewStartupCPUBoost(configSpec, config)
					Expect(err).NotTo(HaveOccurred())

					boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

					Expect(err).NotTo(HaveOccurred())
					Expect(boosted).To(BeFalse())
					Expect(pod.Spec.Containers).To(Equal(originalPod.Spec.Containers))
					_, ok := pod.Annotations[bpod.BoostAnnotationKey]
					Expect(ok).To(BeFalse())
				})
			})
			When("container resource increase changes POD's QOS class", func() {
				It("does not change container resources", func() {
					pod := podTemplate.DeepCopy()
					delete(pod.Annotations, bpod.BoostAnnotationKey)
					setContainerResource(pod, 0, corev1.ResourceCPU, "1", "2")
					setContainerResource(pod, 0, corev1.ResourceMemory, "1Gi", "1Gi")
					setContainerResource(pod, 1, corev1.ResourceCPU, "2", "2")
					setContainerResource(pod, 1, corev1.ResourceMemory, "1Gi", "1Gi")
					originalPod := pod.DeepCopy()

					configSpec := specTemplate.DeepCopy()
					setContainerFixedPolicy(configSpec, "container-one", "2", "2")
					boost, err := cpuboost.NewStartupCPUBoost(configSpec, config)
					Expect(err).NotTo(HaveOccurred())

					boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

					Expect(err).NotTo(HaveOccurred())
					Expect(boosted).To(BeFalse())
					Expect(pod.Spec.Containers).To(Equal(originalPod.Spec.Containers))
					_, ok := pod.Annotations[bpod.BoostAnnotationKey]
					Expect(ok).To(BeFalse())
				})
			})
			When("container meets all requirements for resource increase", func() {
				Context("with CPU limits removal disabled", func() {
					It("increases CPU requests and limits and applies POD metadata", func() {
						pod := podTemplate.DeepCopy()
						delete(pod.Annotations, bpod.BoostAnnotationKey)
						delete(pod.Labels, bpod.BoostLabelKey)
						setContainerResource(pod, 0, corev1.ResourceCPU, "1", "2")

						configSpec := specTemplate.DeepCopy()
						setContainerPercentagePolicy(configSpec, "container-one", 100)

						configVal := *config
						configVal.RemoveLimitsEnabled = false
						boost, err := cpuboost.NewStartupCPUBoost(configSpec, &configVal)
						Expect(err).NotTo(HaveOccurred())

						boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

						Expect(err).NotTo(HaveOccurred())
						Expect(boosted).To(BeTrue())
						Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
						Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().MilliValue()).To(Equal(int64(4000)))
						Expect(pod.Spec.Containers[1].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1000))) // Unmodified
						Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().MilliValue()).To(Equal(int64(2000)))   // Unmodified
						Expect(pod.Labels[bpod.BoostLabelKey]).To(Equal("boost-001"))
						annot, err := bpod.BoostAnnotationFromPod(pod)
						Expect(err).NotTo(HaveOccurred())
						Expect(annot.State).To(Equal(bpod.BoostStateActive))
						Expect(annot.InitCPURequests["container-one"]).To(Equal("1"))
						Expect(annot.InitCPULimits["container-one"]).To(Equal("2"))
					})
				})
				Context("with regex container match policy", func() {
					It("increases CPU requests and limits for matching containers", func() {
						pod := podTemplate.DeepCopy()
						delete(pod.Annotations, bpod.BoostAnnotationKey)
						delete(pod.Labels, bpod.BoostLabelKey)
						setContainerResource(pod, 0, corev1.ResourceCPU, "1", "2")
						setContainerResource(pod, 1, corev1.ResourceCPU, "1", "2")

						configSpec := specTemplate.DeepCopy()
						configSpec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
							ContainerPolicies: []autoscaling.ContainerPolicy{
								{
									MatchContainers: &autoscaling.MatchContainers{
										Type:  autoscaling.MatchContainersTypeRegexName,
										Value: "^container-.*$",
									},
									PercentageIncrease: &autoscaling.PercentageIncrease{Value: 100},
								},
							},
						}

						configVal := *config
						configVal.RemoveLimitsEnabled = false
						boost, err := cpuboost.NewStartupCPUBoost(configSpec, &configVal)
						Expect(err).NotTo(HaveOccurred())

						boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

						Expect(err).NotTo(HaveOccurred())
						Expect(boosted).To(BeTrue())
						Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
						Expect(pod.Spec.Containers[1].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
					})
				})
				Context("with CPU limits removal enabled", func() {
					When("limit removal does not change POD QoS class", func() {
						It("increases CPU requests and removes limit and applies POD metadata", func() {
							pod := podTemplate.DeepCopy()
							delete(pod.Annotations, bpod.BoostAnnotationKey)
							delete(pod.Labels, bpod.BoostLabelKey)
							setContainerResource(pod, 0, corev1.ResourceCPU, "1", "2")
							setContainerResource(pod, 1, corev1.ResourceCPU, "1", "")

							configSpec := specTemplate.DeepCopy()
							setContainerPercentagePolicy(configSpec, "container-one", 100)

							configVal := *config
							configVal.RemoveLimitsEnabled = true
							boost, err := cpuboost.NewStartupCPUBoost(configSpec, &configVal)
							Expect(err).NotTo(HaveOccurred())

							boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

							Expect(err).NotTo(HaveOccurred())
							Expect(boosted).To(BeTrue())
							Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
							Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().IsZero()).To(BeTrue())
							Expect(pod.Spec.Containers[1].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1000)))
							Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().IsZero()).To(BeTrue())
							Expect(pod.Labels[bpod.BoostLabelKey]).To(Equal("boost-001"))
							annot, err := bpod.BoostAnnotationFromPod(pod)
							Expect(err).NotTo(HaveOccurred())
							Expect(annot.State).To(Equal(bpod.BoostStateActive))
							Expect(annot.InitCPURequests["container-one"]).To(Equal("1"))
							Expect(annot.InitCPULimits["container-one"]).To(Equal("2"))
						})
					})
					When("POD is Guaranteed (limits removal is skipped)", func() {
						It("increases CPU requests and limits but does not remove limit", func() {
							pod := podTemplate.DeepCopy()
							delete(pod.Annotations, bpod.BoostAnnotationKey)
							delete(pod.Labels, bpod.BoostLabelKey)
							setContainerResource(pod, 0, corev1.ResourceCPU, "1", "1")
							setContainerResource(pod, 0, corev1.ResourceMemory, "1Gi", "1Gi")
							setContainerResource(pod, 1, corev1.ResourceCPU, "1", "1")
							setContainerResource(pod, 1, corev1.ResourceMemory, "1Gi", "1Gi")

							configSpec := specTemplate.DeepCopy()
							setContainerPercentagePolicy(configSpec, "container-one", 100)

							configVal := *config
							configVal.RemoveLimitsEnabled = true
							boost, err := cpuboost.NewStartupCPUBoost(configSpec, &configVal)
							Expect(err).NotTo(HaveOccurred())

							boosted, err := boost.ApplyResourcePolicy(context.Background(), pod, nil)

							Expect(err).NotTo(HaveOccurred())
							Expect(boosted).To(BeTrue())
							Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(2000)))
							Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().MilliValue()).To(Equal(int64(2000)))
							Expect(pod.Spec.Containers[1].Resources.Requests.Cpu().MilliValue()).To(Equal(int64(1000)))
							Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().MilliValue()).To(Equal(int64(1000)))
							Expect(pod.Labels[bpod.BoostLabelKey]).To(Equal("boost-001"))
							annot, err := bpod.BoostAnnotationFromPod(pod)
							Expect(err).NotTo(HaveOccurred())
							Expect(annot.State).To(Equal(bpod.BoostStateActive))
							Expect(annot.InitCPURequests["container-one"]).To(Equal("1"))
							Expect(annot.InitCPULimits["container-one"]).To(Equal("1"))
						})
					})
				})
			})
		})
	})
})

func setContainerResource(pod *corev1.Pod, containerIdx int, res corev1.ResourceName, req, lim string) {
	if pod.Spec.Containers[containerIdx].Resources.Requests == nil {
		pod.Spec.Containers[containerIdx].Resources.Requests = make(corev1.ResourceList)
	}
	if pod.Spec.Containers[containerIdx].Resources.Limits == nil {
		pod.Spec.Containers[containerIdx].Resources.Limits = make(corev1.ResourceList)
	}
	if req != "" {
		pod.Spec.Containers[containerIdx].Resources.Requests[res] = apiResource.MustParse(req)
	} else {
		delete(pod.Spec.Containers[containerIdx].Resources.Requests, res)
	}
	if lim != "" {
		pod.Spec.Containers[containerIdx].Resources.Limits[res] = apiResource.MustParse(lim)
	} else {
		delete(pod.Spec.Containers[containerIdx].Resources.Limits, res)
	}
}

func setContainerPercentagePolicy(configSpec *autoscaling.StartupCPUBoost, containerName string, percentage int64) {
	configSpec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
		ContainerPolicies: []autoscaling.ContainerPolicy{
			{
				ContainerName:      containerName,
				PercentageIncrease: &autoscaling.PercentageIncrease{Value: percentage},
			},
		},
	}
}

func setContainerFixedPolicy(configSpec *autoscaling.StartupCPUBoost, containerName, req, lim string) {
	configSpec.Spec.ResourcePolicy = autoscaling.ResourcePolicy{
		ContainerPolicies: []autoscaling.ContainerPolicy{
			{
				ContainerName: containerName,
				FixedResources: &autoscaling.FixedResources{
					Requests: apiResource.MustParse(req),
					Limits:   apiResource.MustParse(lim),
				},
			},
		},
	}
}
