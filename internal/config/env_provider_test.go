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
	"fmt"

	"github.com/google/kube-startup-cpu-boost/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnvProvider", func() {
	var (
		provider      config.EnvConfigProvider
		lookupFuncMap map[string]string
	)
	BeforeEach(func() {
		lookupFuncMap = make(map[string]string)
	})
	JustBeforeEach(func() {
		lookupFunc := func(key string) (string, bool) {
			val, ok := lookupFuncMap[key]
			return val, ok
		}
		provider = *config.NewEnvConfigProvider(lookupFunc)
	})
	Describe("Loads the configuration", func() {
		var (
			cfg *config.Config
			err error
		)
		JustBeforeEach(func() {
			cfg, err = provider.LoadConfig()
		})
		It("returns configuration structure", func() {
			Expect(cfg).NotTo(BeNil())
		})
		It("does not error", func() {
			Expect(err).To(BeNil())
		})
		When("namespace env variable is set", func() {
			BeforeEach(func() {
				lookupFuncMap[config.PodNamespaceEnvVar] = "ns-1"
			})
			It("has valid namespace", func() {
				Expect(cfg.Namespace).To(Equal(lookupFuncMap[config.PodNamespaceEnvVar]))
			})
		})
		When("manager check interval env variable is set", func() {
			var interval int
			BeforeEach(func() {
				interval = config.MgrCheckIntervalSecDefault + 11
				lookupFuncMap[config.MgrCheckIntervalSecEnvVar] = fmt.Sprintf("%d", interval)
			})
			It("has valid check manager interval", func() {
				Expect(cfg.MgrCheckIntervalSec).To(Equal(interval))
			})
		})
		When("leader election env variable is set", func() {
			BeforeEach(func() {
				lookupFuncMap[config.LeaderElectionEnvVar] = "true"
			})
			It("has valid leader election", func() {
				Expect(cfg.LeaderElection).To(BeTrue())
			})
		})
		When("metrics probe bind addr variable is set", func() {
			var bindAddr string
			BeforeEach(func() {
				bindAddr = "127.0.0.1:1234"
				lookupFuncMap[config.MetricsProbeBindAddrEnvVar] = bindAddr
			})
			It("has valid metrics probe bind addr", func() {
				Expect(cfg.MetricsProbeBindAddr).To(Equal(bindAddr))
			})
		})
		When("health probe bind addr variable is set", func() {
			var bindAddr string
			BeforeEach(func() {
				bindAddr = "127.0.0.1:1234"
				lookupFuncMap[config.HealthProbeBindAddrEnvVar] = bindAddr
			})
			It("has valid health probe bind addr", func() {
				Expect(cfg.HealthProbeBindAddr).To(Equal(bindAddr))
			})
		})
		When("secure metrics env variable is set", func() {
			BeforeEach(func() {
				lookupFuncMap[config.SecureMetricsEnvVar] = "true"
			})
			It("has valid secure metrics", func() {
				Expect(cfg.SecureMetrics).To(BeTrue())
			})
		})
		When("zap log level variable is set", func() {
			var logLevel int
			BeforeEach(func() {
				logLevel = -5
				lookupFuncMap[config.ZapLogLevelEnvVar] = fmt.Sprintf("%d", logLevel)
			})
			It("has valid check manager interval", func() {
				Expect(cfg.ZapLogLevel).To(Equal(logLevel))
			})
		})
	})
})
