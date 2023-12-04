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

package duration_test

import (
	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("PodConditionPolicy", func() {
	var policy duration.Policy
	var condition corev1.PodConditionType
	var status corev1.ConditionStatus

	BeforeEach(func() {
		condition = corev1.PodReady
		status = corev1.ConditionTrue
		policy = duration.NewPodConditionPolicy(condition, status)
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
