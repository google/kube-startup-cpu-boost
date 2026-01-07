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
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Activation", func() {
	Describe("ShouldActivateForPodCreate", func() {
		When("no triggers are specified", func() {
			It("should return true for backward compatibility", func() {
				spec := autoscaling.StartupCPUBoostSpec{
					Triggers: []autoscaling.BoostTrigger{},
				}
				result := cpuboost.ShouldActivateForPodCreate(spec)
				Expect(result).To(BeTrue())
			})
		})

		When("PodCreate trigger is specified", func() {
			It("should return true", func() {
				podCreateTrigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				spec := autoscaling.StartupCPUBoostSpec{
					Triggers: []autoscaling.BoostTrigger{podCreateTrigger},
				}
				result := cpuboost.ShouldActivateForPodCreate(spec)
				Expect(result).To(BeTrue())
			})
		})

		When("only ContainerRestart trigger is specified", func() {
			It("should return false", func() {
				containerRestartTrigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypeContainerRestart,
				}
				spec := autoscaling.StartupCPUBoostSpec{
					Triggers: []autoscaling.BoostTrigger{containerRestartTrigger},
				}
				result := cpuboost.ShouldActivateForPodCreate(spec)
				Expect(result).To(BeFalse())
			})
		})

		When("multiple triggers including PodCreate", func() {
			It("should return true", func() {
				triggers := []autoscaling.BoostTrigger{
					{Type: autoscaling.BoostTriggerTypeContainerRestart},
					{Type: autoscaling.BoostTriggerTypePodCreate},
				}
				spec := autoscaling.StartupCPUBoostSpec{
					Triggers: triggers,
				}
				result := cpuboost.ShouldActivateForPodCreate(spec)
				Expect(result).To(BeTrue())
			})
		})
	})

	Describe("NewBoostActivation", func() {
		When("fixed duration policy is specified", func() {
			It("should create activation with fixed duration expiry", func() {
				trigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				durationPolicy := autoscaling.DurationPolicy{
					Fixed: &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitSec,
						Value: 60,
					},
				}
				activation := cpuboost.NewBoostActivation(trigger, durationPolicy)
				Expect(activation.TriggerType).To(Equal(autoscaling.BoostTriggerTypePodCreate))
				Expect(activation.ExpiryCondition.Type).To(Equal(cpuboost.ExpiryConditionTypeFixedDuration))
				Expect(activation.ExpiryCondition.FixedDuration).NotTo(BeNil())
				Expect(*activation.ExpiryCondition.FixedDuration).To(Equal(60 * time.Second))
			})
		})

		When("pod condition policy is specified", func() {
			It("should create activation with pod condition expiry", func() {
				trigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				durationPolicy := autoscaling.DurationPolicy{
					PodCondition: &autoscaling.PodConditionDurationPolicy{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}
				activation := cpuboost.NewBoostActivation(trigger, durationPolicy)
				Expect(activation.TriggerType).To(Equal(autoscaling.BoostTriggerTypePodCreate))
				Expect(activation.ExpiryCondition.Type).To(Equal(cpuboost.ExpiryConditionTypePodCondition))
				Expect(activation.ExpiryCondition.PodCondition).NotTo(BeNil())
				Expect(activation.ExpiryCondition.PodCondition.Type).To(Equal(string(corev1.PodReady)))
				Expect(activation.ExpiryCondition.PodCondition.Status).To(Equal(corev1.ConditionTrue))
			})
		})
		When("neither fixed nor pod condition policy is specified", func() {
			It("should create activation with empty expiry condition", func() {
				trigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				durationPolicy := autoscaling.DurationPolicy{
					Fixed:        nil,
					PodCondition: nil,
				}
				activation := cpuboost.NewBoostActivation(trigger, durationPolicy)
				Expect(activation.TriggerType).To(Equal(autoscaling.BoostTriggerTypePodCreate))
				Expect(activation.ExpiryCondition.Type).To(Equal(cpuboost.ExpiryConditionType("")))
				Expect(activation.ExpiryCondition.FixedDuration).To(BeNil())
				Expect(activation.ExpiryCondition.PodCondition).To(BeNil())
			})
		})
		When("fixed duration policy has minutes unit", func() {
			It("should create activation with duration in minutes", func() {
				trigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				durationPolicy := autoscaling.DurationPolicy{
					Fixed: &autoscaling.FixedDurationPolicy{
						Unit:  autoscaling.FixedDurationPolicyUnitMin,
						Value: 5,
					},
				}
				activation := cpuboost.NewBoostActivation(trigger, durationPolicy)
				Expect(activation.TriggerType).To(Equal(autoscaling.BoostTriggerTypePodCreate))
				Expect(activation.ExpiryCondition.Type).To(Equal(cpuboost.ExpiryConditionTypeFixedDuration))
				Expect(activation.ExpiryCondition.FixedDuration).NotTo(BeNil())
				Expect(*activation.ExpiryCondition.FixedDuration).To(Equal(5 * time.Minute))
			})
		})
		When("fixed duration policy has unknown unit", func() {
			It("should default to seconds", func() {
				trigger := autoscaling.BoostTrigger{
					Type: autoscaling.BoostTriggerTypePodCreate,
				}
				durationPolicy := autoscaling.DurationPolicy{
					Fixed: &autoscaling.FixedDurationPolicy{
						Unit:  "UnknownUnit", // Invalid unit
						Value: 60,
					},
				}
				activation := cpuboost.NewBoostActivation(trigger, durationPolicy)
				Expect(activation.TriggerType).To(Equal(autoscaling.BoostTriggerTypePodCreate))
				Expect(activation.ExpiryCondition.Type).To(Equal(cpuboost.ExpiryConditionTypeFixedDuration))
				Expect(activation.ExpiryCondition.FixedDuration).NotTo(BeNil())
				// Should default to seconds
				Expect(*activation.ExpiryCondition.FixedDuration).To(Equal(60 * time.Second))
			})
		})
	})

	Describe("BoostActivation.IsExpired", func() {
		When("fixed duration policy", func() {
			It("should return false if pod has no start time", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type:          cpuboost.ExpiryConditionTypeFixedDuration,
						FixedDuration: durationPtr(60 * time.Second),
					},
				}
				pod := &corev1.Pod{}
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})

			It("should return false if duration not elapsed", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type:          cpuboost.ExpiryConditionTypeFixedDuration,
						FixedDuration: durationPtr(60 * time.Second),
					},
				}
				now := time.Now()
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						StartTime: &metav1.Time{Time: now},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})

			It("should return true if duration has elapsed", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type:          cpuboost.ExpiryConditionTypeFixedDuration,
						FixedDuration: durationPtr(60 * time.Second),
					},
				}
				now := time.Now().Add(-61 * time.Second)
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						StartTime: &metav1.Time{Time: now},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeTrue())
			})
		})

		When("pod condition policy", func() {
			It("should return false if condition not met", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type: cpuboost.ExpiryConditionTypePodCondition,
						PodCondition: &cpuboost.PodConditionExpiry{
							Type:   string(corev1.PodReady),
							Status: corev1.ConditionTrue,
						},
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse,
							},
						},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})

			It("should return true if condition is met", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type: cpuboost.ExpiryConditionTypePodCondition,
						PodCondition: &cpuboost.PodConditionExpiry{
							Type:   string(corev1.PodReady),
							Status: corev1.ConditionTrue,
						},
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeTrue())
			})
		})
		When("expiry condition type is unknown", func() {
			It("should return false for unknown expiry type", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type: cpuboost.ExpiryConditionType("UnknownType"),
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						StartTime: &metav1.Time{Time: time.Now()},
					},
				}
				// Default case should return false
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})
		})
		When("fixed duration type but FixedDuration is nil", func() {
			It("should return false when FixedDuration is nil", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type:          cpuboost.ExpiryConditionTypeFixedDuration,
						FixedDuration: nil,
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						StartTime: &metav1.Time{Time: time.Now()},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})
		})
		When("pod condition type but PodCondition is nil", func() {
			It("should return false when PodCondition is nil", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type:         cpuboost.ExpiryConditionTypePodCondition,
						PodCondition: nil,
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				}
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})
		})
		When("pod condition type but condition not found in pod", func() {
			It("should return false when condition type doesn't match", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type: cpuboost.ExpiryConditionTypePodCondition,
						PodCondition: &cpuboost.PodConditionExpiry{
							Type:   string(corev1.PodScheduled),
							Status: corev1.ConditionTrue,
						},
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				}
				// Condition type doesn't match, should return false
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})
		})
		When("pod condition type but condition status doesn't match", func() {
			It("should return false when condition status doesn't match", func() {
				activation := cpuboost.BoostActivation{
					ExpiryCondition: cpuboost.ExpiryCondition{
						Type: cpuboost.ExpiryConditionTypePodCondition,
						PodCondition: &cpuboost.PodConditionExpiry{
							Type:   string(corev1.PodReady),
							Status: corev1.ConditionTrue,
						},
					},
				}
				pod := &corev1.Pod{
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse, // Different status
							},
						},
					},
				}
				// Condition status doesn't match, should return false
				Expect(activation.IsExpired(pod)).To(BeFalse())
			})
		})
	})
})

func durationPtr(d time.Duration) *time.Duration {
	return &d
}
