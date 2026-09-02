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
	corev1 "k8s.io/api/core/v1"
	apiResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	BoostLabelKey        = "autoscaling.x-k8s.io/startup-cpu-boost"
	BoostAnnotationKey   = "autoscaling.x-k8s.io/startup-cpu-boost"
	BoostStateActive     = "Active"
	BoostStateReverted   = "Reverted"
	BoostStateInfeasible = "Infeasible"
	EmptyPatchString     = "{}"
)

type BoostPodLabel struct {
	BoostName string
}

func (l *BoostPodLabel) Apply(pod *corev1.Pod) {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels[BoostLabelKey] = l.BoostName
}

type BoostPodAnnotation struct {
	State           string            `json:"state,omitempty"`
	BoostTimestamp  time.Time         `json:"timestamp,omitempty"`
	InitCPURequests map[string]string `json:"initCPURequests,omitempty"`
	InitCPULimits   map[string]string `json:"initCPULimits,omitempty"`
}

func NewBoostAnnotation() *BoostPodAnnotation {
	return &BoostPodAnnotation{
		State:           BoostStateActive,
		BoostTimestamp:  time.Now(),
		InitCPURequests: make(map[string]string),
		InitCPULimits:   make(map[string]string),
	}
}

func (a *BoostPodAnnotation) ToJSON() string {
	result, err := json.Marshal(a)
	if err != nil {
		panic("failed to marshall to JSON: " + err.Error())
	}
	return string(result)
}

// HasInitCPUResources returns true if the annotation contains any init CPU resources.
func (a *BoostPodAnnotation) HasInitCPUResources() bool {
	return len(a.InitCPURequests) > 0 || len(a.InitCPULimits) > 0
}

func (a *BoostPodAnnotation) UpdateInitResources(
	containerName string, resources corev1.ResourceRequirements) {
	if cpuRequests, ok := resources.Requests[corev1.ResourceCPU]; ok {
		a.InitCPURequests[containerName] = cpuRequests.String()
	}
	if cpuLimits, ok := resources.Limits[corev1.ResourceCPU]; ok {
		a.InitCPULimits[containerName] = cpuLimits.String()
	}
}

func (a *BoostPodAnnotation) Apply(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations[BoostAnnotationKey] = a.ToJSON()
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

// RevertResourceBoostWithBoostOnRestart reverts the boost resources, labels and annotations
// when boost on pod restart feature is enabled.
func RevertResourceBoostWithBoostOnRestart(pod *corev1.Pod) error {
	if err := revertBoostResources(pod); err != nil {
		return err
	}
	return revertBoostLabelsWithBoostOnRestart(pod)
}

func revertBoostLabelsWithBoostOnRestart(pod *corev1.Pod) error {
	boostAnnotation, err := BoostAnnotationFromPod(pod)
	if err != nil {
		return err
	}
	boostAnnotation.State = BoostStateReverted
	pod.Annotations[BoostAnnotationKey] = boostAnnotation.ToJSON()
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

type staticPatch struct {
	original client.Object
	updated  client.Object
}

func (p *staticPatch) Type() types.PatchType {
	return types.MergePatchType
}

func (p *staticPatch) Data(obj client.Object) ([]byte, error) {
	originalJSON, err := json.Marshal(p.original)
	if err != nil {
		return nil, err
	}
	updatedJSON, err := json.Marshal(p.updated)
	if err != nil {
		return nil, err
	}
	return jsonpatch.CreateMergePatch(originalJSON, updatedJSON)
}

// NewApplyBoostMetadataPatch creates new patch for applying boost metadata modifications
func NewApplyBoostMetadataPatch(originalPod, updatedPod *corev1.Pod) client.Patch {
	updatedPodStripped := updatedPod.DeepCopy()
	updatedPodStripped.Spec = originalPod.Spec
	return &staticPatch{
		original: originalPod,
		updated:  updatedPodStripped,
	}
}

// NewApplyBoostResourcesPatch creates new patch for applying boost resource modifications
func NewApplyBoostResourcesPatch(originalPod, updatedPod *corev1.Pod) client.Patch {
	updatedPodStripped := updatedPod.DeepCopy()
	updatedPodStripped.ObjectMeta = originalPod.ObjectMeta
	return &staticPatch{
		original: originalPod,
		updated:  updatedPodStripped,
	}
}
