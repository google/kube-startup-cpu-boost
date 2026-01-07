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

// Package pod contains implementation of startup-cpu-boost POD manipulation
// functions
package pod

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BoostLabelKey      = "autoscaling.x-k8s.io/startup-cpu-boost"
	BoostAnnotationKey = "autoscaling.x-k8s.io/startup-cpu-boost"
	EmptyPatchString   = "{}"
)

type BoostPodAnnotation struct {
	BoostTimestamp  time.Time         `json:"timestamp,omitempty"`
	InitCPURequests map[string]string `json:"initCPURequests,omitempty"`
	InitCPULimits   map[string]string `json:"initCPULimits,omitempty"`

	// ActivationState tracks boost activation state for runtime triggers
	// These fields are optional and maintain backward compatibility
	ActivationState *ActivationState `json:"activationState,omitempty"`
}

// ActivationState tracks the state of boost activations for a pod
type ActivationState struct {
	// CurrentActivation is the currently active boost activation, if any
	CurrentActivation *ActivationStateEntry `json:"currentActivation,omitempty"`

	// LastActivationTime tracks the last activation time per trigger type
	// Used for cooldown minimum interval enforcement
	LastActivationTime map[string]string `json:"lastActivationTime,omitempty"`

	// ActivationHistory is a list of activation timestamps (RFC3339 format)
	// Used for rate limiting (max activations per hour)
	// Only keeps activations from the last hour
	ActivationHistory []string `json:"activationHistory,omitempty"`
}

// ActivationStateEntry represents a single activation state entry
type ActivationStateEntry struct {
	// TriggerType is the type of trigger that activated this boost
	TriggerType autoscaling.BoostTriggerType `json:"triggerType"`

	// StartTime is when the boost was activated (RFC3339 format)
	StartTime string `json:"startTime"`

	// ExpiryConditionType indicates how this activation expires
	ExpiryConditionType string `json:"expiryConditionType"`

	// ExpiryFixedDuration is set when expiry is based on fixed duration (seconds)
	ExpiryFixedDuration *int64 `json:"expiryFixedDuration,omitempty"`

	// ExpiryPodCondition is set when expiry is based on pod condition
	ExpiryPodCondition *PodConditionExpiryEntry `json:"expiryPodCondition,omitempty"`
}

// PodConditionExpiryEntry represents pod condition-based expiry
type PodConditionExpiryEntry struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type mutatePodFunc func(pod *corev1.Pod) error

func NewBoostAnnotation() *BoostPodAnnotation {
	return &BoostPodAnnotation{
		BoostTimestamp:  time.Now(),
		InitCPURequests: make(map[string]string),
		InitCPULimits:   make(map[string]string),
		ActivationState: &ActivationState{
			LastActivationTime: make(map[string]string),
			ActivationHistory:  make([]string, 0),
		},
	}
}

func (a BoostPodAnnotation) ToJSON() string {
	result, err := json.Marshal(a)
	if err != nil {
		panic("failed to marshall to JSON: " + err.Error())
	}
	return string(result)
}

func BoostAnnotationFromPod(pod *corev1.Pod) (*BoostPodAnnotation, error) {
	annotation := &BoostPodAnnotation{}
	data, ok := pod.Annotations[BoostAnnotationKey]
	if !ok {
		return nil, errors.New("boost annotation not found")
	}
	if err := json.Unmarshal([]byte(data), annotation); err != nil {
		return nil, err
	}
	// Ensure ActivationState is initialized for backward compatibility
	if annotation.ActivationState == nil {
		annotation.ActivationState = &ActivationState{
			LastActivationTime: make(map[string]string),
			ActivationHistory:  make([]string, 0),
		}
	}
	if annotation.ActivationState.LastActivationTime == nil {
		annotation.ActivationState.LastActivationTime = make(map[string]string)
	}
	if annotation.ActivationState.ActivationHistory == nil {
		annotation.ActivationState.ActivationHistory = make([]string, 0)
	}
	return annotation, nil
}

func RevertResourceBoost(pod *corev1.Pod) error {
	if err := revertBoostResources(pod); err != nil {
		return err
	}
	return revertBoostLabels(pod)
}

func revertBoostLabels(pod *corev1.Pod) error {
	delete(pod.Labels, BoostLabelKey)
	delete(pod.Annotations, BoostAnnotationKey)
	return nil
}

func revertBoostResources(pod *corev1.Pod) error {
	annotation, err := BoostAnnotationFromPod(pod)
	if err != nil {
		return fmt.Errorf("failed to get boost annotation from pod: %s", err)
	}
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if request, ok := annotation.InitCPURequests[container.Name]; ok {
			if reqQuantity, err := apiResource.ParseQuantity(request); err == nil {
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceCPU] = reqQuantity
			} else {
				return fmt.Errorf("failed to parse CPU request: %s", err)
			}
		}
		if limit, ok := annotation.InitCPULimits[container.Name]; ok {
			if limitQuantity, err := apiResource.ParseQuantity(limit); err == nil {
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Limits[corev1.ResourceCPU] = limitQuantity
			} else {
				return fmt.Errorf("failed to parse CPU limit: %s", err)
			}
		}
	}
	return nil
}

