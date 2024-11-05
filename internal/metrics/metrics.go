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

// Package metrics provides Kube Startup CPU Boost metrics for Prometheus.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const KubeStartupCPUBoostSubsystem = "boost"

var (
	// regularBoostConfigurations is a number of the container
	// boost configurations registered in a boost manager.
	regularBoostConfigurations *prometheus.GaugeVec
	// boostContainersTotal is a number of a containers which
	// CPU resources were increased.
	boostContainersTotal *prometheus.CounterVec
	// boostContainersActive is a number of a containers which
	// CPU resources and not yet reverted to their original values.
	boostContainersActive *prometheus.GaugeVec
)

// init initializes all of the Kube Startup CPU Boost metrics.
func init() {
	regularBoostConfigurations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: KubeStartupCPUBoostSubsystem,
			Name:      "regular_boosts",
			Help:      "Number of registered Kube Startup CPU Boost configurations",
		}, []string{"namespace"},
	)
	boostContainersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: KubeStartupCPUBoostSubsystem,
			Name:      "containers_total",
			Help:      "Number of a containers which CPU resources were increased",
		}, []string{"namespace", "boost"},
	)
	boostContainersActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Subsystem: KubeStartupCPUBoostSubsystem,
			Name:      "containers_active",
			Help:      "Number of a containers which CPU resources and not yet reverted to their original values",
		}, []string{"namespace", "boost"},
	)
}

// Register registers all of the Kube Startup CPU Boost metrics
// in the Prometheus registry.
func Register() {
	metrics.Registry.MustRegister(
		regularBoostConfigurations,
		boostContainersTotal,
		boostContainersActive,
	)
}

// NewRegularBoostConfiguration updates all of the relevant metrics when
// a new boost configuration is created
func NewRegularBoostConfiguration(namespace string) {
	regularBoostConfigurations.With(
		prometheus.Labels{"namespace": namespace}).
		Inc()
}

// DeleteRegularBoostConfiguration updates all of the relevant metrics when
// a boost configuration is deleted
func DeleteRegularBoostConfiguration(namespace string) {
	regularBoostConfigurations.With(
		prometheus.Labels{"namespace": namespace}).
		Dec()
}

// SetBoostContainersActive updates the activeContainerBoosts metric
// for a given namespace and boost name with a given value
func SetBoostContainersActive(namespace string, boost string, value float64) {
	boostContainersActive.With(
		prometheus.Labels{"namespace": namespace, "boost": boost}).
		Set(value)
}

// AddBoostContainersTotal adds the given value to the TotalContainerBoosts
// metric for a given namespace and boost name
func AddBoostContainersTotal(namespace string, boost string, value float64) {
	boostContainersTotal.With(
		prometheus.Labels{"namespace": namespace, "boost": boost}).
		Add(value)
}

// ClearSystemMetrics clears all of the system metrics.
func ClearSystemMetrics() {
	regularBoostConfigurations.Reset()
}

// ClearBoostMetrics clears all of relevant metrics for given
// namespace and boost
func ClearBoostMetrics(namespace string, boost string) {
	boostContainersTotal.Delete(
		prometheus.Labels{"namespace": namespace, "boost": boost},
	)
	boostContainersActive.Delete(
		prometheus.Labels{"namespace": namespace, "boost": boost},
	)
}

// RegularBoostConfigurations returns value for a totalBoostConfigurations
// metric for a given namespace.
func RegularBoostConfigurations(namespace string) float64 {
	return gaugeVecValue(regularBoostConfigurations, prometheus.Labels{
		"namespace": namespace,
	})
}

// BoostContainersTotal returns value for a totalContainerBoosts
// metric for a given namespace and boost name.
func BoostContainersTotal(namespace string, boost string) float64 {
	return counterVecValue(boostContainersTotal, prometheus.Labels{
		"namespace": namespace,
		"boost":     boost,
	})
}

// BoostContainersActive returns value for a totalContainerBoosts
// metric for a given namespace and boost name.
func BoostContainersActive(namespace string, boost string) float64 {
	return gaugeVecValue(boostContainersActive, prometheus.Labels{
		"namespace": namespace,
		"boost":     boost,
	})
}

// CounterVecValue collects and returns value for a counterVec
// metric for a given labels. Created for purpose of tests.
func counterVecValue(vec *prometheus.CounterVec, labels prometheus.Labels) (value float64) {
	cnt, err := vec.GetMetricWith(labels)
	if err != nil {
		return
	}
	collect(cnt, func(m *dto.Metric) {
		value += m.GetCounter().GetValue()
	})
	return
}

// GaugeVecValue collects and returns value for a gaugeVec
// metric for a given labels. Created for purpose of tests.
func gaugeVecValue(vec *prometheus.GaugeVec, labels prometheus.Labels) (value float64) {
	cnt, err := vec.GetMetricWith(labels)
	if err != nil {
		return
	}
	collect(cnt, func(m *dto.Metric) {
		value += m.GetGauge().GetValue()
	})
	return
}

// collect collects the given prometheus collector and writes
// corresponding metric to the DTO object for further processing.
func collect(col prometheus.Collector, do func(*dto.Metric)) {
	ch := make(chan prometheus.Metric)
	go func() {
		col.Collect(ch)
		close(ch)
	}()
	for x := range ch {
		m := &dto.Metric{}
		x.Write(m)
		do(m)
	}
}
