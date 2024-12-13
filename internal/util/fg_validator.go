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

package util

import (
	"context"
	"errors"
	"strings"

	promclient "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type FeatureGates map[string]map[string]bool
type FeatureGateStage string

const (
	metricsEndpoint             = "/metrics"
	k8sFeatureEnabledMetricName = "kubernetes_feature_enabled"
	k8sFeatureEnabledNameLabel  = "name"
	k8sFeatureEnabledStageLabel = "stage"
)

func (f FeatureGates) IsEnabled(featureGate string, stage FeatureGateStage) bool {
	stages, ok := f[featureGate]
	if !ok {
		return false
	}
	for stageName, enabled := range stages {
		if strings.ToUpper(stageName) == string(stage) {
			return enabled
		}
	}
	return false
}

func (f FeatureGates) IsEnabledAnyStage(featureGate string) bool {
	stages, ok := f[featureGate]
	if !ok {
		return false
	}
	for _, enabled := range stages {
		if enabled {
			return true
		}
	}
	return false
}

// FeatureGateValidator validates if a given feature gates are enabled on a cluster
type FeatureGateValidator interface {
	// GetFeatureGates returns the supported feature gates
	GetFeatureGates() (FeatureGates, error)
}

// metricsFeatureGateValidator validates if a given feature gates are enabled on a cluster
// using /metrics endpoint
type metricsFeatureGateValidator struct {
	client restclient.Interface
	ctx    context.Context
}

func NewMetricsFeatureGateValidatorFromConfig(ctx context.Context, config *restclient.Config) (FeatureGateValidator, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return NewMetricsFeatureGateValidator(ctx, client.RESTClient()), nil
}

func NewMetricsFeatureGateValidator(ctx context.Context, RESTClient restclient.Interface) FeatureGateValidator {
	return &metricsFeatureGateValidator{
		client: RESTClient,
		ctx:    ctx,
	}
}

func (m *metricsFeatureGateValidator) GetFeatureGates() (FeatureGates, error) {
	reader, err := m.client.Get().AbsPath(metricsEndpoint).Stream(m.ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}
	featureGates := make(map[string]map[string]bool)
	for name, family := range metricFamilies {
		if name != k8sFeatureEnabledMetricName {
			continue
		}
		for _, metric := range family.Metric {
			name, stage, err := getNameAndStage(metric.GetLabel())
			if err != nil {
				continue
			}
			if featureGates[name] == nil {
				featureGates[name] = make(map[string]bool)
			}
			featureGates[name][stage] = false
			if metric.GetGauge().GetValue() == 1 {
				featureGates[name][stage] = true
			}
		}
	}
	return featureGates, nil
}

func getNameAndStage(labels []*promclient.LabelPair) (string, string, error) {
	var name string
	var stagePtr *string
	for _, label := range labels {
		if label.GetName() == k8sFeatureEnabledNameLabel {
			name = label.GetValue()
		}
		if label.GetName() == k8sFeatureEnabledStageLabel {
			stagePtr = label.Value
		}
	}
	if name == "" || stagePtr == nil {
		return "", "", errors.New("missing name and stage label")
	}
	return name, *stagePtr, nil
}