func buildPodPatch(pod *corev1.Pod, mutatePodFunc mutatePodFunc) ([]byte, error) {
	podJSON, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	if err := mutatePodFunc(pod); err != nil {
		return nil, err
	}
	updatedPodJSON, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	return jsonpatch.CreateMergePatch(podJSON, updatedPodJSON)

}

func NewRevertBoostLabelsPatch() client.Patch {
	return &revertBoostLabelsPatch{}
}

type revertBoostLabelsPatch struct {
}

func (p *revertBoostLabelsPatch) Type() types.PatchType {
	return types.MergePatchType
}

func (p *revertBoostLabelsPatch) Data(obj client.Object) ([]byte, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, errors.New("revertBoostLabelsPatch applies only on *corev1.Pod objects")
	}
	return buildPodPatch(pod, revertBoostLabels)
}

func NewRevertBootsResourcesPatch() client.Patch {
	return &revertBoostResourcesPatch{}
}

type revertBoostResourcesPatch struct {
}

func (p *revertBoostResourcesPatch) Type() types.PatchType {
	return types.MergePatchType
}

func (p *revertBoostResourcesPatch) Data(obj client.Object) ([]byte, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, errors.New("revertBoostResourcesPatch applies only on *corev1.Pod objects")
	}
	patchData, err := buildPodPatch(pod, revertBoostResources)
	if err != nil {
		return []byte(EmptyPatchString), nil
	}
	return patchData, nil
}

// GetActivationState returns the activation state, initializing it if needed
func (a *BoostPodAnnotation) GetActivationState() *ActivationState {
	if a.ActivationState == nil {
		a.ActivationState = &ActivationState{
			LastActivationTime: make(map[string]string),
			ActivationHistory:  make([]string, 0),
		}
	}
	if a.ActivationState.LastActivationTime == nil {
		a.ActivationState.LastActivationTime = make(map[string]string)
	}
	if a.ActivationState.ActivationHistory == nil {
		a.ActivationState.ActivationHistory = make([]string, 0)
	}
	return a.ActivationState
}

// SetCurrentActivation sets the current active boost activation
func (a *BoostPodAnnotation) SetCurrentActivation(triggerType autoscaling.BoostTriggerType, startTime time.Time, expiryType string, expiryFixedDuration *int64, expiryPodCondition *PodConditionExpiryEntry) {
	state := a.GetActivationState()
	state.CurrentActivation = &ActivationStateEntry{
		TriggerType:          triggerType,
		StartTime:            startTime.Format(time.RFC3339),
		ExpiryConditionType:  expiryType,
		ExpiryFixedDuration:  expiryFixedDuration,
		ExpiryPodCondition:   expiryPodCondition,
	}
}

// ClearCurrentActivation clears the current active boost activation
func (a *BoostPodAnnotation) ClearCurrentActivation() {
	state := a.GetActivationState()
	state.CurrentActivation = nil
}

// GetCurrentActivation returns the current active activation, if any
func (a *BoostPodAnnotation) GetCurrentActivation() *ActivationStateEntry {
	if a.ActivationState == nil {
		return nil
	}
	return a.ActivationState.CurrentActivation
}

// GetLastActivationTime returns the last activation time for a given trigger type
func (a *BoostPodAnnotation) GetLastActivationTime(triggerType autoscaling.BoostTriggerType) (time.Time, bool) {
	state := a.GetActivationState()
	timeStr, ok := state.LastActivationTime[string(triggerType)]
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// SetLastActivationTime sets the last activation time for a given trigger type
func (a *BoostPodAnnotation) SetLastActivationTime(triggerType autoscaling.BoostTriggerType, activationTime time.Time) {
	state := a.GetActivationState()
	state.LastActivationTime[string(triggerType)] = activationTime.Format(time.RFC3339)
}

// AddActivationToHistory adds an activation timestamp to the history
// and removes activations older than 1 hour
func (a *BoostPodAnnotation) AddActivationToHistory(activationTime time.Time) {
	state := a.GetActivationState()
	now := time.Now()
	cutoffTime := now.Add(-1 * time.Hour)

	// Add new activation
	state.ActivationHistory = append(state.ActivationHistory, activationTime.Format(time.RFC3339))

	// Remove activations older than 1 hour
	filtered := make([]string, 0, len(state.ActivationHistory))
	for _, timeStr := range state.ActivationHistory {
		t, err := time.Parse(time.RFC3339, timeStr)
		if err == nil && t.After(cutoffTime) {
			filtered = append(filtered, timeStr)
		}
	}
	state.ActivationHistory = filtered
}

// GetActivationHistoryCount returns the number of activations in the last hour
func (a *BoostPodAnnotation) GetActivationHistoryCount() int {
	if a.ActivationState == nil || a.ActivationState.ActivationHistory == nil {
		return 0
	}
	return len(a.ActivationState.ActivationHistory)
}
