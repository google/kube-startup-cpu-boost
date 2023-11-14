package policy_test

import (
	bpolicy "github.com/google/kube-startup-cpu-boost/internal/boost/policy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("PodConditionPolicy", func() {
	var policy bpolicy.DurationPolicy
	var condition corev1.PodConditionType
	var status corev1.ConditionStatus

	BeforeEach(func() {
		condition = corev1.PodReady
		status = corev1.ConditionTrue
		policy = bpolicy.NewPodConditionPolicy(condition, status)
	})

	Describe("Validates POD", func() {
		Context("when the POD has no condition with matching type", func() {
			BeforeEach(func() {
				pod.Status.Conditions = make([]corev1.PodCondition, 0)
			})
			It("returns policy is valid", func() {
				Expect(policy.Valid(pod)).To(BeTrue())
			})
		})
		Context("when the POD has condition with matching type", func() {
			BeforeEach(func() {
				pod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   condition,
						Status: corev1.ConditionUnknown,
					},
				}
			})
			When("condition status matches status in a policy", func() {
				It("returns policy is invalid", func() {
					pod.Status.Conditions[0].Status = status
					Expect(policy.Valid(pod)).To(BeFalse())
				})
			})
			When("condition status does not match status in a policy", func() {
				It("returns policy is valid", func() {
					Expect(policy.Valid(pod)).To(BeTrue())
				})
			})
		})
	})
})
