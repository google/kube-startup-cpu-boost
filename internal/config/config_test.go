// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config_test

import (
	"github.com/google/kube-startup-cpu-boost/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var cfg config.Config
	BeforeEach(func() {
		cfg = config.Config{}
	})
	Describe("Loads defaults", func() {
		JustBeforeEach(func() {
			cfg.LoadDefaults()
		})
		It("has valid default namespace", func() {
			Expect(cfg.Namespace).To(Equal(config.PodNamespaceDefault))
		})
		It("has valid default manager check time interval", func() {
			Expect(cfg.MgrCheckIntervalSec).To(Equal(config.MgrCheckIntervalSecDefault))
		})
		It("has valid leader election", func() {
			Expect(cfg.LeaderElection).To(Equal(config.LeaderElectionDefault))
		})
		It("has valid metrics probe bind address", func() {
			Expect(cfg.MetricsProbeBindAddr).To(Equal(config.MetricsProbeBindAddrDefault))
		})
		It("has valid health probe bind address", func() {
			Expect(cfg.MetricsProbeBindAddr).To(Equal(config.MetricsProbeBindAddrDefault))
		})
		It("has valid secure metrics", func() {
			Expect(cfg.SecureMetrics).To(Equal(config.SecureMetricsDefault))
		})
		It("has valid ZAP log level", func() {
			Expect(cfg.ZapLogLevel).To(Equal(config.ZapLogLevelDefault))
		})
		It("has valid ZAP development ", func() {
			Expect(cfg.ZapDevelopment).To(Equal(config.ZapDevelopmentDefault))
		})
		It("has valid HTTP2 ", func() {
			Expect(cfg.HTTP2).To(Equal(config.HTTP2Default))
		})
	})
})
