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

package resource

import (
	"context"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ContainerMatcher interface {
	Matches(ctx context.Context, container *corev1.Container) bool
}

type FixedNameContainerMatcher struct {
	Name string
}

func (m FixedNameContainerMatcher) Matches(ctx context.Context, container *corev1.Container) bool {
	return container.Name == m.Name
}

type RegexNameContainerMatcher struct {
	Expr string
}

func (m RegexNameContainerMatcher) Matches(ctx context.Context, container *corev1.Container) bool {
	log := ctrl.LoggerFrom(ctx).WithName("regex-container-matcher").
		WithValues("expr", m.Expr).
		WithValues("name", container.Name)
	r, err := regexp.Compile(m.Expr)
	if err != nil {
		log.Error(err, "failed to compile regular expression")
		return false
	}
	return r.MatchString(container.Name)
}
