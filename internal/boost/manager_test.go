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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Manager", func() {
	var manager cpuboost.Manager
	BeforeEach(func() {
		metrics.ClearSystemMetrics()
	})
	Describe("Registers startup-cpu-boost", func() {
		var (
			spec                *autoscaling.StartupCPUBoost
			boost               cpuboost.StartupCPUBoost
			useLegacyRevertMode bool
			err                 error
		)
		BeforeEach(func() {
			spec = specTemplate.DeepCopy()
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec, useLegacyRevertMode)
			Expect(err).ToNot(HaveOccurred())
		})
		When("startup-cpu-boost exists", func() {
			JustBeforeEach(func() {
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("startup-cpu-boost does not exist", func() {
			When("manager has no matching orphaned pod", func() {
				JustBeforeEach(func() {
					err = manager.AddRegularCPUBoost(context.TODO(), boost)
				})
				It("does not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
				It("stores the startup-cpu-boost", func() {
					stored, ok := manager.GetRegularCPUBoost(context.TODO(), spec.Name, spec.Namespace)
					Expect(ok).To(BeTrue())
					Expect(stored.Name()).To(Equal(spec.Name))
					Expect(stored.Namespace()).To(Equal(spec.Namespace))
				})
				It("updates boost configurations metric", func() {
					Expect(metrics.BoostConfigurations(spec.Namespace)).To(Equal(float64(1)))
				})
			})
			When("manager has matching orphaned pod", func() {
				var (
					pod          *corev1.Pod
					matchedBoost cpuboost.StartupCPUBoost
				)
				BeforeEach(func() {
					podNameLabel := "app.kubernetes.io/name"
					podNameLabelValue := "app-001"
					pod = podTemplate.DeepCopy()
					pod.Labels[podNameLabel] = podNameLabelValue
					spec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{},
						podNameLabel, podNameLabelValue)
				})
				JustBeforeEach(func() {
					matchedBoost, err = manager.UpsertPod(context.TODO(), pod)
					Expect(err).ToNot(HaveOccurred())
					Expect(matchedBoost).To(BeNil())
					err = manager.AddRegularCPUBoost(context.TODO(), boost)
				})
				It("does not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})
				It("stores the startup-cpu-boost", func() {
					stored, ok := manager.GetRegularCPUBoost(context.TODO(), spec.Name, spec.Namespace)
					Expect(ok).To(BeTrue())
					Expect(stored).To(Equal(boost))
				})
				It("stored boost manages orphaned pod", func() {
					managedPod, ok := boost.Pod(pod.Name)
					Expect(ok).To(BeTrue())
					Expect(managedPod).To(Equal(pod))
				})
			})
		})
	})
	Describe("De-registers startup-cpu-boost", func() {
		var (
			spec                *autoscaling.StartupCPUBoost
			boost               cpuboost.StartupCPUBoost
			useLegacyRevertMode bool
			err                 error
		)
		BeforeEach(func() {
			spec = specTemplate.DeepCopy()
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec, useLegacyRevertMode)
			Expect(err).ToNot(HaveOccurred())
		})
		When("startup-cpu-boost exists", func() {
			JustBeforeEach(func() {
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				manager.DeleteRegularCPUBoost(context.TODO(), boost.Namespace(), boost.Name())
			})
			It("removes the startup-cpu-boost", func() {
				_, ok := manager.GetRegularCPUBoost(context.TODO(), spec.Name, spec.Namespace)
				Expect(ok).To(BeFalse())
			})
			It("updates boost configurations metric", func() {
				Expect(metrics.BoostConfigurations(spec.Namespace)).To(Equal(float64(0)))
			})
		})
	})
	Describe("updates startup-cpu-boost from spec", func() {
		var (
			boost               cpuboost.StartupCPUBoost
			err                 error
			useLegacyRevertMode bool
			spec                *autoscaling.StartupCPUBoost
			updatedSpec         *autoscaling.StartupCPUBoost
		)
		BeforeEach(func() {
			spec = specTemplate.DeepCopy()
			updatedSpec = spec.DeepCopy()
			updatedSpec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
				Unit:  autoscaling.FixedDurationPolicyUnitMin,
				Value: 1000,
			}
		})
		JustBeforeEach(func() {
			boost, err = cpuboost.NewStartupCPUBoost(nil, spec, useLegacyRevertMode)
			Expect(err).ToNot(HaveOccurred())
		})
		When("startup-cpu-boost is registered", func() {
			JustBeforeEach(func() {
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				err = manager.UpdateRegularCPUBoost(context.TODO(), updatedSpec)
				Expect(err).ToNot(HaveOccurred())
			})
			It("updates the startup-cpu-boost", func() {
				boost, ok := manager.GetRegularCPUBoost(context.TODO(), updatedSpec.Name,
					updatedSpec.Namespace)
				Expect(ok).To(BeTrue())
				durationPolicies := boost.DurationPolicies()
				durationPolicy, ok := durationPolicies[duration.FixedDurationPolicyName]
				Expect(ok).To(BeTrue())
				Expect(durationPolicy).To(BeAssignableToTypeOf(&duration.FixedDurationPolicy{}))
				fixedDurationPolicy := durationPolicy.(*duration.FixedDurationPolicy)
				Expect(fixedDurationPolicy.Duration()).To(Equal(1000 * time.Minute))
			})
		})
	})
	Describe("retrieves startup-cpu-boost for a POD", func() {
		var (
			pod                 *corev1.Pod
			podNameLabel        string
			podNameLabelValue   string
			boost               cpuboost.StartupCPUBoost
			useLegacyRevertMode bool
			found               bool
		)
		BeforeEach(func() {
			podNameLabel = "app.kubernetes.io/name"
			podNameLabelValue = "app-001"
			pod = podTemplate.DeepCopy()
			pod.Labels[podNameLabel] = podNameLabelValue
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManager(nil)
		})
		When("matching startup-cpu-boost does not exist", func() {
			JustBeforeEach(func() {
				boost, found = manager.GetCPUBoostForPod(context.TODO(), pod)
			})
			It("returns false", func() {
				Expect(found).To(BeFalse())
			})
			It("return nil", func() {
				Expect(boost).To(BeNil())
			})
		})
		When("matching startup-cpu-boost exists", func() {
			var (
				spec *autoscaling.StartupCPUBoost
				err  error
			)
			BeforeEach(func() {
				spec = specTemplate.DeepCopy()
				spec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{}, podNameLabel, podNameLabelValue)
			})
			JustBeforeEach(func() {
				boost, err = cpuboost.NewStartupCPUBoost(nil, spec, useLegacyRevertMode)
				Expect(err).NotTo(HaveOccurred())
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).NotTo(HaveOccurred())
				boost, found = manager.GetCPUBoostForPod(context.TODO(), pod)
			})
			It("returns true", func() {
				Expect(found).To(BeTrue())
			})
			It("returns valid boost", func() {
				Expect(boost).NotTo(BeNil())
				Expect(boost.Name()).To(Equal(spec.Name))
				Expect(boost.Namespace()).To(Equal(spec.Namespace))
			})
		})
	})
	Describe("handles pod upsert", func() {
		var (
			podNameLabel      string
			podNameLabelValue string
			pod               *corev1.Pod
			matchedBoost      cpuboost.StartupCPUBoost
			err               error
		)
		BeforeEach(func() {
			podNameLabel = "app.kubernetes.io/name"
			podNameLabelValue = "app-001"
			pod = podTemplate.DeepCopy()
			pod.Labels[podNameLabel] = podNameLabelValue
		})
		JustBeforeEach(func() {
			matchedBoost, err = manager.UpsertPod(context.TODO(), pod)
		})
		When("there is a matching boost", func() {
			var (
				boost cpuboost.StartupCPUBoost
			)
			BeforeEach(func() {
				boostSpec := specTemplate.DeepCopy()
				boostSpec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{},
					podNameLabel, podNameLabelValue)
				boost, err = cpuboost.NewStartupCPUBoost(nil, boostSpec, false)
				Expect(err).ToNot(HaveOccurred())
				manager = cpuboost.NewManager(nil)
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
			})
			It("doesn't error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("return's valid matched boost", func() {
				Expect(matchedBoost).To(Equal(boost))
			})
		})
		When("there is no matching boost", func() {
			BeforeEach(func() {
				manager = cpuboost.NewManager(nil)
			})
			It("doesn't error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("doesn't return matched boost", func() {
				Expect(matchedBoost).To(BeNil())
			})
		})
	})
	Describe("handles pod delete", func() {
		var (
			podNameLabel      string
			podNameLabelValue string
			pod               *corev1.Pod
			matchedBoost      cpuboost.StartupCPUBoost
			err               error
		)
		BeforeEach(func() {
			podNameLabel = "app.kubernetes.io/name"
			podNameLabelValue = "app-001"
			pod = podTemplate.DeepCopy()
			pod.Labels[podNameLabel] = podNameLabelValue
		})
		JustBeforeEach(func() {
			matchedBoost, err = manager.DeletePod(context.TODO(), pod)
		})
		When("there is a matching boost", func() {
			var (
				boost cpuboost.StartupCPUBoost
			)
			BeforeEach(func() {
				boostSpec := specTemplate.DeepCopy()
				boostSpec.Selector = *metav1.AddLabelToSelector(&metav1.LabelSelector{},
					podNameLabel, podNameLabelValue)
				boost, err = cpuboost.NewStartupCPUBoost(nil, boostSpec, false)
				Expect(err).ToNot(HaveOccurred())
				manager = cpuboost.NewManager(nil)
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ToNot(HaveOccurred())
				err = boost.UpsertPod(context.TODO(), pod)
				Expect(err).ToNot(HaveOccurred())
			})
			It("doesn't error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("return's valid matched boost", func() {
				Expect(matchedBoost).To(Equal(boost))
			})
			It("matched boost doesn't manage deleted pod", func() {
				managedPod, ok := boost.Pod(pod.Name)
				Expect(managedPod).To(BeNil())
				Expect(ok).To(BeFalse())
			})
		})
		When("there is no matching boost", func() {
			BeforeEach(func() {
				manager = cpuboost.NewManager(nil)
			})
			It("doesn't error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("doesn't return matched boost", func() {
				Expect(matchedBoost).To(BeNil())
			})
		})
	})
	Describe("runs on a time tick", func() {
		var (
			mockCtrl   *gomock.Controller
			mockTicker *mock.MockTimeTicker
			ctx        context.Context
			cancel     context.CancelFunc
			err        error
			done       chan int
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockTicker = mock.NewMockTimeTicker(mockCtrl)
			ctx, cancel = context.WithCancel(context.TODO())
			done = make(chan int)
		})
		JustBeforeEach(func() {
			manager = cpuboost.NewManagerWithTicker(nil, mockTicker)
			go func() {
				defer GinkgoRecover()
				err = manager.Start(ctx)
				done <- 1
			}()
		})
		When("There are no startup-cpu-boosts with fixed duration policy", func() {
			var c chan time.Time
			BeforeEach(func() {
				c = make(chan time.Time, 1)
				mockTicker.EXPECT().Tick().MinTimes(1).Return(c)
				mockTicker.EXPECT().Stop().Return()
			})
			JustBeforeEach(func() {
				c <- time.Now()
				time.Sleep(500 * time.Millisecond)
				cancel()
				<-done
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("There are startup-cpu-boosts with fixed duration policy", func() {
			var (
				spec                *autoscaling.StartupCPUBoost
				boost               cpuboost.StartupCPUBoost
				useLegacyRevertMode bool
				pod                 *corev1.Pod
				mockClient          *mock.MockClient
				mockReconciler      *mock.MockReconciler
				c                   chan time.Time
				durationSeconds     int64
			)
			var itShouldRevertBoost = func() {
				When("legacy revert mode is not used", func() {
					var (
						mockSubResourceClient *mock.MockSubResourceClient
					)
					BeforeEach(func() {
						useLegacyRevertMode = false
						mockSubResourceClient = mock.NewMockSubResourceClient(mockCtrl)
						mockSubResourceClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBootsResourcesPatch())).Return(nil).Times(1)
						mockClient.EXPECT().SubResource("resize").
							Return(mockSubResourceClient).Times(1)
						mockClient.EXPECT().Patch(gomock.Any(), gomock.Eq(pod),
							gomock.Eq(bpod.NewRevertBoostLabelsPatch())).Return(nil).Times(1)
					})
					It("doesn't error", func() {
						Expect(err).NotTo(HaveOccurred())
					})
				})
				When("legacy revert mode is used", func() {
					BeforeEach(func() {
						useLegacyRevertMode = true
						mockClient.EXPECT().Update(gomock.Any(), gomock.Eq(pod)).
							MinTimes(1).Return(nil)
					})
					It("doesn't error", func() {
						Expect(err).NotTo(HaveOccurred())
					})
				})
			}
			BeforeEach(func() {
				spec = specTemplate.DeepCopy()
				durationSeconds = 60

				pod = podTemplate.DeepCopy()
				creationTimestamp := time.Now().
					Add(-1 * time.Duration(durationSeconds) * time.Second).
					Add(-1 * time.Minute)
				pod.CreationTimestamp = metav1.NewTime(creationTimestamp)
				mockClient = mock.NewMockClient(mockCtrl)
				mockReconciler = mock.NewMockReconciler(mockCtrl)

				c = make(chan time.Time, 1)
				mockTicker.EXPECT().Tick().MinTimes(1).Return(c)
				mockTicker.EXPECT().Stop().Return()
				reconcileReq := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: spec.Name, Namespace: spec.Namespace,
					}}
				mockReconciler.EXPECT().Reconcile(gomock.Any(), gomock.Eq(reconcileReq)).Times(1)
			})
			JustBeforeEach(func() {
				manager.SetStartupCPUBoostReconciler(mockReconciler)
				boost, err = cpuboost.NewStartupCPUBoost(mockClient, spec, useLegacyRevertMode)
				Expect(err).ShouldNot(HaveOccurred())
				err = boost.UpsertPod(ctx, pod)
				Expect(err).ShouldNot(HaveOccurred())
				err = manager.AddRegularCPUBoost(context.TODO(), boost)
				Expect(err).ShouldNot(HaveOccurred())
			})
			When("The startup-cpu-boost was created with fixed duration policy", func() {
				BeforeEach(func() {
					spec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}
				})
				JustBeforeEach(func() {
					c <- time.Now()
					time.Sleep(500 * time.Millisecond)
					cancel()
					<-done
				})
				itShouldRevertBoost()
			})
			When("The startup-cpu-boost was updated with fixed duration policy", func() {
				var updatedSpec *autoscaling.StartupCPUBoost
				BeforeEach(func() {
					spec.Spec.DurationPolicy.PodCondition = &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					}
					updatedSpec = specTemplate.DeepCopy()
					updatedSpec.Spec.DurationPolicy.Fixed = &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: durationSeconds,
					}
				})
				JustBeforeEach(func() {
					err = manager.UpdateRegularCPUBoost(ctx, updatedSpec)
					Expect(err).ShouldNot(HaveOccurred())

					c <- time.Now()
					time.Sleep(500 * time.Millisecond)
					cancel()
					<-done
				})
				itShouldRevertBoost()
			})
		})
	})
})
