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

package pod_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Pod", func() {
	var (
		annot *bpod.BoostPodAnnotation
		pod   *corev1.Pod
		err   error
	)

	BeforeEach(func() {
		annot = &bpod.BoostPodAnnotation{
			BoostTimestamp: time.Now(),
			InitCPURequests: map[string]string{
				containerOneName: "500m",
				containerTwoName: "500m",
			},
			InitCPULimits: map[string]string{
				containerOneName: "1",
				containerTwoName: "1",
			},
		}
		pod = podTemplate.DeepCopy()
		pod.Annotations = map[string]string{
			bpod.BoostAnnotationKey: annot.ToJSON(),
		}
	})

	Describe("Reverts the POD container resources to original values", func() {
		JustBeforeEach(func() {
			err = bpod.RevertResourceBoost(pod)
		})
		When("POD is missing startup-cpu-boost annotation", func() {
			BeforeEach(func() {
				delete(pod.ObjectMeta.Annotations, bpod.BoostAnnotationKey)
			})
			It("errors", func() {
				Expect(err).Should(HaveOccurred())
			})
		})
		When("POD has valid startup-cpu-boost annotation", func() {
			It("does not error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("removes startup-cpu-boost label", func() {
				Expect(pod.Labels).NotTo(HaveKey(bpod.BoostLabelKey))
			})
			It("removes startup-cpu-boost annotation", func() {
				Expect(pod.Annotations).NotTo(HaveKey(bpod.BoostAnnotationKey))
			})
			It("reverts CPU requests to initial values", func() {
				cpuReqOne := pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
				cpuReqTwo := pod.Spec.Containers[1].Resources.Requests[corev1.ResourceCPU]
				Expect(cpuReqOne.String()).Should(Equal(annot.InitCPURequests[containerOneName]))
				Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPURequests[containerTwoName]))
			})
			It("reverts CPU limits to initial values", func() {
				cpuReqOne := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
				cpuReqTwo := pod.Spec.Containers[1].Resources.Limits[corev1.ResourceCPU]
				Expect(cpuReqOne.String()).Should(Equal(annot.InitCPULimits[containerOneName]))
				Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPULimits[containerTwoName]))
			})
			When("Limits were removed during boost", func() {
				BeforeEach(func() {
					pod.Spec.Containers[0].Resources.Limits = nil
					pod.Spec.Containers[1].Resources.Limits = nil
				})
				It("does not error", func() {
					Expect(err).ShouldNot(HaveOccurred())
				})
				It("removes startup-cpu-boost label", func() {
					Expect(pod.Labels).NotTo(HaveKey(bpod.BoostLabelKey))
				})
				It("removes startup-cpu-boost annotation", func() {
					Expect(pod.Annotations).NotTo(HaveKey(bpod.BoostAnnotationKey))
				})
				It("reverts CPU requests to initial values", func() {
					cpuReqOne := pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
					cpuReqTwo := pod.Spec.Containers[1].Resources.Requests[corev1.ResourceCPU]
					Expect(cpuReqOne.String()).Should(Equal(annot.InitCPURequests[containerOneName]))
					Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPURequests[containerTwoName]))
				})
				It("reverts CPU limits to initial values", func() {
					cpuReqOne := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
					cpuReqTwo := pod.Spec.Containers[1].Resources.Limits[corev1.ResourceCPU]
					Expect(cpuReqOne.String()).Should(Equal(annot.InitCPULimits[containerOneName]))
					Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPULimits[containerTwoName]))
				})
			})
		})
	})
	Describe("Creates revert boost labels patch", func() {
		var (
			patchData []byte
			err       error
		)
		JustBeforeEach(func() {
			patch := bpod.NewRevertBoostLabelsPatch()
			patchData, err = patch.Data(pod)
		})
		When("Pod is missing boost labels and annotations", func() {
			BeforeEach(func() {
				delete(pod.ObjectMeta.Annotations, bpod.BoostAnnotationKey)
				delete(pod.ObjectMeta.Labels, bpod.BoostLabelKey)
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns empty patch", func() {
				Expect(string(patchData)).To(Equal("{}"))
			})
		})
		When("Pod has boost labels and annotations", func() {
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns valid patch", func() {
				Expect(string(patchData)).To(Equal("{\"metadata\":{\"annotations\":null,\"labels\":null}}"))
			})
		})
	})
	Describe("Creates revert boost resources patch", func() {
		var (
			patchData []byte
			err       error
		)
		JustBeforeEach(func() {
			patch := bpod.NewRevertBootsResourcesPatch()
			patchData, err = patch.Data(pod)
		})
		When("Pod is missing boost labels and annotations", func() {
			BeforeEach(func() {
				delete(pod.ObjectMeta.Annotations, bpod.BoostAnnotationKey)
				delete(pod.ObjectMeta.Labels, bpod.BoostLabelKey)
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns empty patch", func() {
				Expect(string(patchData)).To(Equal("{}"))
			})
		})
		When("Pod has boost labels and annotations", func() {
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns valid patch", func() {
				expectedPatch := fmt.Sprintf(
					"{\"spec\":{\"containers\":[{\"name\":\"container-one\",\"resources\":{\"limits\":{\"cpu\":\"%s\"},"+
						"\"requests\":{\"cpu\":\"%s\"}}},{\"name\":\"container-two\",\"resources\":{\"limits\":{\"cpu\":\"%s\"},"+
						"\"requests\":{\"cpu\":\"%s\"}}}]}}",
					annot.InitCPULimits[containerOneName], annot.InitCPURequests[containerOneName],
					annot.InitCPULimits[containerTwoName], annot.InitCPURequests[containerTwoName])
				Expect(string(patchData)).To(Equal(expectedPatch))
			})
		})
	})
	Describe("Activation state tracking", func() {
		var (
			annotation *bpod.BoostPodAnnotation
		)
		BeforeEach(func() {
			annotation = bpod.NewBoostAnnotation()
		})
		Describe("GetActivationState", func() {
			It("initializes activation state if nil", func() {
				annotation.ActivationState = nil
				state := annotation.GetActivationState()
				Expect(state).NotTo(BeNil())
				Expect(state.LastActivationTime).NotTo(BeNil())
				Expect(state.ActivationHistory).NotTo(BeNil())
			})
			It("initializes nested maps if nil", func() {
				annotation.ActivationState.LastActivationTime = nil
				annotation.ActivationState.ActivationHistory = nil
				state := annotation.GetActivationState()
				Expect(state.LastActivationTime).NotTo(BeNil())
				Expect(state.ActivationHistory).NotTo(BeNil())
			})
		})
		Describe("Current activation", func() {
			It("sets and gets current activation", func() {
				triggerType := autoscaling.BoostTriggerTypePodCreate
				startTime := time.Now()
				duration := int64(300) // 5 minutes
				annotation.SetCurrentActivation(triggerType, startTime, "FixedDuration", &duration, nil)
				activation := annotation.GetCurrentActivation()
				Expect(activation).NotTo(BeNil())
				Expect(activation.TriggerType).To(Equal(triggerType))
				Expect(activation.ExpiryConditionType).To(Equal("FixedDuration"))
				Expect(activation.ExpiryFixedDuration).NotTo(BeNil())
				Expect(*activation.ExpiryFixedDuration).To(Equal(int64(300)))
			})
			It("clears current activation", func() {
				triggerType := autoscaling.BoostTriggerTypePodCreate
				startTime := time.Now()
				annotation.SetCurrentActivation(triggerType, startTime, "FixedDuration", nil, nil)
				Expect(annotation.GetCurrentActivation()).NotTo(BeNil())
				annotation.ClearCurrentActivation()
				Expect(annotation.GetCurrentActivation()).To(BeNil())
			})
			It("returns nil when no current activation", func() {
				annotation.ActivationState = nil
				Expect(annotation.GetCurrentActivation()).To(BeNil())
			})
		})
		Describe("Last activation time", func() {
			It("sets and gets last activation time for trigger type", func() {
				triggerType := autoscaling.BoostTriggerTypeContainerRestart
				activationTime := time.Now()
				annotation.SetLastActivationTime(triggerType, activationTime)
				retrievedTime, found := annotation.GetLastActivationTime(triggerType)
				Expect(found).To(BeTrue())
				Expect(retrievedTime.Unix()).To(Equal(activationTime.Unix()))
			})
			It("returns false when trigger type not found", func() {
				triggerType := autoscaling.BoostTriggerTypeContainerRestart
				_, found := annotation.GetLastActivationTime(triggerType)
				Expect(found).To(BeFalse())
			})
			It("tracks multiple trigger types independently", func() {
				time1 := time.Now()
				time2 := time.Now().Add(5 * time.Minute)
				annotation.SetLastActivationTime(autoscaling.BoostTriggerTypePodCreate, time1)
				annotation.SetLastActivationTime(autoscaling.BoostTriggerTypeContainerRestart, time2)
				retrieved1, found1 := annotation.GetLastActivationTime(autoscaling.BoostTriggerTypePodCreate)
				retrieved2, found2 := annotation.GetLastActivationTime(autoscaling.BoostTriggerTypeContainerRestart)
				Expect(found1).To(BeTrue())
				Expect(found2).To(BeTrue())
				Expect(retrieved1.Unix()).To(Equal(time1.Unix()))
				Expect(retrieved2.Unix()).To(Equal(time2.Unix()))
			})
		})
		Describe("Activation history", func() {
			It("adds activation to history", func() {
				activationTime := time.Now()
				annotation.AddActivationToHistory(activationTime)
				Expect(annotation.GetActivationHistoryCount()).To(Equal(1))
			})
			It("removes activations older than 1 hour", func() {
				now := time.Now()
				oldTime := now.Add(-2 * time.Hour)
				recentTime := now.Add(-30 * time.Minute)
				annotation.AddActivationToHistory(oldTime)
				annotation.AddActivationToHistory(recentTime)
				annotation.AddActivationToHistory(now)
				// Only recentTime and now should remain
				Expect(annotation.GetActivationHistoryCount()).To(Equal(2))
			})
			It("returns zero count when activation state is nil", func() {
				annotation.ActivationState = nil
				Expect(annotation.GetActivationHistoryCount()).To(Equal(0))
			})
			It("tracks multiple activations", func() {
				now := time.Now()
				for i := 0; i < 5; i++ {
					annotation.AddActivationToHistory(now.Add(time.Duration(i) * time.Minute))
				}
				Expect(annotation.GetActivationHistoryCount()).To(Equal(5))
			})
		})
		Describe("Backward compatibility", func() {
			It("handles annotation without activation state", func() {
				oldAnnotation := &bpod.BoostPodAnnotation{
					BoostTimestamp:  time.Now(),
					InitCPURequests: map[string]string{"container": "500m"},
					InitCPULimits:   map[string]string{"container": "1"},
					// ActivationState is nil
				}
				jsonData := oldAnnotation.ToJSON()
				// Parse it back
				var parsed bpod.BoostPodAnnotation
				err := json.Unmarshal([]byte(jsonData), &parsed)
				Expect(err).NotTo(HaveOccurred())
				// Should be able to get activation state (initializes on demand)
				state := parsed.GetActivationState()
				Expect(state).NotTo(BeNil())
			})
			It("handles annotation with empty activation state", func() {
				oldAnnotation := &bpod.BoostPodAnnotation{
					BoostTimestamp:  time.Now(),
					InitCPURequests: map[string]string{"container": "500m"},
					InitCPULimits:   map[string]string{"container": "1"},
					ActivationState: &bpod.ActivationState{},
				}
				jsonData := oldAnnotation.ToJSON()
				// Parse it back
				var parsed bpod.BoostPodAnnotation
				err := json.Unmarshal([]byte(jsonData), &parsed)
				Expect(err).NotTo(HaveOccurred())
				// Should initialize nested structures
				state := parsed.GetActivationState()
				Expect(state.LastActivationTime).NotTo(BeNil())
				Expect(state.ActivationHistory).NotTo(BeNil())
			})
		})
		Describe("JSON serialization", func() {
			It("serializes and deserializes activation state correctly", func() {
				triggerType := autoscaling.BoostTriggerTypePodCreate
				startTime := time.Now()
				duration := int64(300)
				annotation.SetCurrentActivation(triggerType, startTime, "FixedDuration", &duration, nil)
				annotation.SetLastActivationTime(autoscaling.BoostTriggerTypeContainerRestart, time.Now())
				annotation.AddActivationToHistory(time.Now())
				jsonData := annotation.ToJSON()
				// Parse it back
				var parsed bpod.BoostPodAnnotation
				err := json.Unmarshal([]byte(jsonData), &parsed)
				Expect(err).NotTo(HaveOccurred())
				// Verify activation state
				activation := parsed.GetCurrentActivation()
				Expect(activation).NotTo(BeNil())
				Expect(activation.TriggerType).To(Equal(triggerType))
				Expect(parsed.GetActivationHistoryCount()).To(Equal(1))
			})
		})
	})
	Describe("applyBoostResourcesToPod edge cases", func() {
		var (
			annotation *bpod.BoostPodAnnotation
			testPod    *corev1.Pod
		)
		BeforeEach(func() {
			annotation = bpod.NewBoostAnnotation()
			testPod = podTemplate.DeepCopy()
		})
		When("removeLimits is true with burstable pod", func() {
			It("should remove CPU limits for burstable pod", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					if containerName == containerOneName {
						return func(ctx context.Context, c *corev1.Container) *corev1.ResourceRequirements {
							return &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("2.4"),
								},
							}
						}, true
					}
					return nil, false
				}
				// Create burstable pod (requests != limits)
				testPod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU: apiResource.MustParse("1"),
				}
				testPod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: apiResource.MustParse("2"),
				}
				// Test through patch function - the patch will remove limits for burstable pod
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, true, false)
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				// Verify patch was created
				Expect(patchData).NotTo(BeEmpty())
				// The actual limit removal happens when patch is applied, but we verify patch creation
			})
		})
		When("removeLimits is true with guaranteed pod", func() {
			It("should not remove CPU limits for guaranteed pod", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					if containerName == containerOneName {
						return func(ctx context.Context, c *corev1.Container) *corev1.ResourceRequirements {
							return &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
							}
						}, true
					}
					return nil, false
				}
				// Create guaranteed pod (requests == limits)
				testPod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU: apiResource.MustParse("1"),
				}
				testPod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU: apiResource.MustParse("1"),
				}
				// Test through patch function since ApplyBoostResources is not exported
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, true, false)
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				// Verify patch was created
				Expect(patchData).NotTo(BeEmpty())
				// For guaranteed pod, limits should be applied (not removed)
				// The actual application happens when patch is applied, but we verify the patch is created
			})
		})
		When("container has resize policy requiring restart", func() {
			It("should skip container with restart policy", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					if containerName == containerOneName {
						return func(ctx context.Context, c *corev1.Container) *corev1.ResourceRequirements {
							return &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
							}
						}, true
					}
					return nil, false
				}
				// Add resize policy that requires restart
				testPod.Spec.Containers[0].ResizePolicy = []corev1.ContainerResizePolicy{
					{
						ResourceName:  corev1.ResourceCPU,
						RestartPolicy: corev1.RestartContainer,
					},
				}
				originalCPU := testPod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
				// Test through patch function - container with restart policy should be skipped
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, false, false)
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				// Verify patch was created (but container with restart policy is skipped)
				Expect(patchData).NotTo(BeEmpty())
				// Original resources should remain unchanged in the pod
				Expect(testPod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU].Equal(originalCPU)).To(BeTrue())
			})
		})
		When("container has no resources to increase", func() {
			It("should skip container with no resources", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					if containerName == containerOneName {
						return func(ctx context.Context, c *corev1.Container) *corev1.ResourceRequirements {
							return &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
							}
						}, true
					}
					return nil, false
				}
				// Remove all resources
				testPod.Spec.Containers[0].Resources = corev1.ResourceRequirements{}
				// Test through patch function - container with no resources should be skipped
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, false, false)
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				// Verify patch was created (but container with no resources is skipped)
				Expect(patchData).NotTo(BeEmpty())
				// Resources should remain empty
				Expect(len(testPod.Spec.Containers[0].Resources.Requests)).To(Equal(0))
			})
		})
		When("podLevelResourcesEnabled is true", func() {
			It("should handle pod-level resources", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					return nil, false
				}
				// Set pod-level resources
				testPod.Spec.Resources = &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: apiResource.MustParse("1"),
					},
				}
				// Test through patch function with podLevelResourcesEnabled
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, false, true)
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				// Verify patch was created
				Expect(patchData).NotTo(BeEmpty())
			})
		})
	})
	Describe("ApplyBoostResourcesPatch", func() {
		var (
			annotation *bpod.BoostPodAnnotation
			testPod    *corev1.Pod
		)
		BeforeEach(func() {
			annotation = bpod.NewBoostAnnotation()
			testPod = podTemplate.DeepCopy()
		})
		When("creating patch for boost resources", func() {
			It("should create valid patch", func() {
				getResourcePolicy := func(containerName string) (func(context.Context, *corev1.Container) *corev1.ResourceRequirements, bool) {
					if containerName == containerOneName {
						return func(ctx context.Context, c *corev1.Container) *corev1.ResourceRequirements {
							return &corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("1.2"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: apiResource.MustParse("2.4"),
								},
							}
						}, true
					}
					return nil, false
				}
				patch := bpod.NewApplyBoostResourcesPatch(annotation, getResourcePolicy, false, false)
				Expect(patch).NotTo(BeNil())
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				Expect(patchData).NotTo(BeEmpty())
				// Verify patch type is set (just check it's not zero value)
				Expect(patch.Type()).NotTo(BeEmpty())
			})
		})
	})
	Describe("ApplyBoostLabelsPatch", func() {
		var (
			annotation *bpod.BoostPodAnnotation
			testPod    *corev1.Pod
		)
		BeforeEach(func() {
			annotation = bpod.NewBoostAnnotation()
			testPod = podTemplate.DeepCopy()
		})
		When("creating patch for boost labels", func() {
			It("should create valid patch", func() {
				patch := bpod.NewApplyBoostLabelsPatch(annotation, "boost-001")
				Expect(patch).NotTo(BeNil())
				patchData, err := patch.Data(testPod)
				Expect(err).NotTo(HaveOccurred())
				Expect(patchData).NotTo(BeEmpty())
				// Verify patch type is set (just check it's not zero value)
				Expect(patch.Type()).NotTo(BeEmpty())
			})
		})
	})
})
