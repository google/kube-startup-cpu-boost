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

package pattern

import (
	"testing"
)

func TestExactMatching(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		containerNames []string
		shouldMatch    []bool
	}{
		{
			name:           "exact single match",
			pattern:        "app",
			containerNames: []string{"app", "application", "app-v1", "my-app"},
			shouldMatch:    []bool{true, false, false, false},
		},
		{
			name:           "exact with hyphens",
			pattern:        "web-server",
			containerNames: []string{"web-server", "web-server-v1", "web-serve", "web-servers"},
			shouldMatch:    []bool{true, false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("NewMatcher failed: %v", err)
			}

			for i, containerName := range tt.containerNames {
				got := m.Matches(containerName)
				want := tt.shouldMatch[i]
				if got != want {
					t.Errorf("pattern %q container %q: got %v, want %v", tt.pattern, containerName, got, want)
				}
			}
		})
	}
}

func TestGlobPatterns(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		containerNames []string
		shouldMatch    []bool
	}{
		{
			name:           "wildcard prefix",
			pattern:        "web-*",
			containerNames: []string{"web-app", "web-server", "web", "myapp-web", "web-"},
			shouldMatch:    []bool{true, true, false, false, true},
		},
		{
			name:           "wildcard suffix",
			pattern:        "*-sidecar",
			containerNames: []string{"istio-sidecar", "envoy-sidecar", "sidecar", "my-sidecar-app", "sidecar-app"},
			shouldMatch:    []bool{true, true, false, false, false},
		},
		{
			name:           "wildcard both sides",
			pattern:        "*app*",
			containerNames: []string{"app", "myapp", "app-sidecar", "sidecar-app", "application", "myapplicationx"},
			shouldMatch:    []bool{true, true, true, true, true, true},
		},
		{
			name:           "multiple wildcards",
			pattern:        "app-*-v*",
			containerNames: []string{"app-main-v1", "app-db-v2", "app-v1", "app-cache-v1-test", "app-service"},
			shouldMatch:    []bool{true, true, false, true, false},
		},
		{
			name:           "question mark single char",
			pattern:        "app-v?",
			containerNames: []string{"app-v1", "app-v2", "app-va", "app-v", "app-v12"},
			shouldMatch:    []bool{true, true, true, false, false},
		},
		{
			name:           "character class",
			pattern:        "app-v[0-9]",
			containerNames: []string{"app-v0", "app-v5", "app-va", "app-v", "app-v10"},
			shouldMatch:    []bool{true, true, false, false, false},
		},
		{
			name:           "character class with negation",
			pattern:        "app-v[!a-z]",
			containerNames: []string{"app-v1", "app-va", "app-v-", "app-v"},
			shouldMatch:    []bool{true, false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("NewMatcher failed: %v", err)
			}

			if !m.IsGlob() {
				t.Errorf("Expected glob pattern for %q", tt.pattern)
			}

			for i, containerName := range tt.containerNames {
				got := m.Matches(containerName)
				want := tt.shouldMatch[i]
				if got != want {
					t.Errorf("pattern %q container %q: got %v, want %v", tt.pattern, containerName, got, want)
				}
			}
		})
	}
}

func TestRegexPatterns(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		containerNames []string
		shouldMatch    []bool
	}{
		{
			name:           "simple prefix regex",
			pattern:        "^web-.*$",
			containerNames: []string{"web-app", "web-server", "web", "myapp-web", "web-"},
			shouldMatch:    []bool{true, true, false, false, true},
		},
		{
			name:           "alternation regex",
			pattern:        "^(app|api).*$",
			containerNames: []string{"app", "api", "application", "api-server", "myapp", "my-api"},
			shouldMatch:    []bool{true, true, true, true, false, false},
		},
		{
			name:           "version pattern regex",
			pattern:        "^.*-v[0-9]+$",
			containerNames: []string{"app-v1", "service-v2", "v1", "app-v", "app-v1beta"},
			shouldMatch:    []bool{true, true, false, false, false},
		},
		{
			name:           "complex pattern",
			pattern:        "^(app|svc)-[a-z]+-v\\d+$",
			containerNames: []string{"app-db-v1", "svc-cache-v2", "app-v1", "app-db-1", "app-DB-v1"},
			shouldMatch:    []bool{true, true, false, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("NewMatcher failed: %v", err)
			}

			if !m.IsRegex() {
				t.Errorf("Expected regex pattern for %q", tt.pattern)
			}

			for i, containerName := range tt.containerNames {
				got := m.Matches(containerName)
				want := tt.shouldMatch[i]
				if got != want {
					t.Errorf("pattern %q container %q: got %v, want %v", tt.pattern, containerName, got, want)
				}
			}
		})
	}
}

func TestWildcardSpecial(t *testing.T) {
	tests := []struct {
		name           string
		pattern        string
		containerNames []string
		shouldMatch    []bool
	}{
		{
			name:           "single asterisk matches all",
			pattern:        "*",
			containerNames: []string{"app", "web-server", "", "service-v1", "x"},
			shouldMatch:    []bool{true, true, true, true, true},
		},
		{
			name:           "double asterisk",
			pattern:        "**",
			containerNames: []string{"app", "web-server", "", "any-thing"},
			shouldMatch:    []bool{true, true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("NewMatcher failed: %v", err)
			}

			for i, containerName := range tt.containerNames {
				got := m.Matches(containerName)
				want := tt.shouldMatch[i]
				if got != want {
					t.Errorf("pattern %q container %q: got %v, want %v", tt.pattern, containerName, got, want)
				}
			}
		})
	}
}

func TestInvalidPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "valid regex",
			pattern: "^[a-z]+$",
			wantErr: false,
		},
		{
			name:    "invalid regex unclosed bracket",
			pattern: "^[a-z$",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMatcher(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMatcher(%q): got err %v, wantErr %v", tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestMatcherTypes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		isExact bool
		isGlob  bool
		isRegex bool
	}{
		{
			name:    "exact match",
			pattern: "app",
			isExact: true,
			isGlob:  false,
			isRegex: false,
		},
		{
			name:    "glob pattern",
			pattern: "app-*",
			isExact: false,
			isGlob:  true,
			isRegex: false,
		},
		{
			name:    "regex pattern",
			pattern: "^app-.*$",
			isExact: false,
			isGlob:  false,
			isRegex: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.pattern)
			if err != nil {
				t.Fatalf("NewMatcher failed: %v", err)
			}

			if m.IsExact() != tt.isExact {
				t.Errorf("IsExact: got %v, want %v", m.IsExact(), tt.isExact)
			}
			if m.IsGlob() != tt.isGlob {
				t.Errorf("IsGlob: got %v, want %v", m.IsGlob(), tt.isGlob)
			}
			if m.IsRegex() != tt.isRegex {
				t.Errorf("IsRegex: got %v, want %v", m.IsRegex(), tt.isRegex)
			}
		})
	}
}
