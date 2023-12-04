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

package duration

import (
	"time"

	v1 "k8s.io/api/core/v1"
)

const (
	FixedDurationPolicyName = "FixedDuration"
)

type TimeFunc func() time.Time

type FixedDurationPolicy struct {
	timeFunc TimeFunc
	duration time.Duration
}

func NewFixedDurationPolicy(duration time.Duration) Policy {
	return NewFixedDurationPolicyWithTimeFunc(time.Now, duration)
}

func NewFixedDurationPolicyWithTimeFunc(timeFunc TimeFunc, duration time.Duration) Policy {
	return &FixedDurationPolicy{
		timeFunc: timeFunc,
		duration: duration,
	}
}

func (*FixedDurationPolicy) Name() string {
	return FixedDurationPolicyName
}

func (p *FixedDurationPolicy) Duration() time.Duration {
	return p.duration
}

func (p *FixedDurationPolicy) Valid(pod *v1.Pod) bool {
	now := p.timeFunc()
	return pod.CreationTimestamp.Add(p.duration).After(now)
}
