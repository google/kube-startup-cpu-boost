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
	"time"

	"github.com/google/kube-startup-cpu-boost/internal/boost/duration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("FixedDurationPolicy", func() {
	var policy duration.Policy
	var now time.Time
	var timeDuration time.Duration
	var timeFunc duration.TimeFunc

	BeforeEach(func() {
		now = time.Now()
		timeDuration = 5 * time.Second
		timeFunc = func() time.Time {
			return now
		}
		policy = duration.NewFixedDurationPolicyWithTimeFunc(timeFunc, timeDuration)
	})

	Describe("Validates POD", func() {
		When("the POD has no status conditions", func() {
			It("returns policy is valid", func() {
				pod.Status.Conditions = []v1.PodCondition{}
				Expect(policy.Valid(pod)).To(BeTrue())
			})
		})
		When("the life time of a POD exceeds the policy duration", func() {
			It("returns policy is not valid", func() {
				scheduleTime := now.Add(-1 * timeDuration).Add(-1 * time.Minute)
				pod.Status.Conditions = []v1.PodCondition{
					{
						LastTransitionTime: metav1.NewTime(scheduleTime),
						Type:               v1.PodScheduled,
						Status:             v1.ConditionTrue,
					}}
				Expect(policy.Valid(pod)).To(BeFalse())
			})
		})
		When("the life time of a POD is within policy duration", func() {
			It("returns policy is valid", func() {
				scheduleTime := now.Add(-1 * timeDuration).Add(1 * time.Minute)
				pod.Status.Conditions = []v1.PodCondition{
					{
						LastTransitionTime: metav1.NewTime(scheduleTime),
						Type:               v1.PodScheduled,
						Status:             v1.ConditionTrue,
					}}
				Expect(policy.Valid(pod)).To(BeTrue())
			})
		})
	})
})
