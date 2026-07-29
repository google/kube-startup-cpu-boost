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
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Manager", func() {
	var (
		mockCtrl       *gomock.Controller
		mockClient     *mock.MockClient
		mockReconciler *mock.MockReconciler
		spec           *autoscaling.StartupCPUBoost
		config         *cpuboost.StartupCPUBoostConfig
	)

	BeforeEach(func() {
		metrics.ClearSystemMetrics()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		mockReconciler = mock.NewMockReconciler(mockCtrl)
		spec = specTemplate.DeepCopy()
		config = &cpuboost.StartupCPUBoostConfig{
			Client: mockClient,
		}
	})

	Describe("AddRegularCPUBoost", func() {
		var boost cpuboost.StartupCPUBoost

		BeforeEach(func() {
			var err error
			boost, err = cpuboost.NewStartupCPUBoost(spec, config)
			Expect(err).To(Succeed())
		})

		When("startup-cpu-boost exists", func() {
			It("errors when added again", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())
				Expect(manager.AddRegularCPUBoost(ctx, boost)).ToNot(Succeed())
			})
		})

		When("startup-cpu-boost does not exist", func() {
			When("manager has no matching orphaned pod", func() {
				It("stores the startup-cpu-boost and updates metrics", func(ctx context.Context) {
					manager := cpuboost.NewManager(nil)
					Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

					stored, ok := manager.GetRegularCPUBoost(ctx, spec.Name, spec.Namespace)
					Expect(ok).To(BeTrue())
					Expect(stored.Name()).To(Equal(spec.Name))
					Expect(stored.Namespace()).To(Equal(spec.Namespace))
					Expect(metrics.BoostConfigurations(spec.Namespace)).To(Equal(float64(1)))
				})
			})

			When("manager has matching orphaned pod", func() {
				var pod *corev1.Pod
				BeforeEach(func() {
					pod = podTemplate.DeepCopy()
					pod.Labels["app.kubernetes.io/name"] = "app-001"
					spec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, "app.kubernetes.io/name", "app-001")

					var err error
					boost, err = cpuboost.NewStartupCPUBoost(spec, config)
					Expect(err).To(Succeed())
				})

				It("stores the boost and manages the orphaned pod", func(ctx context.Context) {
					manager := cpuboost.NewManager(nil)
					matchedBoost, err := manager.UpsertPod(ctx, pod)
					Expect(err).To(Succeed())
					Expect(matchedBoost).To(BeNil())

					Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

					stored, ok := manager.GetRegularCPUBoost(ctx, spec.Name, spec.Namespace)
					Expect(ok).To(BeTrue())
					Expect(stored).To(Equal(boost))

					managedPod, ok := boost.Pod(pod.Name)
					Expect(ok).To(BeTrue())
					Expect(managedPod).To(Equal(pod))
				})
			})
		})
	})

	Describe("DeleteRegularCPUBoost", func() {
		When("startup-cpu-boost exists", func() {
			It("removes the startup-cpu-boost and updates metrics", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).To(Succeed())
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

				manager.DeleteRegularCPUBoost(ctx, boost.Namespace(), boost.Name())

				_, ok := manager.GetRegularCPUBoost(ctx, spec.Name, spec.Namespace)
				Expect(ok).To(BeFalse())
				Expect(metrics.BoostConfigurations(spec.Namespace)).To(Equal(float64(0)))
			})
		})
	})

	Describe("UpdateRegularCPUBoost", func() {
		var updatedSpec *autoscaling.StartupCPUBoost

		BeforeEach(func() {
			updatedSpec = spec.DeepCopy()
			updatedSpec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
				Unit:  autoscaling.FixedDurationPolicyUnitMin,
				Value: 1000,
			}
		})

		When("startup-cpu-boost is registered", func() {
			It("updates the startup-cpu-boost", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).To(Succeed())
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

				Expect(manager.UpdateRegularCPUBoost(ctx, updatedSpec)).To(Succeed())

				updatedBoost, ok := manager.GetRegularCPUBoost(ctx, updatedSpec.Name, updatedSpec.Namespace)
				Expect(ok).To(BeTrue())

				durationPolicies := updatedBoost.DurationPolicies()
				durationPolicy, ok := durationPolicies[duration.FixedDurationPolicyName]
				Expect(ok).To(BeTrue())
				fixedDurationPolicy, ok := durationPolicy.(*duration.FixedDurationPolicy)
				Expect(ok).To(BeTrue(), "expected durationPolicy to be *duration.FixedDurationPolicy")
				Expect(fixedDurationPolicy.Duration()).To(Equal(1000 * time.Minute))
			})
		})
	})

	Describe("GetCPUBoostForPod", func() {
		var pod *corev1.Pod

		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			pod.Labels["app.kubernetes.io/name"] = "app-001"
		})

		When("matching startup-cpu-boost does not exist", func() {
			It("returns false and nil boost", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				boost, found := manager.GetCPUBoostForPod(ctx, pod)
				Expect(found).To(BeFalse())
				Expect(boost).To(BeNil())
			})
		})

		When("matching startup-cpu-boost exists", func() {
			It("returns true and valid boost", func(ctx context.Context) {
				spec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, "app.kubernetes.io/name", "app-001")
				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).To(Succeed())

				manager := cpuboost.NewManager(nil)
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

				foundBoost, found := manager.GetCPUBoostForPod(ctx, pod)
				Expect(found).To(BeTrue())
				Expect(foundBoost).NotTo(BeNil())
				Expect(foundBoost.Name()).To(Equal(spec.Name))
				Expect(foundBoost.Namespace()).To(Equal(spec.Namespace))
			})
		})
	})

	Describe("UpsertPod", func() {
		var pod *corev1.Pod

		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			pod.Labels["app.kubernetes.io/name"] = "app-001"
		})

		When("there is a matching boost", func() {
			It("returns valid matched boost without error", func(ctx context.Context) {
				boostSpec := specTemplate.DeepCopy()
				boostSpec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, "app.kubernetes.io/name", "app-001")
				boost, err := cpuboost.NewStartupCPUBoost(boostSpec, config)
				Expect(err).To(Succeed())

				manager := cpuboost.NewManager(nil)
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

				matchedBoost, err := manager.UpsertPod(ctx, pod)
				Expect(err).To(Succeed())
				Expect(matchedBoost).To(Equal(boost))
			})
		})

		When("there is no matching boost", func() {
			It("returns nil matched boost without error", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				matchedBoost, err := manager.UpsertPod(ctx, pod)
				Expect(err).To(Succeed())
				Expect(matchedBoost).To(BeNil())
			})
		})
	})

	Describe("DeletePod", func() {
		var pod *corev1.Pod

		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			pod.Labels["app.kubernetes.io/name"] = "app-001"
		})

		When("there is a matching boost", func() {
			It("removes the pod from the matched boost", func(ctx context.Context) {
				boostSpec := specTemplate.DeepCopy()
				boostSpec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, "app.kubernetes.io/name", "app-001")
				boost, err := cpuboost.NewStartupCPUBoost(boostSpec, config)
				Expect(err).To(Succeed())

				manager := cpuboost.NewManager(nil)
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())
				Expect(boost.UpsertPod(ctx, pod)).To(Succeed())

				matchedBoost, err := manager.DeletePod(ctx, pod)
				Expect(err).To(Succeed())
				Expect(matchedBoost).To(Equal(boost))

				managedPod, ok := boost.Pod(pod.Name)
				Expect(managedPod).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})

		When("there is no matching boost", func() {
			It("returns nil matched boost", func(ctx context.Context) {
				manager := cpuboost.NewManager(nil)
				matchedBoost, err := manager.DeletePod(ctx, pod)
				Expect(err).To(Succeed())
				Expect(matchedBoost).To(BeNil())
			})
		})
	})

	Describe("Start (runs on a time tick)", func() {
		var (
			mockTicker *mock.MockTimeTicker
			c          chan time.Time
			manager    cpuboost.Manager
		)

		BeforeEach(func() {
			mockTicker = mock.NewMockTimeTicker(mockCtrl)
			c = make(chan time.Time, 1)
			mockTicker.EXPECT().Tick().MinTimes(1).Return(c)
			mockTicker.EXPECT().Stop().Return()
		})

		When("There are no startup-cpu-boosts with fixed duration policy", func() {
			It("doesn't error", func(ctx context.Context) {
				manager = cpuboost.NewManagerWithTicker(nil, mockTicker)

				startCtx, cancel := context.WithCancel(ctx)
				done := make(chan struct{})
				var startErr error

				go func() {
					defer GinkgoRecover()
					startErr = manager.Start(startCtx)
					close(done)
				}()

				c <- time.Now()
				Consistently(func() error { return startErr }, "100ms").Should(Succeed())

				cancel()
				<-done
				Expect(startErr).To(Succeed())
			})
		})

		When("There are startup-cpu-boosts with fixed duration policy", func() {
			var (
				pod             *corev1.Pod
				durationSeconds int64
			)

			var runWithTickAndRevert = func(ctx context.Context, setupExpectations func(patchCalled chan struct{})) {
				patchCalled := make(chan struct{}, 1)
				setupExpectations(patchCalled)

				manager = cpuboost.NewManagerWithTicker(nil, mockTicker)
				manager.SetStartupCPUBoostReconciler(mockReconciler)

				boost, err := cpuboost.NewStartupCPUBoost(spec, config)
				Expect(err).To(Succeed())
				Expect(boost.UpsertPod(ctx, pod)).To(Succeed())
				Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

				startCtx, cancel := context.WithCancel(ctx)
				done := make(chan struct{})
				var startErr error

				go func() {
					defer GinkgoRecover()
					startErr = manager.Start(startCtx)
					close(done)
				}()

				c <- time.Now()
				Eventually(patchCalled).Should(Receive())

				cancel()
				<-done
				Expect(startErr).To(Succeed())
			}

			BeforeEach(func() {
				durationSeconds = 60
				pod = podTemplate.DeepCopy()
				scheduledTimestamp := time.Now().
					Add(-1 * time.Duration(durationSeconds) * time.Second).
					Add(-1 * time.Minute)
				pod.Status.Conditions = []corev1.PodCondition{
					{
						LastTransitionTime: metav1.NewTime(scheduledTimestamp),
						Type:               corev1.PodScheduled,
						Status:             corev1.ConditionTrue,
					}}

				reconcileReq := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: spec.Name, Namespace: spec.Namespace,
					}}
				mockReconciler.EXPECT().Reconcile(gomock.Any(), gomock.Eq(reconcileReq)).Times(1)
			})

			Context("with legacy revert mode disabled", func() {
				It("reverts boost using sub-resource client", func(ctx context.Context) {
					spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}

					runWithTickAndRevert(ctx, func(patchCalled chan struct{}) {
						mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
						mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBootsResourcesPatch())).
							DoAndReturn(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
								select {
								case patchCalled <- struct{}{}:
								default:
								}
								return nil
							}).Times(1)

						mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(1)
						mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBoostLabelsPatch())).Return(nil).Times(1)
					})
				})
			})

			Context("with legacy revert mode enabled", func() {
				It("reverts boost using direct client update", func(ctx context.Context) {
					config.LegacyRevertMode = true
					spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}

					runWithTickAndRevert(ctx, func(patchCalled chan struct{}) {
						mockClient.EXPECT().Update(gomock.Any(), gomock.Eq(pod)).
							DoAndReturn(func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
								select {
								case patchCalled <- struct{}{}:
								default:
								}
								return nil
							}).MinTimes(1)
					})
				})
			})

			Context("when boost was updated with fixed duration policy", func() {
				It("reverts boost", func(ctx context.Context) {
					spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					}
					updatedSpec := specTemplate.DeepCopy()
					updatedSpec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}

					patchCalled := make(chan struct{}, 1)
					mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
					mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
						gomock.Eq(bpod.NewRevertBootsResourcesPatch())).
						DoAndReturn(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
							select {
							case patchCalled <- struct{}{}:
							default:
							}
							return nil
						}).Times(1)

					mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(1)
					mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
						gomock.Eq(bpod.NewRevertBoostLabelsPatch())).Return(nil).Times(1)

					manager = cpuboost.NewManagerWithTicker(nil, mockTicker)
					manager.SetStartupCPUBoostReconciler(mockReconciler)

					boost, err := cpuboost.NewStartupCPUBoost(spec, config)
					Expect(err).To(Succeed())
					Expect(boost.UpsertPod(ctx, pod)).To(Succeed())
					Expect(manager.AddRegularCPUBoost(ctx, boost)).To(Succeed())

					Expect(manager.UpdateRegularCPUBoost(ctx, updatedSpec)).To(Succeed())

					startCtx, cancel := context.WithCancel(ctx)
					done := make(chan struct{})
					var startErr error

					go func() {
						defer GinkgoRecover()
						startErr = manager.Start(startCtx)
						close(done)
					}()

					c <- time.Now()
					Eventually(patchCalled).Should(Receive())

					cancel()
					<-done
					Expect(startErr).To(Succeed())
				})
			})

			Context("when boost has both fixed and pod condition duration policies", func() {
				It("reverts boost based on fixed duration", func(ctx context.Context) {
					spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}
					spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					}

					runWithTickAndRevert(ctx, func(patchCalled chan struct{}) {
						mockSubResourceClient := mock.NewMockSubResourceClient(mockCtrl)
						mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBootsResourcesPatch())).
							DoAndReturn(func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
								select {
								case patchCalled <- struct{}{}:
								default:
								}
								return nil
							}).Times(1)

						mockClient.EXPECT().SubResource("resize").Return(mockSubResourceClient).Times(1)
						mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBoostLabelsPatch())).Return(nil).Times(1)
					})
				})
			})
		})
	})
})
