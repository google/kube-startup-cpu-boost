package policy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var pod *corev1.Pod

func TestPolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Policy Suite")
}

var _ = BeforeSuite(func() {
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{},
		},
	}
})
