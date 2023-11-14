package pod_test

import (
	"time"

	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Pod", func() {
	var pod *corev1.Pod
	var annot *bpod.BoostPodAnnotation
	var containerOne, containerTwo string
	var reqQuantity, limitQuantity apiResource.Quantity
	var err error

	BeforeEach(func() {
		containerOne = "container-one"
		containerTwo = "container-one"
		reqQuantity, err = apiResource.ParseQuantity("1")
		Expect(err).ShouldNot(HaveOccurred())
		limitQuantity, err = apiResource.ParseQuantity("2")
		Expect(err).ShouldNot(HaveOccurred())
		annot = &bpod.BoostPodAnnotation{
			BoostTimestamp: time.Now(),
			InitCPURequests: map[string]string{
				containerOne: "500m",
				containerTwo: "500m",
			},
			InitCPULimits: map[string]string{
				containerOne: "1",
				containerTwo: "1",
			},
		}
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
				Labels: map[string]string{
					bpod.BoostLabelKey: "boost-001",
				},
				Annotations: map[string]string{
					bpod.BoostAnnotationKey: annot.ToJSON(),
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: containerOne,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: reqQuantity,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: limitQuantity,
							},
						},
					},
					{
						Name: containerTwo,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: reqQuantity,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: limitQuantity,
							},
						},
					},
				},
			},
		}
	})

	Describe("Reverts the POD container resources to original values", func() {
		When("POD is missing startup-cpu-boost annotation", func() {
			BeforeEach(func() {
				delete(pod.ObjectMeta.Annotations, bpod.BoostAnnotationKey)
				err = bpod.RevertResourceBoost(pod)
			})
			It("errors", func() {
				Expect(err).Should(HaveOccurred())
			})
		})
		When("POD has valid startup-cpu-boost annotation", func() {
			BeforeEach(func() {
				err = bpod.RevertResourceBoost(pod)
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
				Expect(cpuReqOne.String()).Should(Equal(annot.InitCPURequests[containerOne]))
				Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPURequests[containerTwo]))
			})
			It("reverts CPU limits to initial values", func() {
				cpuReqOne := pod.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
				cpuReqTwo := pod.Spec.Containers[1].Resources.Limits[corev1.ResourceCPU]
				Expect(cpuReqOne.String()).Should(Equal(annot.InitCPULimits[containerOne]))
				Expect(cpuReqTwo.String()).Should(Equal(annot.InitCPULimits[containerTwo]))
			})
		})
	})
})
