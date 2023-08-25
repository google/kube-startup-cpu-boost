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

package boost

import (
	"errors"
	"sync"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	errPodAlreadyExists = errors.New("pod already exists")
)

type startupCPUBoost struct {
	sync.RWMutex
	name      string
	namespace string
	percent   int64
	time      time.Duration
	selector  labels.Selector
	pods      map[string]*startupCPUBoostPod
}

func (b *startupCPUBoost) Name() string {
	return b.name
}

func (b *startupCPUBoost) Namespace() string {
	return b.namespace
}

func (b *startupCPUBoost) BoostPercent() int64 {
	return b.percent
}

func (b *startupCPUBoost) AddPod(pod *startupCPUBoostPod) error {
	b.Lock()
	defer b.Unlock()
	if _, ok := b.pods[pod.name]; ok {
		return errPodAlreadyExists
	}
	b.pods[pod.name] = pod
	return nil
}

func newStartupCPUBoost(boost *autoscaling.StartupCPUBoost) (*startupCPUBoost, error) {
	selector, err := metav1.LabelSelectorAsSelector(&boost.Selector)
	if err != nil {
		return nil, err
	}
	return &startupCPUBoost{
		name:      boost.Name,
		namespace: boost.Namespace,
		selector:  selector,
		percent:   boost.Spec.BoostPercent,
		time:      time.Duration(boost.Spec.TimePeriod) * time.Second,
		pods:      make(map[string]*startupCPUBoostPod),
	}, nil
}
