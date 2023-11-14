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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/google/kube-startup-cpu-boost/internal/boost (interfaces: TimeTicker)
//
// Generated by this command:
//
//	mockgen -package mock --copyright_file hack/boilerplate.go.txt --destination internal/mock/timeticker.go github.com/google/kube-startup-cpu-boost/internal/boost TimeTicker
//

package mock

import (
	reflect "reflect"
	time "time"

	gomock "go.uber.org/mock/gomock"
)

// MockTimeTicker is a mock of TimeTicker interface.
type MockTimeTicker struct {
	ctrl     *gomock.Controller
	recorder *MockTimeTickerMockRecorder
}

// MockTimeTickerMockRecorder is the mock recorder for MockTimeTicker.
type MockTimeTickerMockRecorder struct {
	mock *MockTimeTicker
}

// NewMockTimeTicker creates a new mock instance.
func NewMockTimeTicker(ctrl *gomock.Controller) *MockTimeTicker {
	mock := &MockTimeTicker{ctrl: ctrl}
	mock.recorder = &MockTimeTickerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTimeTicker) EXPECT() *MockTimeTickerMockRecorder {
	return m.recorder
}

// Stop mocks base method.
func (m *MockTimeTicker) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockTimeTickerMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockTimeTicker)(nil).Stop))
}

// Tick mocks base method.
func (m *MockTimeTicker) Tick() <-chan time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tick")
	ret0, _ := ret[0].(<-chan time.Time)
	return ret0
}

// Tick indicates an expected call of Tick.
func (mr *MockTimeTickerMockRecorder) Tick() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tick", reflect.TypeOf((*MockTimeTicker)(nil).Tick))
}
