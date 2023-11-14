package boost_test

import (
	"testing"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBoost(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Boost Suite")
}

var (
	podTemplate   *corev1.Pod
	annotTemplate *bpod.BoostPodAnnotation
	specTemplate  *autoscaling.StartupCPUBoost
)

var _ = BeforeSuite(func() {
	specTemplate = &autoscaling.StartupCPUBoost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "boost-001",
			Namespace: "demo",
		},
		Spec: autoscaling.StartupCPUBoostSpec{
			BoostPercent: 55,
		},
	}
	annotTemplate = &bpod.BoostPodAnnotation{
		BoostTimestamp: time.Now(),
		InitCPURequests: map[string]string{
			"container-one": "500m",
			"continer-two":  "500m",
		},
		InitCPULimits: map[string]string{
			"container-one": "1",
			"continer-two":  "1",
		},
	}
	reqQuantity, err := apiResource.ParseQuantity("1")
	Expect(err).ShouldNot(HaveOccurred())
	limitQuantity, err := apiResource.ParseQuantity("2")
	Expect(err).ShouldNot(HaveOccurred())
	podTemplate = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: specTemplate.Namespace,
			Labels: map[string]string{
				bpod.BoostLabelKey: specTemplate.Name,
			},
			Annotations: map[string]string{
				bpod.BoostAnnotationKey: annotTemplate.ToJSON(),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container-one",
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
					Name: "container-two",
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
