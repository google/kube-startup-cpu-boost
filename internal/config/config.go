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

// Package config provides operator's configuration types
package config

const (
	PodNamespaceDefault           = "kube-startup-cpu-boost-system"
	MgrCheckIntervalSecDefault    = 5
	LeaderElectionDefault         = false
	MetricsProbeBindAddrDefault   = ":8080"
	HealthProbeBindAddrDefault    = ":8081"
	SecureMetricsDefault          = false
	ZapLogLevelDefault            = 0 // zapcore.InfoLevel
	ZapDevelopmentDefault         = false
	HTTP2Default                  = false
	RemoveLimitsDefault           = true
	ValidateFeatureEnabledDefault = true
	WebhookServiceNameDefault     = "kube-startup-cpu-boost-webhook-service"
	WebhookSecretNameDefault      = "kube-startup-cpu-boost-webhook-secret"
	MutatingWebhookNameDefault    = "kube-startup-cpu-boost-mutating-webhook-configuration"
	ValidatingWebhookNameDefault  = "kube-startup-cpu-boost-validating-webhook-configuration"
)

// ConfigProvider provides the Kube Startup CPU Boost configuration
type ConfigProvider interface {
	LoadConfig() *Config
}

// Config holds Kube Startup CPU configuration parameters
type Config struct {
	// Kube Startup CPU Boost namespace
	Namespace string
	// MgrCheckIntervalSec duration in seconds between boost manager checks
	// for time based boost duration policy
	MgrCheckIntervalSec int
	// LeaderElection enables leader election for controller manager
	// Enabling this will ensure there is only one active controller manager
	LeaderElection bool
	// MetricsProbeBindAddr is the address the metrics endpoint binds to
	MetricsProbeBindAddr string
	// HeathProbeBindAddr is the address the health probe endpoint binds to
	HealthProbeBindAddr string
	// SecureMetrics determines if the metrics endpoint is served securely
	SecureMetrics bool
	// ZapLogLevel determines the log level for the ZAP logger
	ZapLogLevel int
	// ZapDevelopment determines if the ZAP logger is in development mode
	ZapDevelopment bool
	// HTTP2 determines if the HTTP/2 protocol is used for webhook and metrics servers
	HTTP2 bool
	// RemoveLimits determines if CPU resource limits should be removed during boost
	RemoveLimits bool
	// ValidateFeatureEnabled determines if InPlacePodVerticalScaling feature state
	// is validated at operator's start
	ValidateFeatureEnabled bool
	// WebhookServiceName is the name of the webhook service
	WebhookServiceName string
	// WebhookSecretName is the name of the webhook TLS secret
	WebhookSecretName string
	// MutatingWebhookName is the name of the MutatingWebhookConfiguration
	MutatingWebhookName string
	// ValidatingWebhookName is the name of the ValidatingWebhookConfiguration
	ValidatingWebhookName string
}

// LoadDefaults loads the default configuration values
func (c *Config) LoadDefaults() {
	c.Namespace = PodNamespaceDefault
	c.MgrCheckIntervalSec = MgrCheckIntervalSecDefault
	c.LeaderElection = LeaderElectionDefault
	c.MetricsProbeBindAddr = MetricsProbeBindAddrDefault
	c.HealthProbeBindAddr = HealthProbeBindAddrDefault
	c.SecureMetrics = SecureMetricsDefault
	c.ZapLogLevel = ZapLogLevelDefault
	c.ZapDevelopment = ZapDevelopmentDefault
	c.HTTP2 = HTTP2Default
	c.RemoveLimits = RemoveLimitsDefault
	c.ValidateFeatureEnabled = ValidateFeatureEnabledDefault
	c.WebhookServiceName = WebhookServiceNameDefault
	c.WebhookSecretName = WebhookSecretNameDefault
	c.MutatingWebhookName = MutatingWebhookNameDefault
	c.ValidatingWebhookName = ValidatingWebhookNameDefault
}
