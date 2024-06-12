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

package config

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	PodNamespaceEnvVar         = "POD_NAMESPACE"
	MgrCheckIntervalSecEnvVar  = "MGR_CHECK_INTERVAL"
	LeaderElectionEnvVar       = "LEADER_ELECTION"
	MetricsProbeBindAddrEnvVar = "METRICS_PROBE_BIND_ADDR"
	HealthProbeBindAddrEnvVar  = "HEALTH_PROBE_BIND_ADDR"
	SecureMetricsEnvVar        = "SECURE_METRICS"
	ZapLogLevelEnvVar          = "ZAP_LOG_LEVEL"
	ZapDevelopmentEnvVar       = "ZAP_DEVELOPMENT"
	HTTP2EnvVar                = "HTTP2"
)

type LookupEnvFunc func(key string) (string, bool)

type EnvConfigProvider struct {
	lookupFunc LookupEnvFunc
}

func NewEnvConfigProvider(f LookupEnvFunc) *EnvConfigProvider {
	return &EnvConfigProvider{
		lookupFunc: f,
	}
}

func (p *EnvConfigProvider) LoadConfig() (*Config, error) {
	var errs []error
	var config Config
	config.LoadDefaults()
	p.loadPodNamespace(&config)
	errs = p.loadMgrCheckIntervalSec(&config, errs)
	errs = p.loadLeaderElection(&config, errs)
	p.loadMetricsProbeBindAddr(&config)
	p.loadHealthProbeBindAddr(&config)
	errs = p.loadSecureMetrics(&config, errs)
	errs = p.loadZapLogLevel(&config, errs)
	errs = p.loadZapDevelopment(&config, errs)
	errs = p.loadHTTP2(&config, errs)
	var err error
	if len(errs) > 0 {
		err = errors.Join(errs...)
	}
	return &config, err
}

func (p *EnvConfigProvider) loadPodNamespace(config *Config) {
	if v, ok := p.lookupFunc(PodNamespaceEnvVar); ok {
		config.Namespace = v
	}
}

func (p *EnvConfigProvider) loadMgrCheckIntervalSec(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(MgrCheckIntervalSecEnvVar); ok {
		intVal, err := strconv.Atoi(v)
		config.MgrCheckIntervalSec = intVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not an int: %s", MgrCheckIntervalSecEnvVar, err))
		}
	}
	return
}

func (p *EnvConfigProvider) loadLeaderElection(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(LeaderElectionEnvVar); ok {
		boolVal, err := strconv.ParseBool(v)
		config.LeaderElection = boolVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not a bool: %s", LeaderElectionEnvVar, err))
		}
	}
	return
}

func (p *EnvConfigProvider) loadMetricsProbeBindAddr(config *Config) {
	if v, ok := p.lookupFunc(MetricsProbeBindAddrEnvVar); ok {
		config.MetricsProbeBindAddr = v
	}
}

func (p *EnvConfigProvider) loadHealthProbeBindAddr(config *Config) {
	if v, ok := p.lookupFunc(HealthProbeBindAddrEnvVar); ok {
		config.HealthProbeBindAddr = v
	}
}

func (p *EnvConfigProvider) loadSecureMetrics(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(SecureMetricsEnvVar); ok {
		boolVal, err := strconv.ParseBool(v)
		config.SecureMetrics = boolVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not a bool: %s", SecureMetricsEnvVar, err))
		}
	}
	return
}

func (p *EnvConfigProvider) loadZapLogLevel(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(ZapLogLevelEnvVar); ok {
		intVal, err := strconv.Atoi(v)
		config.ZapLogLevel = intVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not an int: %s", ZapLogLevelEnvVar, err))
		}
	}
	return
}

func (p *EnvConfigProvider) loadZapDevelopment(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(ZapDevelopmentEnvVar); ok {
		boolVal, err := strconv.ParseBool(v)
		config.ZapDevelopment = boolVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not a bool: %s", ZapDevelopmentEnvVar, err))
		}
	}
	return
}

func (p *EnvConfigProvider) loadHTTP2(config *Config, curErrs []error) (errs []error) {
	if v, ok := p.lookupFunc(HTTP2EnvVar); ok {
		boolVal, err := strconv.ParseBool(v)
		config.HTTP2 = boolVal
		if err != nil {
			errs = append(curErrs, fmt.Errorf("%s value is not a bool: %s", LeaderElectionEnvVar, err))
		}
	}
	return
}
