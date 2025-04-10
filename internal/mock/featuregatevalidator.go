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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/google/kube-startup-cpu-boost/internal/util (interfaces: FeatureGateValidator)
//
// Generated by this command:
//
//	mockgen -package mock --copyright_file hack/boilerplate.go.txt --destination internal/mock/featuregatevalidator.go github.com/google/kube-startup-cpu-boost/internal/util FeatureGateValidator
//

package mock

import (
	reflect "reflect"

	util "github.com/google/kube-startup-cpu-boost/internal/util"
	gomock "go.uber.org/mock/gomock"
)

// MockFeatureGateValidator is a mock of FeatureGateValidator interface.
type MockFeatureGateValidator struct {
	ctrl     *gomock.Controller
	recorder *MockFeatureGateValidatorMockRecorder
	isgomock struct{}
}

// MockFeatureGateValidatorMockRecorder is the mock recorder for MockFeatureGateValidator.
type MockFeatureGateValidatorMockRecorder struct {
	mock *MockFeatureGateValidator
}

// NewMockFeatureGateValidator creates a new mock instance.
func NewMockFeatureGateValidator(ctrl *gomock.Controller) *MockFeatureGateValidator {
	mock := &MockFeatureGateValidator{ctrl: ctrl}
	mock.recorder = &MockFeatureGateValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFeatureGateValidator) EXPECT() *MockFeatureGateValidatorMockRecorder {
	return m.recorder
}

// GetFeatureGates mocks base method.
func (m *MockFeatureGateValidator) GetFeatureGates() (util.FeatureGates, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFeatureGates")
	ret0, _ := ret[0].(util.FeatureGates)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFeatureGates indicates an expected call of GetFeatureGates.
func (mr *MockFeatureGateValidatorMockRecorder) GetFeatureGates() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFeatureGates", reflect.TypeOf((*MockFeatureGateValidator)(nil).GetFeatureGates))
}
