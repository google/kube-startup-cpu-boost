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

package controller_test

import (
	"context"

	"github.com/go-logr/logr"
	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	"github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/controller"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("BoostController", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *mock.MockClient
		mockManager *mock.MockManager
		mockBoost   *mock.MockStartupCPUBoost
		boostCtrl   controller.StartupCPUBoostReconciler
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		mockManager = mock.NewMockManager(mockCtrl)
		mockBoost = mock.NewMockStartupCPUBoost(mockCtrl)
		boostCtrl = controller.StartupCPUBoostReconciler{
			Log:     logr.Discard(),
			Client:  mockClient,
			Manager: mockManager,
		}
	})
	Describe("Setups with manager", func() {
		var (
			mockCtrlManager *mock.MockCtrlManager
			serverVersion   *version.Info
			err             error
		)
		BeforeEach(func() {
			scheme := runtime.NewScheme()
			utilruntime.Must(clientgoscheme.AddToScheme(scheme))
			utilruntime.Must(autoscaling.AddToScheme(scheme))
			mockCtrlManager = mock.NewMockCtrlManager(mockCtrl)
			skipNameValidation := true
			mockCtrlManager.EXPECT().GetControllerOptions().
				Return(config.Controller{SkipNameValidation: &skipNameValidation}).MinTimes(1)
			mockCtrlManager.EXPECT().GetScheme().Return(scheme).MinTimes(1)
			mockCtrlManager.EXPECT().GetLogger().Return(logr.Discard()).MinTimes(1)
			mockCtrlManager.EXPECT().Add(gomock.Any()).Return(nil).MinTimes(1)
			mockCtrlManager.EXPECT().GetCache().Return(&informertest.FakeInformers{}).MinTimes(1)
		})
		JustBeforeEach(func() {
			err = boostCtrl.SetupWithManager(mockCtrlManager, serverVersion)
		})
		When("server version is newer or equal to 1.32.0", func() {
			BeforeEach(func() {
				serverVersion = &version.Info{
					GitVersion: "v1.32.0",
				}
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("runs new revert mode", func() {
				Expect(boostCtrl.LegacyRevertMode).To(BeFalse())
			})
		})
		When("server version is less than 1.32.0", func() {
			BeforeEach(func() {
				serverVersion = &version.Info{
					GitVersion: "v1.29.2",
				}
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("runs legacy revert mode", func() {
				Expect(boostCtrl.LegacyRevertMode).To(BeTrue())
			})
		})
	})
	Describe("Receives reconcile request", func() {
		var (
			req       ctrl.Request
			name      string
			namespace string
			result    ctrl.Result
			err       error
		)
		BeforeEach(func() {
			name = "boost-001"
			namespace = "demo"
			req = ctrl.Request{
				NamespacedName: types.NamespacedName{Name: name, Namespace: namespace},
			}
		})
		JustBeforeEach(func() {
			result, err = boostCtrl.Reconcile(context.TODO(), req)
		})
		When("boost is registered in boost manager", func() {
			var (
				totalContainerBoosts  = 10
				activeContainerBoosts = 5
				activeConditionTrue   = metav1.Condition{
					Type:    "Active",
					Status:  metav1.ConditionTrue,
					Reason:  controller.BoostActiveConditionTrueReason,
					Message: controller.BoostActiveConditionTrueMessage,
				}
			)
			BeforeEach(func() {
				stats := boost.StartupCPUBoostStats{
					TotalContainerBoosts:  totalContainerBoosts,
					ActiveContainerBoosts: activeContainerBoosts,
				}
				mockManager.EXPECT().StartupCPUBoost(gomock.Eq(namespace), gomock.Eq(name)).Times(1).Return(mockBoost, true)
				mockBoost.EXPECT().Stats().Times(1).Return(stats)
			})
			When("there existing status is up to date", func() {
				BeforeEach(func() {
					mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(req.NamespacedName), gomock.Any()).
						Times(1).DoAndReturn(func(c context.Context, cc client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						boostObj := obj.(*autoscaling.StartupCPUBoost)
						boostObj.Name = name
						boostObj.Namespace = namespace
						meta.SetStatusCondition(&boostObj.Status.Conditions, activeConditionTrue)
						boostObj.Status.TotalContainerBoosts = int32(totalContainerBoosts)
						boostObj.Status.ActiveContainerBoosts = int32(activeContainerBoosts)
						return nil
					})
				})
				It("does not error", func() {
					Expect(err).To(BeNil())
				})
				It("returns empty result", func() {
					Expect(result).To(Equal(ctrl.Result{}))
				})
			})
			When("there existing status is not up to date", func() {
				var mockSubResClient *mock.MockSubResourceClient
				BeforeEach(func() {
					mockSubResClient = mock.NewMockSubResourceClient(mockCtrl)
					mockSubResClient.EXPECT().Update(
						gomock.Any(),
						gomock.Cond(func(b any) bool {
							boostObj := b.(*autoscaling.StartupCPUBoost)
							ret := boostObj.Status.ActiveContainerBoosts == int32(activeContainerBoosts)
							ret = ret && boostObj.Status.TotalContainerBoosts == int32(totalContainerBoosts)
							ret = ret && boostObj.Name == name
							ret = ret && boostObj.Namespace == namespace
							return ret
						})).
						Return(nil).Times(1)
					mockClient.EXPECT().Get(gomock.Any(), gomock.Eq(req.NamespacedName), gomock.Any()).
						Times(1).DoAndReturn(func(c context.Context, cc client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						boostObj := obj.(*autoscaling.StartupCPUBoost)
						boostObj.Name = name
						boostObj.Namespace = namespace
						return nil
					})
					mockClient.EXPECT().Status().Return(mockSubResClient).Times(1)
				})
				It("does not error", func() {
					Expect(err).To(BeNil())
				})
				It("returns empty result", func() {
					Expect(result).To(Equal(ctrl.Result{}))
				})
			})
		})
	})
	Describe("receives update event", func() {
		var (
			updateEvent event.UpdateEvent
			mgrMockCall *gomock.Call
		)
		BeforeEach(func() {
			updateEvent = event.UpdateEvent{
				ObjectNew: &autoscaling.StartupCPUBoost{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "boost-001",
						Namespace: "demo",
					},
				},
			}
			mgrMockCall = mockManager.EXPECT().UpdateStartupCPUBoost(
				gomock.Any(), gomock.Eq(updateEvent.ObjectNew))
		})
		JustBeforeEach(func() {
			ok := boostCtrl.Update(updateEvent)
			Expect(ok).To(BeTrue())
		})
		It("calls manager with valid update", func() {
			mgrMockCall.Times(1)
		})
	})
})
