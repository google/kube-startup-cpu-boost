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

	if v, ok := p.lookupFunc(PodNamespaceEnvVar); ok {
		config.Namespace = v
	}
	if v, ok := p.lookupFunc(MgrCheckIntervalSecEnvVar); ok {
		intVal, err := strconv.Atoi(v)
		config.MgrCheckIntervalSec = intVal
		if err != nil {
			errs = append(errs, fmt.Errorf("%s value is not an int: %s", MgrCheckIntervalSecEnvVar, err))
		}
	}
	if v, ok := p.lookupFunc(LeaderElectionEnvVar); ok {
		boolVal, err := strconv.ParseBool(v)
		config.LeaderElection = boolVal
		if err != nil {
			errs = append(errs, fmt.Errorf("%s value is not a bool: %s", LeaderElectionEnvVar, err))
		}
	}
	if v, ok := p.lookupFunc(MetricsProbeBindAddrEnvVar); ok {
		config.MetricsProbeBindAddr = v
	}
	if v, ok := p.lookupFunc(HealthProbeBindAddrEnvVar); ok {
		config.HealthProbeBindAddr = v
	}
	var err error
	if len(errs) > 0 {
		err = errors.Join(errs...)
	}
	return &config, err
}
