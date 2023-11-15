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

package policy_test

import (
	"time"

	bpolicy "github.com/google/kube-startup-cpu-boost/internal/boost/policy"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("FixedDurationPolicy", func() {
	var policy bpolicy.DurationPolicy
	var now time.Time
	var duration time.Duration
	var timeFunc bpolicy.TimeFunc

	BeforeEach(func() {
		now = time.Now()
		duration = 5 * time.Second
		timeFunc = func() time.Time {
			return now
		}
		policy = bpolicy.NewFixedDurationPolicyWithTimeFunc(timeFunc, duration)
	})

	Describe("Validates POD", func() {
		When("the life time of a POD exceeds the policy duration", func() {
			It("returns policy is not valid", func() {
				creationTimesamp := now.Add(-1 * duration).Add(-1 * time.Minute)
				pod.CreationTimestamp = metav1.NewTime(creationTimesamp)
				Expect(policy.Valid(pod)).To(BeFalse())
			})
		})
		When("the life time of a POD is within policy duration", func() {
			It("returns policy is valid", func() {
				creationTimesamp := now.Add(-1 * duration).Add(1 * time.Minute)
				pod.CreationTimestamp = metav1.NewTime(creationTimesamp)
				Expect(policy.Valid(pod)).To(BeTrue())
			})
		})
	})
})
