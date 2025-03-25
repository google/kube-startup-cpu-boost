// Copyright 2025 Google LLC
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

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// ClusterInfo provides information about the cluster
type ClusterInfo interface {
	//GetClusterVersion returns the cluster version
	GetClusterVersion() (*version.Info, error)
	// GetFeatureGates returns the supported feature gates
	GetFeatureGates() (FeatureGates, error)
}

// clusterInfo implements ClusterInfo with a discovery client
type clusterInfo struct {
	ctx            context.Context
	client         kubernetes.Interface
	clusterVersion *version.Info
	fgValidator    FeatureGateValidator
}

func NewClusterInfoFromConfig(ctx context.Context, config *restclient.Config) (ClusterInfo, error) {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	fgValidator := NewMetricsFeatureGateValidator(ctx, client.RESTClient())
	return NewClusterInfo(ctx, client, fgValidator), nil
}

func NewClusterInfo(ctx context.Context, client kubernetes.Interface,
	fgValidator FeatureGateValidator) ClusterInfo {
	return &clusterInfo{
		ctx:         ctx,
		client:      client,
		fgValidator: fgValidator,
	}
}

func (c *clusterInfo) GetClusterVersion() (*version.Info, error) {
	if c.clusterVersion != nil {
		return c.clusterVersion, nil
	}
	clusterVersion, err := c.client.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	c.clusterVersion = clusterVersion
	return clusterVersion, nil
}

func (c *clusterInfo) GetFeatureGates() (FeatureGates, error) {
	return c.fgValidator.GetFeatureGates()
}
