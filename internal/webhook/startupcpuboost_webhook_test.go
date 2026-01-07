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

	"github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StartupCPUBoost webhook", func() {
	var w webhook.StartupCPUBoostWebhook
	BeforeEach(func() {
		w = webhook.StartupCPUBoostWebhook{}
	})

	When("Validates StartupCPUBoost", func() {
		var (
			boost v1alpha1.StartupCPUBoost
			err   error
		)
		When("Startup CPU Boost has no duration policy", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						DurationPolicy: v1alpha1.DurationPolicy{},
					},
				}
			})
			It("errors", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).To(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).To(HaveOccurred())
			})
		})
		When("Startup CPU Boost has more than one duration policy", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						DurationPolicy: v1alpha1.DurationPolicy{
							Fixed:        &v1alpha1.FixedDurationPolicy{},
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
					},
				}
			})
			It("errors", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).To(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).To(HaveOccurred())
			})
		})
		When("Startup CPU Boost has one duration policy", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
					},
				}
			})
			It("does not error", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("Startup CPU Boost has container without resource policies", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName: "container-one",
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
					},
				}
			})
			It("errors", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).To(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).To(HaveOccurred())
			})
		})
		When("Startup CPU Boost has container with two resource policies", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:      "container-one",
									FixedResources:     &v1alpha1.FixedResources{},
									PercentageIncrease: &v1alpha1.PercentageIncrease{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
					},
				}
			})
			It("errors", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).To(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).To(HaveOccurred())
			})
		})
		When("Startup CPU Boost has container with one resource policies", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:  "container-one",
									FixedResources: &v1alpha1.FixedResources{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
					},
				}
			})
			It("does not error", func() {
				By("validating create event")
				_, err = w.ValidateCreate(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())

				By("validating update event")
				_, err = w.ValidateUpdate(context.TODO(), nil, &boost)
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("Startup CPU Boost has triggers", func() {
			When("PodCreate trigger is valid", func() {
				BeforeEach(func() {
					podCreateType := v1alpha1.BoostTriggerTypePodCreate
					boost = v1alpha1.StartupCPUBoost{
						Spec: v1alpha1.StartupCPUBoostSpec{
							ResourcePolicy: v1alpha1.ResourcePolicy{
								ContainerPolicies: []v1alpha1.ContainerPolicy{
									{
										ContainerName:  "container-one",
										FixedResources: &v1alpha1.FixedResources{},
									},
								},
							},
							DurationPolicy: v1alpha1.DurationPolicy{
								PodCondition: &v1alpha1.PodConditionDurationPolicy{},
							},
							Triggers: []v1alpha1.BoostTrigger{
								{
									Type: podCreateType,
								},
							},
						},
					}
				})
				It("does not error", func() {
					_, err = w.ValidateCreate(context.TODO(), &boost)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			When("ContainerRestart trigger is valid", func() {
				BeforeEach(func() {
					containerRestartType := v1alpha1.BoostTriggerTypeContainerRestart
					containerName := "*"
					boost = v1alpha1.StartupCPUBoost{
						Spec: v1alpha1.StartupCPUBoostSpec{
							ResourcePolicy: v1alpha1.ResourcePolicy{
								ContainerPolicies: []v1alpha1.ContainerPolicy{
									{
										ContainerName:  "container-one",
										FixedResources: &v1alpha1.FixedResources{},
									},
								},
							},
							DurationPolicy: v1alpha1.DurationPolicy{
								PodCondition: &v1alpha1.PodConditionDurationPolicy{},
							},
							Triggers: []v1alpha1.BoostTrigger{
								{
									Type:          containerRestartType,
									ContainerName: &containerName,
								},
							},
						},
					}
				})
				It("does not error", func() {
					_, err = w.ValidateCreate(context.TODO(), &boost)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			When("PodConditionTransition trigger is valid", func() {
				BeforeEach(func() {
					transitionType := v1alpha1.BoostTriggerTypePodConditionTransition
					conditionType := "Ready"
					fromStatus := "False"
					toStatus := "True"
					boost = v1alpha1.StartupCPUBoost{
						Spec: v1alpha1.StartupCPUBoostSpec{
							ResourcePolicy: v1alpha1.ResourcePolicy{
								ContainerPolicies: []v1alpha1.ContainerPolicy{
									{
										ContainerName:  "container-one",
										FixedResources: &v1alpha1.FixedResources{},
									},
								},
							},
							DurationPolicy: v1alpha1.DurationPolicy{
								PodCondition: &v1alpha1.PodConditionDurationPolicy{},
							},
							Triggers: []v1alpha1.BoostTrigger{
								{
									Type:          transitionType,
									ConditionType: &conditionType,
									FromStatus:    &fromStatus,
									ToStatus:      &toStatus,
								},
							},
						},
					}
				})
				It("does not error", func() {
					_, err = w.ValidateCreate(context.TODO(), &boost)
					Expect(err).NotTo(HaveOccurred())
				})
			})
			When("PodConditionTransition trigger missing required fields", func() {
				BeforeEach(func() {
					transitionType := v1alpha1.BoostTriggerTypePodConditionTransition
					boost = v1alpha1.StartupCPUBoost{
						Spec: v1alpha1.StartupCPUBoostSpec{
							ResourcePolicy: v1alpha1.ResourcePolicy{
								ContainerPolicies: []v1alpha1.ContainerPolicy{
									{
										ContainerName:  "container-one",
										FixedResources: &v1alpha1.FixedResources{},
									},
								},
							},
							DurationPolicy: v1alpha1.DurationPolicy{
								PodCondition: &v1alpha1.PodConditionDurationPolicy{},
							},
							Triggers: []v1alpha1.BoostTrigger{
								{
									Type: transitionType,
									// Missing conditionType, fromStatus, toStatus
								},
							},
						},
					}
				})
				It("errors", func() {
					_, err = w.ValidateCreate(context.TODO(), &boost)
					Expect(err).To(HaveOccurred())
				})
			})
			When("Startup CPU Boost has no triggers (backward compatibility)", func() {
				BeforeEach(func() {
					boost = v1alpha1.StartupCPUBoost{
						Spec: v1alpha1.StartupCPUBoostSpec{
							ResourcePolicy: v1alpha1.ResourcePolicy{
								ContainerPolicies: []v1alpha1.ContainerPolicy{
									{
										ContainerName:  "container-one",
										FixedResources: &v1alpha1.FixedResources{},
									},
								},
							},
							DurationPolicy: v1alpha1.DurationPolicy{
								PodCondition: &v1alpha1.PodConditionDurationPolicy{},
							},
							// Triggers field omitted (nil/empty)
						},
					}
				})
				It("does not error (backward compatible)", func() {
					_, err = w.ValidateCreate(context.TODO(), &boost)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})
	When("Defaults StartupCPUBoost", func() {
		var boost v1alpha1.StartupCPUBoost
		When("Startup CPU Boost has no triggers", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:  "container-one",
									FixedResources: &v1alpha1.FixedResources{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
						// Triggers field omitted (nil/empty)
					},
				}
			})
			It("defaults to PodCreate trigger", func() {
				err := w.Default(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())
				Expect(boost.Spec.Triggers).NotTo(BeEmpty())
				Expect(len(boost.Spec.Triggers)).To(Equal(1))
				Expect(boost.Spec.Triggers[0].Type).To(Equal(v1alpha1.BoostTriggerTypePodCreate))
			})
		})
		When("Startup CPU Boost has empty triggers array", func() {
			BeforeEach(func() {
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:  "container-one",
									FixedResources: &v1alpha1.FixedResources{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
						Triggers: []v1alpha1.BoostTrigger{}, // Empty array
					},
				}
			})
			It("defaults to PodCreate trigger", func() {
				err := w.Default(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())
				Expect(boost.Spec.Triggers).NotTo(BeEmpty())
				Expect(len(boost.Spec.Triggers)).To(Equal(1))
				Expect(boost.Spec.Triggers[0].Type).To(Equal(v1alpha1.BoostTriggerTypePodCreate))
			})
		})
		When("Startup CPU Boost already has triggers", func() {
			BeforeEach(func() {
				containerRestartType := v1alpha1.BoostTriggerTypeContainerRestart
				containerName := "*"
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:  "container-one",
									FixedResources: &v1alpha1.FixedResources{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
						Triggers: []v1alpha1.BoostTrigger{
							{
								Type:          containerRestartType,
								ContainerName: &containerName,
							},
						},
					},
				}
			})
			It("does not modify existing triggers", func() {
				originalTriggers := boost.Spec.Triggers
				err := w.Default(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())
				Expect(boost.Spec.Triggers).To(Equal(originalTriggers))
				Expect(len(boost.Spec.Triggers)).To(Equal(1))
				Expect(boost.Spec.Triggers[0].Type).To(Equal(v1alpha1.BoostTriggerTypeContainerRestart))
			})
		})
		When("Startup CPU Boost has PodCreate trigger already", func() {
			BeforeEach(func() {
				podCreateType := v1alpha1.BoostTriggerTypePodCreate
				boost = v1alpha1.StartupCPUBoost{
					Spec: v1alpha1.StartupCPUBoostSpec{
						ResourcePolicy: v1alpha1.ResourcePolicy{
							ContainerPolicies: []v1alpha1.ContainerPolicy{
								{
									ContainerName:  "container-one",
									FixedResources: &v1alpha1.FixedResources{},
								},
							},
						},
						DurationPolicy: v1alpha1.DurationPolicy{
							PodCondition: &v1alpha1.PodConditionDurationPolicy{},
						},
						Triggers: []v1alpha1.BoostTrigger{
							{
								Type: podCreateType,
							},
						},
					},
				}
			})
			It("does not modify existing triggers", func() {
				originalTriggers := boost.Spec.Triggers
				err := w.Default(context.TODO(), &boost)
				Expect(err).NotTo(HaveOccurred())
				Expect(boost.Spec.Triggers).To(Equal(originalTriggers))
				Expect(len(boost.Spec.Triggers)).To(Equal(1))
				Expect(boost.Spec.Triggers[0].Type).To(Equal(v1alpha1.BoostTriggerTypePodCreate))
			})
		})
	})
})
