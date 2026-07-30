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
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				It("returns valid resource policy for container one", func() {
					p, ok := boost.ResourcePolicy(context.TODO(), &corev1.Container{Name: containerOneName})
					Expect(ok).To(BeTrue())
					Expect(p).To(BeAssignableToTypeOf(&resource.PercentageContainerPolicy{}))
					percPolicy, _ := p.(*resource.PercentageContainerPolicy)
					Expect(percPolicy.Percentage()).To(Equal(containerOnePercValue))
				})
				It("returns valid resource policy for container two", func() {
					p, ok := boost.ResourcePolicy(context.TODO(), &corev1.Container{Name: containerTwoName})
					Expect(ok).To(BeTrue())
					Expect(p).To(BeAssignableToTypeOf(&resource.FixedPolicy{}))
					fixedPolicy, _ := p.(*resource.FixedPolicy)
					Expect(fixedPolicy.Requests()).To(Equal(containerTwoFixedReq))
					Expect(fixedPolicy.Limits()).To(Equal(containerTwoFixedLim))
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
				It("returns valid resource policy for container matching exact rule", func() {
					p, ok := boost.ResourcePolicy(context.TODO(), &corev1.Container{Name: containerOneName})
					Expect(ok).To(BeTrue())
					Expect(p).To(BeAssignableToTypeOf(&resource.PercentageContainerPolicy{}))
					percPolicy, _ := p.(*resource.PercentageContainerPolicy)
					Expect(percPolicy.Percentage()).To(Equal(containerOnePercValue))
				})
				It("returns valid resource policy for container matching regex rule", func() {
					p, ok := boost.ResourcePolicy(context.TODO(), &corev1.Container{Name: containerTwoName})
					Expect(ok).To(BeTrue())
					Expect(p).To(BeAssignableToTypeOf(&resource.FixedPolicy{}))
					fixedPolicy, _ := p.(*resource.FixedPolicy)
					Expect(fixedPolicy.Requests()).To(Equal(containerTwoFixedReq))
					Expect(fixedPolicy.Limits()).To(Equal(containerTwoFixedLim))
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
							mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Any(),
								gomock.Any()).Return(nil).Times(0)
							mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(0)
							mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(),
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
							func(ctx context.Context, eventType bpod.PodEventType) {
								pod := podTemplate.DeepCopy()
								pod.Status.Conditions = []corev1.PodCondition{{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								}}
								mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
								mockClient := mock.NewMockClient(mockCtrl)
								mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
									gomock.Eq(bpod.NewRevertBootsResourcesPatch())).Return(nil).Times(1)
								mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(1)
								mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
									gomock.Eq(bpod.NewRevertBoostLabelsPatch())).Return(nil).Times(1)
								config.Client = mockClient
								boost, err = cpuboost.NewStartupCPUBoost(spec, config)
								Expect(err).NotTo(HaveOccurred())

								err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
									Type: eventType,
									Pod:  pod,
								})

								Expect(err).NotTo(HaveOccurred())
							},
							Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated),
							Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged),
						)
					})
					Context("using legacy revert mode", func() {
						DescribeTable("reverts resources with pod update",
							func(ctx context.Context, eventType bpod.PodEventType) {
								pod := podTemplate.DeepCopy()
								pod.Status.Conditions = []corev1.PodCondition{{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								}}
								mockClient := mock.NewMockClient(mockCtrl)
								mockClient.EXPECT().
									Update(gomock.Any(), gomock.Eq(pod)).
									Return(nil)
								config.Client = mockClient
								config.LegacyRevertMode = true
								boost, err = cpuboost.NewStartupCPUBoost(spec, config)
								Expect(err).NotTo(HaveOccurred())

								err = boost.HandlePodEvent(ctx, &bpod.PodEvent{
									Type: eventType,
									Pod:  pod,
								})

								Expect(err).NotTo(HaveOccurred())
							},
							Entry("via PodCreatedEvent", bpod.PodEventTypePodCreated),
							Entry("via ConditionChanged event", bpod.PodEventTypeConditionChanged),
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
			var (
				resourcePolicy      resource.ContainerPolicy
				resourcePolicyFound bool
			)
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
				resourcePolicy, resourcePolicyFound = boost.ResourcePolicy(context.TODO(), &corev1.Container{Name: "test"})
			})
			It("finds resource policy", func() {
				Expect(resourcePolicyFound).To(BeTrue())
			})
			It("has valid resource policy", func() {
				Expect(resourcePolicy).To(BeAssignableToTypeOf(&resource.PercentageContainerPolicy{}))
				percentagePolicy := resourcePolicy.(*resource.PercentageContainerPolicy)
				Expect(percentagePolicy.Percentage()).To(Equal(int64(1000)))
			})
		})
	})
})
