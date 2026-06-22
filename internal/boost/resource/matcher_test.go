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

package resource_test

import (
	"context"

	"github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("ContainerMatcher", func() {
	var (
		matcher   resource.ContainerMatcher
		matches   bool
		container *corev1.Container
	)
	JustBeforeEach(func() {
		matches = matcher.Matches(context.TODO(), container)
	})
	Context("with fixed name", func() {
		BeforeEach(func() {
			matcher = &resource.FixedNameContainerMatcher{
				Name: "test-container",
			}
		})
		When("container name matches", func() {
			BeforeEach(func() {
				container = &corev1.Container{
					Name: "test-container",
				}
			})
			It("returns true", func() {
				Expect(matches).To(BeTrue())
			})
		})
		When("container name does not match", func() {
			BeforeEach(func() {
				container = &corev1.Container{
					Name: "different-container",
				}
			})
			It("returns false", func() {
				Expect(matches).To(BeFalse())
			})
		})
	})
	Context("with regex name matcher", func() {
		Context("regex is valid", func() {
			BeforeEach(func() {
				matcher = &resource.RegexNameContainerMatcher{
					Expr: "container-[0-9]+-.*",
				}
			})
			When("container name matches", func() {
				BeforeEach(func() {
					container = &corev1.Container{
						Name: "container-1-test",
					}
				})
				It("returns true", func() {
					Expect(matches).To(BeTrue())
				})
			})
			When("container name does not match", func() {
				BeforeEach(func() {
					container = &corev1.Container{
						Name: "container-one",
					}
				})
				It("returns false", func() {
					Expect(matches).To(BeFalse())
				})
			})
		})
		Context("regex compilation fails", func() {
			BeforeEach(func() {
				matcher = &resource.RegexNameContainerMatcher{
					Expr: "container-[0-9++.*",
				}
			})
			It("returns false", func() {
				Expect(matches).To(BeFalse())
			})
		})
	})
})
