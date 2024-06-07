// Copyright 2024 Google LLC
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

package metrics_test

import (
	"github.com/google/kube-startup-cpu-boost/internal/metrics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	Describe("registers new boost configuration", func() {
		var (
			namespace = "default"
		)
		BeforeEach(func() {
			metrics.ClearSystemMetrics()
		})
		JustBeforeEach(func() {
			metrics.NewBoostConfiguration(namespace)
			metrics.NewBoostConfiguration(namespace)
		})
		It("updates the boost configurations metric", func() {
			Expect(metrics.BoostConfigurations(namespace)).To(Equal(float64(2)))
		})
	})
	Describe("deletes boost configuration", func() {
		var (
			namespace = "default"
		)
		BeforeEach(func() {
			metrics.ClearSystemMetrics()
		})
		JustBeforeEach(func() {
			metrics.NewBoostConfiguration(namespace)
			metrics.NewBoostConfiguration(namespace)
			metrics.DeleteBoostConfiguration(namespace)
		})
		It("updates the boost configurations metric", func() {
			Expect(metrics.BoostConfigurations(namespace)).To(Equal(float64(1)))
		})
	})
	Describe("sets active container boost metric", func() {
		var (
			namespace = "default"
			boost     = "boost-01"
			value     = float64(5)
		)
		BeforeEach(func() {
			metrics.ClearBoostMetrics(namespace, boost)
		})
		JustBeforeEach(func() {
			metrics.SetBoostContainersActive(namespace, boost, value)
		})
		It("updates the active container boosts metric", func() {
			Expect(metrics.BoostContainersActive(namespace, boost)).To(Equal(value))
		})
	})
	Describe("adds total container boost metric", func() {
		var (
			namespace = "default"
			boost     = "boost-01"
			value     = float64(5)
		)
		BeforeEach(func() {
			metrics.ClearBoostMetrics(namespace, boost)
		})
		JustBeforeEach(func() {
			metrics.AddBoostContainersTotal(namespace, boost, 3)
			metrics.AddBoostContainersTotal(namespace, boost, value)
		})
		It("updates the total container boosts metric", func() {
			Expect(metrics.BoostContainersTotal(namespace, boost)).To(Equal(float64(8)))
		})
	})
})
