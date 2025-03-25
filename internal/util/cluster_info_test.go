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

package util_test

import (
	"context"
	"errors"

	"github.com/google/kube-startup-cpu-boost/internal/mock"
	"github.com/google/kube-startup-cpu-boost/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

var _ = Describe("Cluster info", func() {
	var (
		clusterInfo util.ClusterInfo
	)
	Describe("retrieves cluster version)", func() {
		var (
			fakeClientSet kubernetes.Interface
			versionInfo   *version.Info
			err           error
		)
		JustBeforeEach(func() {
			clusterInfo = util.NewClusterInfo(context.TODO(), fakeClientSet, nil)
			versionInfo, err = clusterInfo.GetClusterVersion()
		})
		When("discovery client errors", func() {
			BeforeEach(func() {
				fakeClientSet = fakeclientset.NewSimpleClientset()
				fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery).
					PrependReactor("*", "*", func(action kubetesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("test error")
					})
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("discovery client returns server version", func() {
			BeforeEach(func() {
				fakeClientSet = fakeclientset.NewSimpleClientset()
				fakeClientSet.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion =
					&version.Info{
						GitVersion: "1.1.1",
					}
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns valid version info", func() {
				Expect(versionInfo.GitVersion).To(Equal("1.1.1"))
			})
		})
	})
	Describe("retrieves feature gates", func() {
		var (
			mockCtrl        *gomock.Controller
			mockFgValidator *mock.MockFeatureGateValidator
			featureGates    util.FeatureGates
			err             error
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockFgValidator = mock.NewMockFeatureGateValidator(mockCtrl)
		})
		JustBeforeEach(func() {
			clusterInfo = util.NewClusterInfo(context.TODO(), nil, mockFgValidator)
			featureGates, err = clusterInfo.GetFeatureGates()
		})
		When("feature gate validator errors", func() {
			BeforeEach(func() {
				mockFgValidator.EXPECT().GetFeatureGates().Return(nil, errors.New("test error"))
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("feature gate validator returns feature gates", func() {
			var (
				expectedFeatureGates util.FeatureGates
			)
			BeforeEach(func() {
				expectedFeatureGates = util.FeatureGates{
					"testFeature": {"testStage": true}}
				mockFgValidator.EXPECT().GetFeatureGates().Return(expectedFeatureGates, nil)
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("returns valid feature gates", func() {
				Expect(featureGates).To(Equal(expectedFeatureGates))
			})
		})
	})
})
