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
		Context("Event routing and reconciliation", func() {
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
						newContainerStatus("container-one", withRunningState()),
					}
					newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
						newContainerStatus("container-one", withTerminatedState(1)),
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
			When("Pod conditions change and container restarts in the same update", func() {
				BeforeEach(func() {
					oldPod.Status.Conditions = []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionTrue},
					}
					newPod.Status.Conditions = []corev1.PodCondition{
						{Type: corev1.PodReady, Status: corev1.ConditionFalse},
					}
					oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{
						newContainerStatus("container-one", withRestartCount(1), withRunningState()),
					}
					newPod.Status.ContainerStatuses = []corev1.ContainerStatus{
						newContainerStatus("container-one",
							withRestartCount(1),
							withWaitingState("CrashLoopBackOff"),
							withLastTerminatedState(143),
						),
					}
					gomock.InOrder(
						mgrMock.EXPECT().HandlePodEvent(
							gomock.Any(),
							gomock.Eq(&bpod.PodEvent{
								Type: bpod.PodEventTypeConditionChanged,
								Pod:  newPod,
							}),
						).Return(nil, nil),
						mgrMock.EXPECT().HandlePodEvent(
							gomock.Any(),
							gomock.Eq(&bpod.PodEvent{
								Type:                     bpod.PodEventTypeContainerRestarting,
								Pod:                      newPod,
								RestartingContainerNames: []string{"container-one"},
							}),
						).Return(nil, nil),
					)
				})
				It("handles ConditionChanged before ContainerRestarting", func() {
					Expect(wq.Len()).To(Equal(0))
				})
			})
		})
		DescribeTable("Container restart detection scenarios",
			func(oldStatus, newStatus corev1.ContainerStatus, expectRestart bool, podOpts ...podOpt) {
				oldPod.Status.ContainerStatuses = []corev1.ContainerStatus{oldStatus}
				newPod.Status.ContainerStatuses = []corev1.ContainerStatus{newStatus}
				for _, opt := range podOpts {
					opt(newPod)
				}
				if expectRestart {
					mgrMock.EXPECT().HandlePodEvent(
						gomock.Any(),
						gomock.Eq(&bpod.PodEvent{
							Type:                     bpod.PodEventTypeContainerRestarting,
							Pod:                      newPod,
							RestartingContainerNames: []string{newStatus.Name},
						}),
					).Return(nil, nil)
				} else {
					mgrMock.EXPECT().HandlePodEvent(gomock.Any(), gomock.Any()).Times(0)
				}

				podHandler.Update(context.TODO(), updateEvent, wq)
				Expect(wq.Len()).To(Equal(0))
			},
			Entry("terminates while pod is being deleted",
				newContainerStatus("container-one", withRunningState()),
				newContainerStatus("container-one", withTerminatedState(143)),
				false,
				withDeletionTimestamp(),
			),
			Entry("terminates while pod is in Failed phase",
				newContainerStatus("container-one", withRunningState()),
				newContainerStatus("container-one", withTerminatedState(1)),
				false,
				withPodPhase(corev1.PodFailed),
			),
			Entry("terminates on RestartPolicyNever pod",
				newContainerStatus("container-one", withRunningState()),
				newContainerStatus("container-one", withTerminatedState(1)),
				false,
				withRestartPolicy(corev1.RestartPolicyNever),
			),
			Entry("terminates cleanly with exit code 0 on RestartPolicyOnFailure pod",
				newContainerStatus("container-one", withRunningState()),
				newContainerStatus("container-one", withTerminatedState(0)),
				false,
				withRestartPolicy(corev1.RestartPolicyOnFailure),
			),
			Entry("terminates with error on RestartPolicyOnFailure pod",
				newContainerStatus("container-one", withRunningState()),
				newContainerStatus("container-one", withTerminatedState(1)),
				true,
				withRestartPolicy(corev1.RestartPolicyOnFailure),
			),
			Entry("crashes directly into Waiting (CrashLoopBackOff)",
				newContainerStatus("container-one", withRestartCount(13), withRunningState()),
				newContainerStatus("container-one", withRestartCount(13), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				true,
			),
			Entry("in CrashLoopBackOff starts and immediately crashes into new CrashLoopBackOff",
				newContainerStatus("container-one", withRestartCount(13), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				newContainerStatus("container-one", withRestartCount(14), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				true,
			),
			Entry("initial container in Waiting starts and crashes into CrashLoopBackOff",
				newContainerStatus("container-one", withRestartCount(0), withWaitingState("ContainerCreating")),
				newContainerStatus("container-one", withRestartCount(0), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(1)),
				true,
			),
			Entry("stays in the same CrashLoopBackOff across pod updates",
				newContainerStatus("container-one", withRestartCount(13), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				newContainerStatus("container-one", withRestartCount(13), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				false,
			),
			Entry("moves from Terminated to CrashLoopBackOff for the same attempt",
				newContainerStatus("container-one", withRestartCount(1), withTerminatedState(1)),
				newContainerStatus("container-one", withRestartCount(1), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(1)),
				false,
			),
			Entry("transitions from CrashLoopBackOff directly to Running",
				newContainerStatus("container-one", withRestartCount(13), withWaitingState("CrashLoopBackOff"), withLastTerminatedState(143)),
				newContainerStatus("container-one", withRestartCount(14), withRunningState(), withLastTerminatedState(143)),
				true,
			),
			Entry("restarts rapidly from Running to Running with incremented RestartCount",
				newContainerStatus("container-one", withRestartCount(0), withRunningState()),
				newContainerStatus("container-one", withRestartCount(1), withRunningState(), withLastTerminatedState(137)),
				true,
			),
			Entry("starts for the first time (Waiting to Running with RestartCount 0)",
				newContainerStatus("container-one", withRestartCount(0), withWaitingState("ContainerCreating")),
				newContainerStatus("container-one", withRestartCount(0), withRunningState()),
				false,
			),
			Entry("previously restarted container stays Running with unchanged RestartCount",
				newContainerStatus("container-one", withRestartCount(2), withRunningState()),
				newContainerStatus("container-one", withRestartCount(2), withRunningState()),
				false,
			),
		)
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

type containerStatusOpt func(*corev1.ContainerStatus)

func newContainerStatus(name string, opts ...containerStatusOpt) corev1.ContainerStatus {
	status := corev1.ContainerStatus{Name: name}
	for _, opt := range opts {
		opt(&status)
	}
	return status
}

func withRestartCount(count int32) containerStatusOpt {
	return func(s *corev1.ContainerStatus) {
		s.RestartCount = count
	}
}

func withRunningState() containerStatusOpt {
	return func(s *corev1.ContainerStatus) {
		s.State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	}
}

func withWaitingState(reason string) containerStatusOpt {
	return func(s *corev1.ContainerStatus) {
		s.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}
	}
}

func withTerminatedState(exitCode int32) containerStatusOpt {
	return func(s *corev1.ContainerStatus) {
		s.State = corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: exitCode}}
	}
}

func withLastTerminatedState(exitCode int32) containerStatusOpt {
	return func(s *corev1.ContainerStatus) {
		s.LastTerminationState = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{ExitCode: exitCode},
		}
	}
}

type podOpt func(*corev1.Pod)

func withDeletionTimestamp() podOpt {
	return func(p *corev1.Pod) {
		now := metav1.Now()
		p.DeletionTimestamp = &now
	}
}

func withPodPhase(phase corev1.PodPhase) podOpt {
	return func(p *corev1.Pod) {
		p.Status.Phase = phase
	}
}

func withRestartPolicy(policy corev1.RestartPolicy) podOpt {
	return func(p *corev1.Pod) {
		p.Spec.RestartPolicy = policy
	}
}
