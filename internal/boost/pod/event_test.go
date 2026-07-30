// Copyright 2026 Google LLC
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("PodEvent", func() {
	Describe("Validate", func() {
		DescribeTable("returns appropriate error for invalid events",
			func(event *pod.PodEvent, expectedErr error) {
				err := event.Validate()

				if expectedErr == nil {
					Expect(err).NotTo(HaveOccurred())
				} else {
					Expect(err).To(Equal(expectedErr))
				}
			},
			Entry("nil event", nil, pod.ErrNilPodEvent),
			Entry("empty type", &pod.PodEvent{Pod: &corev1.Pod{}}, pod.ErrEmptyType),
			Entry("nil pod", &pod.PodEvent{Type: pod.PodEventTypePodCreated}, pod.ErrNilPod),
			Entry("valid event", &pod.PodEvent{Type: pod.PodEventTypePodCreated, Pod: &corev1.Pod{}}, nil),
		)
	})
})
