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

package controller_test

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/controller"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("BoostPodHandler", func() {
	var (
		mockCtrl    *gomock.Controller
		mgrMock     *mock.MockManager
		mgrMockCall *gomock.Call
		podHandler  controller.BoostPodHandler
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mgrMock = mock.NewMockManager(mockCtrl)
	})
	JustBeforeEach(func() {
		podHandler = controller.NewBoostPodHandler(mgrMock, logr.Discard())
	})
	Describe("Receives create event", func() {
		var (
			pod         *corev1.Pod
			createEvent event.CreateEvent
		)
		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			createEvent = event.CreateEvent{
				Object: pod,
			}
			mgrMockCall = mgrMock.EXPECT().StartupCPUBoost(
				gomock.Eq(pod.Namespace),
				gomock.Eq(specTemplate.Name),
			)
		})
		JustBeforeEach(func() {
			podHandler.Create(context.TODO(), createEvent, nil)
		})
		When("There is no boost matching the POD", func() {
			BeforeEach(func() {
				mgrMockCall.Return(nil, false)
			})
			It("sends a valid call to the boost manager", func() {
				mgrMockCall.Times(1)
			})
		})
		When("There is a boost matching the POD", func() {
			var (
				boostMockNameCall   *gomock.Call
				boostMockUpsertCall *gomock.Call
			)
			BeforeEach(func() {
				boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
				boostMockNameCall = boostMock.EXPECT().Name().
					Return(specTemplate.Name)
				boostMockUpsertCall = boostMock.EXPECT().UpsertPod(
					gomock.Any(),
					gomock.Eq(pod),
				).Return(nil)
				mgrMockCall.Return(boostMock, true)
			})
			It("sends a valid call to the boost manager and a boost", func() {
				mgrMockCall.Times(1)
				boostMockNameCall.Times(1)
				boostMockUpsertCall.Times(1)
			})
		})
	})
	Describe("Receives delete event", func() {
		var (
			pod         *corev1.Pod
			deleteEvent event.DeleteEvent
		)
		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			deleteEvent = event.DeleteEvent{
				Object: pod,
			}
			mgrMockCall = mgrMock.EXPECT().StartupCPUBoost(
				gomock.Eq(pod.Namespace),
				gomock.Eq(specTemplate.Name),
			)
		})
		JustBeforeEach(func() {
			podHandler.Delete(context.TODO(), deleteEvent, nil)
		})
		When("There is no boost matching the POD", func() {
			BeforeEach(func() {
				mgrMockCall.Return(nil, false)
			})
			It("sends a valid call to the boost manager", func() {
				mgrMockCall.Times(1)
			})
		})
		When("There is a boost matching the POD", func() {
			var (
				boostMockDeleteCall *gomock.Call
			)
			BeforeEach(func() {
				boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
				boostMockDeleteCall = boostMock.EXPECT().DeletePod(
					gomock.Any(),
					gomock.Eq(pod),
				).Return(nil)
				mgrMockCall.Return(boostMock, true)
			})
			It("sends a valid call to the boost manager and a boost", func() {
				mgrMockCall.Times(1)
				boostMockDeleteCall.Times(1)
			})
		})
	})
	Describe("Receives an update event", func() {
		var (
			pod         *corev1.Pod
			updateEvent event.UpdateEvent
		)
		BeforeEach(func() {
			pod = podTemplate.DeepCopy()
			updateEvent = event.UpdateEvent{
				ObjectNew: pod,
			}
			mgrMockCall = mgrMock.EXPECT().StartupCPUBoost(
				gomock.Eq(pod.Namespace),
				gomock.Eq(specTemplate.Name),
			)
		})
		JustBeforeEach(func() {
			podHandler.Update(context.TODO(), updateEvent, nil)
		})
		When("There is no boost matching the POD", func() {
			BeforeEach(func() {
				mgrMockCall.Return(nil, false)
			})
			It("sends a valid call to the boost manager", func() {
				mgrMockCall.Times(1)
			})
		})
		When("There is a boost matching the POD", func() {
			var (
				boostMockUpsertCall *gomock.Call
			)
			BeforeEach(func() {
				boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
				boostMockUpsertCall = boostMock.EXPECT().UpsertPod(
					gomock.Any(),
					gomock.Eq(pod),
				).Return(nil)
				mgrMockCall.Return(boostMock, true)
			})
			It("sends a valid call to the boost manager and a boost", func() {
				mgrMockCall.Times(1)
				boostMockUpsertCall.Times(1)
			})
		})
	})
	Describe("Provides the POD label selector", func() {
		var selector *metav1.LabelSelector
		JustBeforeEach(func() {
			selector = podHandler.GetPodLabelSelector()
		})
		It("returns selector with a single match expression", func() {
			Expect(selector.MatchExpressions).To(HaveLen(1))
		})
		When("The selector has a single match expression", func() {
			var m *metav1.LabelSelectorRequirement
			JustBeforeEach(func() {
				m = &selector.MatchExpressions[0]
			})
			It("has a valid key", func() {
				Expect(m.Key).To(Equal(boost.StartupCPUBoostPodLabelKey))
			})
			It("has a valid operator", func() {
				Expect(m.Key).To(Equal(boost.StartupCPUBoostPodLabelKey))
			})
			It("has empty values list", func() {
				Expect(m.Values).To(HaveLen(0))
			})
		})
	})
})
