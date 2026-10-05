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

// Package mock is a generated GoMock package.
//
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination boost_manager.go github.com/google/kube-startup-cpu-boost/internal/boost Manager
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination ctrl_manager.go -mock_names Manager=MockCtrlManager sigs.k8s.io/controller-runtime Manager
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination featuregatevalidator.go github.com/google/kube-startup-cpu-boost/internal/util FeatureGateValidator
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination k8s_client.go sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination k8s_client_rest_interface.go k8s.io/client-go/rest Interface
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination k8s_subresourceclient.go sigs.k8s.io/controller-runtime/pkg/client SubResourceClient
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination reconciler.go sigs.k8s.io/controller-runtime/pkg/reconcile Reconciler
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination startupcpuboost.go github.com/google/kube-startup-cpu-boost/internal/boost StartupCPUBoost
//go:generate mockgen -package mock --copyright_file ../../hack/boilerplate.go.txt --destination timeticker.go github.com/google/kube-startup-cpu-boost/internal/boost TimeTicker
package mock
