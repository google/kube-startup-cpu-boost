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
	bpod "github.com/google/kube-startup-cpu-boost/internal/boost/pod"
	"github.com/google/kube-startup-cpu-boost/internal/controller"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("BoostPodHandler", func() {
	var (
		mockCtrl    *gomock.Controller
		mgrMock     *mock.MockManager
		mgrMockCall *gomock.Call
		podHandler  controller.BoostPodHandler
		wq          workqueue.TypedRateLimitingInterface[reconcile.Request]
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mgrMock = mock.NewMockManager(mockCtrl)
		wq = workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
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
			mgrMockCall = mgrMock.EXPECT().HandlePodEvent(
				gomock.Any(),
				gomock.Eq(&bpod.PodEvent{Type: bpod.PodEventTypePodCreated, Pod: pod}),
			)
		})
		JustBeforeEach(func() {
			podHandler.Create(context.TODO(), createEvent, wq)
		})
		When("There is no boost matching the POD", func() {
			BeforeEach(func() {
				mgrMockCall.Return(nil, nil)
			})
			It("sends a valid call to the boost manager", func() {
				mgrMockCall.Times(1)
			})
		})
		When("There is a boost matching the POD", func() {
			BeforeEach(func() {
				boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
				boostMock.EXPECT().Name().Return(specTemplate.Name).MinTimes(1)
				boostMock.EXPECT().Namespace().Return(specTemplate.Namespace).MinTimes(1)
				mgrMockCall.Return(boostMock, nil)
			})
			It("sends reconciliation request", func() {
				Expect(wq.Len()).To(Equal(1))
				req, _ := wq.Get()
				Expect(req.Name).To(Equal(specTemplate.Name))
				Expect(req.Namespace).To(Equal(specTemplate.Namespace))
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
			mgrMockCall = mgrMock.EXPECT().HandlePodEvent(
				gomock.Any(),
				gomock.Eq(&bpod.PodEvent{Type: bpod.PodEventTypePodDeleted, Pod: pod}),
			)
		})
		JustBeforeEach(func() {
			podHandler.Delete(context.TODO(), deleteEvent, wq)
		})
		When("There is no boost matching the POD", func() {
			BeforeEach(func() {
				mgrMockCall.Return(nil, nil)
			})
			It("sends a valid call to the boost manager", func() {
				mgrMockCall.Times(1)
			})
		})
		When("There is a boost matching the POD", func() {
			BeforeEach(func() {
				boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
				boostMock.EXPECT().Name().Return(specTemplate.Name).MinTimes(1)
				boostMock.EXPECT().Namespace().Return(specTemplate.Namespace).MinTimes(1)
				mgrMockCall.Return(boostMock, nil)
			})
			It("sends a valid call to the boost manager and a boost", func() {
				mgrMockCall.Times(1)
			})
			It("sends reconciliation request", func() {
				Expect(wq.Len()).To(Equal(1))
				req, _ := wq.Get()
				Expect(req.Name).To(Equal(specTemplate.Name))
				Expect(req.Namespace).To(Equal(specTemplate.Namespace))
			})
		})
	})
	Describe("Receives an update event", func() {
		var (
			oldPod      *corev1.Pod
			newPod      *corev1.Pod
			updateEvent event.UpdateEvent
		)
		BeforeEach(func() {
			oldPod = podTemplate.DeepCopy()
			newPod = podTemplate.DeepCopy()
			updateEvent = event.UpdateEvent{
				ObjectNew: newPod,
				ObjectOld: oldPod,
			}
		})
		JustBeforeEach(func() {
			podHandler.Update(context.TODO(), updateEvent, wq)
		})
		When("Pod status conditions has not change", func() {
			BeforeEach(func() {
				mgrMockCall = mgrMock.EXPECT().HandlePodEvent(
					gomock.Any(),
					gomock.Eq(&bpod.PodEvent{Type: bpod.PodEventTypeConditionChanged, Pod: newPod}),
				).Times(0)
			})
			It("does not send reconciliation request", func() {
				Expect(wq.Len()).To(Equal(0))
			})
		})
		When("Pod status conditions has changed", func() {
			BeforeEach(func() {
				oldPod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				}
				newPod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}
				mgrMockCall = mgrMock.EXPECT().HandlePodEvent(
					gomock.Any(),
					gomock.Eq(&bpod.PodEvent{Type: bpod.PodEventTypeConditionChanged, Pod: newPod}),
				)
			})
			When("There is no boost matching the POD", func() {
				BeforeEach(func() {
					mgrMockCall.Return(nil, nil)
				})
				It("sends a valid call to the boost manager", func() {
					mgrMockCall.Times(1)
				})
			})
			When("There is a boost matching the POD", func() {
				BeforeEach(func() {
					boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
					boostMock.EXPECT().Name().Return(specTemplate.Name).MinTimes(1)
					boostMock.EXPECT().Namespace().Return(specTemplate.Namespace).MinTimes(1)
					mgrMockCall.Return(boostMock, nil)
				})
				It("sends a valid call to the boost manager and a boost", func() {
					mgrMockCall.Times(1)
				})
				It("sends reconciliation request", func() {
					Expect(wq.Len()).To(Equal(1))
					req, _ := wq.Get()
					Expect(req.Name).To(Equal(specTemplate.Name))
					Expect(req.Namespace).To(Equal(specTemplate.Namespace))
				})
			})
		})
		When("Container is restarting", func() {
			BeforeEach(func() {
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}},
				}
				mgrMockCall = mgrMock.EXPECT().HandlePodEvent(
					gomock.Any(),
					gomock.Eq(&bpod.PodEvent{
						Type:                     bpod.PodEventTypeContainerRestarting,
						Pod:                      newPod,
						RestartingContainerNames: []string{"container-one"},
					}),
				)
			})
			When("There is no boost matching the POD", func() {
				BeforeEach(func() {
					mgrMockCall.Return(nil, nil)
				})
				It("sends a valid call to the boost manager", func() {
					mgrMockCall.Times(1)
				})
			})
			When("There is a boost matching the POD", func() {
				BeforeEach(func() {
					boostMock := mock.NewMockStartupCPUBoost(mockCtrl)
					boostMock.EXPECT().Name().Return(specTemplate.Name).MinTimes(1)
					boostMock.EXPECT().Namespace().Return(specTemplate.Namespace).MinTimes(1)
					mgrMockCall.Return(boostMock, nil)
				})
				It("sends a valid call to the boost manager and a boost", func() {
					mgrMockCall.Times(1)
				})
				It("sends reconciliation request", func() {
					Expect(wq.Len()).To(Equal(1))
					req, _ := wq.Get()
					Expect(req.Name).To(Equal(specTemplate.Name))
					Expect(req.Namespace).To(Equal(specTemplate.Namespace))
				})
			})
		})
		When("Container is terminating but pod is being deleted", func() {
			BeforeEach(func() {
				now := metav1.Now()
				newPod.DeletionTimestamp = &now
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 143}}},
				}
				mgrMock.EXPECT().HandlePodEvent(gomock.Any(), gomock.Any()).Times(0)
			})
			It("does not trigger restart event or reconciliation", func() {
				Expect(wq.Len()).To(Equal(0))
			})
		})
		When("Container is terminating but pod is in terminal phase", func() {
			BeforeEach(func() {
				newPod.Status.Phase = corev1.PodFailed
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}},
				}
				mgrMock.EXPECT().HandlePodEvent(gomock.Any(), gomock.Any()).Times(0)
			})
			It("does not trigger restart event or reconciliation", func() {
				Expect(wq.Len()).To(Equal(0))
			})
		})
		When("Container terminates on RestartPolicyNever pod", func() {
			BeforeEach(func() {
				newPod.Spec.RestartPolicy = corev1.RestartPolicyNever
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}},
				}
				mgrMock.EXPECT().HandlePodEvent(gomock.Any(), gomock.Any()).Times(0)
			})
			It("does not trigger restart event or reconciliation", func() {
				Expect(wq.Len()).To(Equal(0))
			})
		})
		When("Container terminates cleanly with 0 on RestartPolicyOnFailure pod", func() {
			BeforeEach(func() {
				newPod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0}}},
				}
				mgrMock.EXPECT().HandlePodEvent(gomock.Any(), gomock.Any()).Times(0)
			})
			It("does not trigger restart event or reconciliation", func() {
				Expect(wq.Len()).To(Equal(0))
			})
		})
		When("Container terminates with error on RestartPolicyOnFailure pod", func() {
			BeforeEach(func() {
				newPod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{Name: "container-one", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1}}},
				}
				mgrMock.EXPECT().HandlePodEvent(
					gomock.Any(),
					gomock.Eq(&bpod.PodEvent{
						Type:                     bpod.PodEventTypeContainerRestarting,
						Pod:                      newPod,
						RestartingContainerNames: []string{"container-one"},
					}),
				).Return(nil, nil)
			})
			It("triggers restart event", func() {
				Expect(wq.Len()).To(Equal(0))
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
				Expect(m.Key).To(Equal(bpod.BoostLabelKey))
			})
			It("has empty values list", func() {
				Expect(m.Values).To(HaveLen(0))
			})
		})
	})
})
