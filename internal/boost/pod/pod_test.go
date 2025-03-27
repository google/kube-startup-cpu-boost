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
	"fmt"
	"time"

	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
})
