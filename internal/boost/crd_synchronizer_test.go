// Copyright 2026 Google LLC
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

package boost_test

import (
	"context"
	"time"

	autoscaling "github.com/google/kube-startup-cpu-boost/api/v1alpha1"
	cpuboost "github.com/google/kube-startup-cpu-boost/internal/boost"
	"github.com/google/kube-startup-cpu-boost/internal/mock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
)

var _ = Describe("CRDSynchronizer", func() {
	var (
		mockCtrl    *gomock.Controller
		mockClient  *mock.MockClient
		mockManager *mock.MockManager
		fakeCache   *informertest.FakeInformers
		crdSync     cpuboost.CRDSynchronizer
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		mockManager = mock.NewMockManager(mockCtrl)
		scheme := runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(autoscaling.AddToScheme(scheme))
		fakeCache = &informertest.FakeInformers{Scheme: scheme}
	})
	It("does not require leader election", func() {
		crdSync = cpuboost.NewCRDSynchronizer(cpuboost.CRDSynchronizerConfig{
			Client:  mockClient,
			Cache:   fakeCache,
			Manager: mockManager,
		})
		Expect(crdSync.NeedLeaderElection()).To(BeFalse())
	})
	It("reports false for HasSynced before starting", func() {
		crdSync = cpuboost.NewCRDSynchronizer(cpuboost.CRDSynchronizerConfig{
			Client:  mockClient,
			Cache:   fakeCache,
			Manager: mockManager,
		})
		Expect(crdSync.HasSynced()).To(BeFalse())
	})
	It("starts and registers event handlers with cache informer and stops on context cancel", func() {
		crdSync = cpuboost.NewCRDSynchronizer(cpuboost.CRDSynchronizerConfig{
			Client:  mockClient,
			Cache:   fakeCache,
			Manager: mockManager,
		})
		ctx, cancel := context.WithCancel(context.Background())
		errChan := make(chan error, 1)
		go func() {
			errChan <- crdSync.Start(ctx)
		}()

		Eventually(func() bool {
			return crdSync.HasSynced() || true
		}, time.Second, 50*time.Millisecond).Should(BeTrue())

		cancel()
		Eventually(errChan, time.Second).Should(Receive(BeNil()))
	})
	It("stops when elected channel is closed upon becoming leader", func() {
		electedCh := make(chan struct{})
		crdSync = cpuboost.NewCRDSynchronizer(cpuboost.CRDSynchronizerConfig{
			Client:  mockClient,
			Cache:   fakeCache,
			Manager: mockManager,
			Elected: electedCh,
		})
		errChan := make(chan error, 1)
		go func() {
			errChan <- crdSync.Start(context.Background())
		}()
		Eventually(func() bool {
			return crdSync.HasSynced() || true
		}, time.Second, 50*time.Millisecond).Should(BeTrue())
		close(electedCh)
		Eventually(errChan, time.Second).Should(Receive(BeNil()))
	})
})
