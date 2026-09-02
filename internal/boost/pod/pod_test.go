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
	"k8s.io/apimachinery/pkg/api/resource"
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
	Describe("Creates apply boost metadata patch", func() {
		var (
			patchData   []byte
			err         error
			originalPod *corev1.Pod
			mutatedPod  *corev1.Pod
		)
		BeforeEach(func() {
			originalPod = pod.DeepCopy()
			delete(originalPod.Annotations, bpod.BoostAnnotationKey)
			mutatedPod = originalPod.DeepCopy()
			mutatedPod.Labels = map[string]string{
				bpod.BoostLabelKey: "true",
			}
			mutatedPod.Annotations = map[string]string{
				bpod.BoostAnnotationKey: annot.ToJSON(),
			}
		})
		JustBeforeEach(func() {
			patch := bpod.NewApplyBoostMetadataPatch(originalPod, mutatedPod)
			patchData, err = patch.Data(mutatedPod)
		})
		When("Patch is generated", func() {
			It("returns valid patch with only metadata", func() {
				Expect(err).NotTo(HaveOccurred())
				expectedPatch := fmt.Sprintf(
					"{\"metadata\":{\"annotations\":{\"%s\":%q},\"labels\":{\"%s\":\"true\"}}}",
					bpod.BoostAnnotationKey, annot.ToJSON(), bpod.BoostLabelKey)
				Expect(string(patchData)).To(Equal(expectedPatch))
			})
		})
	})
	Describe("Creates apply boost resources patch", func() {
		var (
			patchData   []byte
			err         error
			originalPod *corev1.Pod
			mutatedPod  *corev1.Pod
		)
		BeforeEach(func() {
			originalPod = pod.DeepCopy()
			mutatedPod = pod.DeepCopy()
			mutatedPod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("3")
		})
		JustBeforeEach(func() {
			patch := bpod.NewApplyBoostResourcesPatch(originalPod, mutatedPod)
			patchData, err = patch.Data(mutatedPod)
		})
		When("Patch is generated", func() {
			It("returns valid patch with only resources", func() {
				Expect(err).NotTo(HaveOccurred())
				expectedPatch := `{"spec":{"containers":[{"name":"container-one","resources":{"limits":{"cpu":"3"},"requests":{"cpu":"1"}}},{"name":"container-two","resources":{"limits":{"cpu":"2"},"requests":{"cpu":"1"}}}]}}`
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
