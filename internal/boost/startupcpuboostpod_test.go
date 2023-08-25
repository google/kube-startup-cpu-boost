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
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewStartupCPUBoostPod(t *testing.T) {
	type exp struct {
		pod *startupCPUBoostPod
		err error
	}
	inputs := []*corev1.Pod{
		{ObjectMeta: v1.ObjectMeta{Name: "pod-001", Namespace: "demo-1"}},
		{ObjectMeta: v1.ObjectMeta{Name: "pod-001", Namespace: "demo-1", Labels: map[string]string{StartupCPUBoostPodLabelKey: "boost-001"}}},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:        "pod-001",
				Namespace:   "demo-1",
				Labels:      map[string]string{StartupCPUBoostPodLabelKey: "boost-001"},
				Annotations: map[string]string{StartupCPUBoostPodAnnotationKey: "{\"timestamp\": \"2023-06-21T15:04:28.000+01:00\", \"initCPURequests\":{\"container_one\":\"800m\"}}"}},
		},
	}
	expected := []exp{
		{pod: nil, err: errInvalidPodSpecNoLabel},
		{pod: nil, err: errInvalidPodSpecNoAnnotation},
		{pod: &startupCPUBoostPod{name: "pod-001", namespace: "demo-1", boostName: "boost-001"}, err: nil},
	}
	for i := range inputs {
		result, err := NewStartupCPUBoostPod(inputs[i])
		if validateError(t, err, expected[i].err) {
			continue
		}
		if result.name != expected[i].pod.name {
			t.Errorf("input %d: name = %v; want %v", i, result.name, expected[i].pod.name)
		}
		if result.namespace != expected[i].pod.namespace {
			t.Errorf("input %d: namespace = %v; want %v", i, result.namespace, expected[i].pod.namespace)
		}
		if result.boostName != expected[i].pod.boostName {
			t.Errorf("input %d: boostName = %v; want %v", i, result.boostName, expected[i].pod.boostName)
		}
	}
}

func TestPodBoostAnnotationToPod(t *testing.T) {
	type exp struct {
		pod *startupCPUBoostPod
		err error
	}
	inputs := []string{
		"{\"timestamp\": \"2023-06-21T15:04:28.000+01:00\", \"initCPURequests\":{\"container_one\":\"800m\"}, \"initCPULimits\":{\"container_one\":\"1200m\"}}",
		"{\"timestamp\": blabla",
		"{\"initCPURequests\":{\"container_one\":\"800m\"}}",
		"{\"timestamp\": \"2023-06-21T15:04:28.000+01:00\"}",
	}
	expected := []exp{
		{
			pod: &startupCPUBoostPod{
				boostTimestamp:  mustParseTimestamp("2023-06-21T15:04:28.000+01:00"),
				initCPURequests: map[string]string{"container_one": "800m"},
				initCPULimits:   map[string]string{"container_one": "1200m"},
			},
			err: nil,
		},
		{pod: nil, err: errors.New("invalid character 'b' looking for beginning of value")},
		{pod: nil, err: errInvalidPodSpecAnnotationNoTimestamp},
		{pod: nil, err: errInvalidPodSpecAnnotationNoRequests},
	}
	for i := range inputs {
		result, err := podBoostAnnotationToPod(inputs[i])
		if validateError(t, err, expected[i].err) {
			continue
		}
		if result.boostTimestamp != expected[i].pod.boostTimestamp {
			t.Errorf("input %d: boostTimestamp = %v; want %v", i, result.boostTimestamp, expected[i].pod.boostTimestamp)
		}
		if len(result.initCPURequests) != len(expected[i].pod.initCPURequests) {
			t.Fatalf("input %d: len(initCPURequests) = %v; want %v", i, len(result.initCPURequests), len(expected[i].pod.initCPURequests))
		}
		if len(result.initCPULimits) != len(expected[i].pod.initCPULimits) {
			t.Fatalf("input %d: len(initCPULimits) = %v; want %v", i, len(result.initCPULimits), len(expected[i].pod.initCPULimits))
		}
	}
}

func validateError(t *testing.T, err error, expErr error) (cont bool) {
	if err != nil && expErr == nil {
		t.Fatalf("err = %v; want nil", err)
	}
	if err != nil && expErr != nil {
		if err.Error() != expErr.Error() {
			t.Fatalf("err = %v; want %v", err, expErr)
		}
		cont = true
	}
	if err == nil && expErr != nil {
		t.Fatalf("err = nil; want %v", expErr)
	}
	return
}

func mustParseTimestamp(timestampStr string) time.Time {
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		panic("unparsable timestamp string")
	}
	return timestamp
}
