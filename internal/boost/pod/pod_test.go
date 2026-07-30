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
			State:          bpod.BoostStateActive,
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
	Describe("Reverts the POD container resources", func() {
		Context("boost on restart feature is disabled", func() {
			JustBeforeEach(func() {
				err = bpod.RevertResourceBoost(pod)
			})
			When("POD is missing startup-cpu-boost annotation", func() {
				BeforeEach(func() {
					delete(pod.Annotations, bpod.BoostAnnotationKey)
				})
				It("returns an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			When("POD has valid startup-cpu-boost annotation", func() {
				It("reverts pod metadata and resources", func() {
					Expect(err).NotTo(HaveOccurred())
					expectPodMetadataReverted(pod)
					expectPodResourcesReverted(pod, annot)
				})
			})
			When("limits were removed during boost", func() {
				BeforeEach(func() {
					pod.Spec.Containers[0].Resources.Limits = nil
					pod.Spec.Containers[1].Resources.Limits = nil
				})
				It("reverts pod metadata and resources", func() {
					Expect(err).NotTo(HaveOccurred())
					expectPodMetadataReverted(pod)
					expectPodResourcesReverted(pod, annot)
				})
			})
		})
		Context("boost on restart feature is enabled", func() {
			JustBeforeEach(func() {
				err = bpod.RevertResourceBoostWithBoostOnRestart(pod)
			})
			When("POD is missing startup-cpu-boost annotation", func() {
				BeforeEach(func() {
					delete(pod.Annotations, bpod.BoostAnnotationKey)
				})
				It("returns an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			When("POD has valid startup-cpu-boost annotation", func() {
				It("reverts pod metadata and resources", func() {
					Expect(err).NotTo(HaveOccurred())
					expectPodMetadataRevertedWithBoostOnRestart(pod, annot)
					expectPodResourcesReverted(pod, annot)
				})
			})
			When("limits were removed during boost", func() {
				BeforeEach(func() {
					pod.Spec.Containers[0].Resources.Limits = nil
					pod.Spec.Containers[1].Resources.Limits = nil
				})
				It("reverts pod metadata and resources", func() {
					Expect(err).NotTo(HaveOccurred())
					expectPodMetadataRevertedWithBoostOnRestart(pod, annot)
					expectPodResourcesReverted(pod, annot)
				})
			})
		})
	})
	Describe("Creates revert boost labels patch", func() {
		Context("boost on restart feature is disabled", func() {
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
					delete(pod.Annotations, bpod.BoostAnnotationKey)
					delete(pod.Labels, bpod.BoostLabelKey)
				})
				It("returns valid patch", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(string(patchData)).To(Equal("{}"))
				})
			})
			When("Pod has boost labels and annotations", func() {
				It("returns valid patch", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(string(patchData)).To(Equal("{\"metadata\":{\"annotations\":null,\"labels\":null}}"))
				})
			})
		})
		Context("boost on restart feature is enabled", func() {
			var (
				patchData []byte
				err       error
			)
			JustBeforeEach(func() {
				patch := bpod.NewRevertBoostLabelsWithBoostOnRestartPatch()
				patchData, err = patch.Data(pod)
			})
			When("Pod is missing boost labels and annotations", func() {
				BeforeEach(func() {
					delete(pod.Annotations, bpod.BoostAnnotationKey)
					delete(pod.Labels, bpod.BoostLabelKey)
				})
				It("returns an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
			When("Pod has boost labels and annotations", func() {
				It("returns valid patch", func() {
					Expect(err).NotTo(HaveOccurred())
					expectedAnnot := bpod.BoostPodAnnotation{
						State:           bpod.BoostStateReverted,
						BoostTimestamp:  annot.BoostTimestamp,
						InitCPURequests: annot.InitCPURequests,
						InitCPULimits:   annot.InitCPULimits,
					}
					expectedPatch := fmt.Sprintf(
						"{\"metadata\":{\"annotations\":{\"%s\":%q}}}",
						bpod.BoostAnnotationKey, expectedAnnot.ToJSON())
					Expect(string(patchData)).To(Equal(expectedPatch))
				})
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
				delete(pod.Annotations, bpod.BoostAnnotationKey)
				delete(pod.Labels, bpod.BoostLabelKey)
			})
			It("returns valid patch", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(string(patchData)).To(Equal("{}"))
			})
		})
		When("Pod has boost labels and annotations", func() {
			It("returns valid patch", func() {
				Expect(err).NotTo(HaveOccurred())
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

func expectPodMetadataRevertedWithBoostOnRestart(pod *corev1.Pod, annot *bpod.BoostPodAnnotation) {
	GinkgoHelper()
	Expect(pod.Labels).To(HaveKey(bpod.BoostLabelKey))
	Expect(pod.Annotations).To(HaveKey(bpod.BoostAnnotationKey))
	boostAnnot, err := bpod.BoostAnnotationFromPod(pod)
	Expect(err).NotTo(HaveOccurred())
	Expect(boostAnnot.State).To(Equal(bpod.BoostStateReverted))
	Expect(boostAnnot.InitCPURequests).To(Equal(annot.InitCPURequests))
	Expect(boostAnnot.InitCPULimits).To(Equal(annot.InitCPULimits))
}

func expectPodMetadataReverted(pod *corev1.Pod) {
	GinkgoHelper()
	Expect(pod.Labels).NotTo(HaveKey(bpod.BoostLabelKey))
	Expect(pod.Annotations).NotTo(HaveKey(bpod.BoostAnnotationKey))
}

func expectPodResourcesReverted(pod *corev1.Pod, annot *bpod.BoostPodAnnotation) {
	GinkgoHelper()
	cpuReqOne := pod.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
	cpuReqTwo := pod.Spec.Containers[1].Resources.Requests[corev1.ResourceCPU]
	Expect(cpuReqOne.String()).To(Equal(annot.InitCPURequests[containerOneName]))
	Expect(cpuReqTwo.String()).To(Equal(annot.InitCPURequests[containerTwoName]))

	cpuLimOne := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
	cpuLimTwo := pod.Spec.Containers[1].Resources.Limits[corev1.ResourceCPU]
	Expect(cpuLimOne.String()).To(Equal(annot.InitCPULimits[containerOneName]))
	Expect(cpuLimTwo.String()).To(Equal(annot.InitCPULimits[containerTwoName]))
}
