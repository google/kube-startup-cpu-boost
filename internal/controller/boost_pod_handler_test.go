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

package controller

import (
	"testing"

	"github.com/google/kube-startup-cpu-boost/internal/boost"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestPodLabelSelector(t *testing.T) {
	h := &boostPodHandler{}
	labelSelector := h.GetPodLabelSelector()
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		t.Fatalf("could not create selector from label selector: %s", err)
	}

	inputs := []labels.Set{
		{"test": "value"},
		{"test": "value", boost.StartupCPUBoostPodLabelKey: "boost-001"},
		{"test": "value", boost.StartupCPUBoostPodLabelKey: ""},
	}
	expected := []bool{
		false,
		true,
		true,
	}
	for i := range inputs {
		result := selector.Matches(inputs[i])
		if result != expected[i] {
			t.Errorf("input %d result = %v; want %v", i, result, expected[i])
		}
	}
}
