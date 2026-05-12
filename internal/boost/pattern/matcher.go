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
	"fmt"
	"regexp"
	"strings"
)

// Matcher matches container names against patterns (glob, regex, or exact)
type Matcher struct {
	pattern string
	regex   *regexp.Regexp
	isRegex bool
	isGlob  bool
	isExact bool
}

// NewMatcher creates a new pattern matcher from a container name pattern
func NewMatcher(pattern string) (*Matcher, error) {
	m := &Matcher{
		pattern: pattern,
	}

	// Check if pattern is a regex (starts with ^ and/or ends with $)
	if strings.HasPrefix(pattern, "^") || strings.HasSuffix(pattern, "$") {
		m.isRegex = true
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
		m.regex = regex
		return m, nil
	}

	// Check if pattern contains glob wildcards
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") || strings.Contains(pattern, "[") {
		m.isGlob = true
		return m, nil
	}

	// Exact match
	m.isExact = true
	return m, nil
}

// Matches returns true if the container name matches the pattern
func (m *Matcher) Matches(containerName string) bool {
	if m.isRegex {
		return m.regex.MatchString(containerName)
	}

	if m.isGlob {
		return matchGlob(m.pattern, containerName)
	}

	// Exact match
	return m.pattern == containerName
}

// matchGlob implements simple glob pattern matching
// Supports: *, ?, [abc], [a-z], [!abc]
func matchGlob(pattern, name string) bool {
	return globMatch(pattern, name, 0, 0)
}

func globMatch(pattern, name string, pi, ni int) bool {
	for pi < len(pattern) {
		switch pattern[pi] {
		case '*':
			// Handle multiple consecutive asterisks
			for pi < len(pattern) && pattern[pi] == '*' {
				pi++
			}

			// If asterisk is at the end, it matches everything
			if pi == len(pattern) {
				return true
			}

			// Try to match the rest of the pattern with all suffixes of name
			for ni <= len(name) {
				if globMatch(pattern, name, pi, ni) {
					return true
				}
				ni++
			}
			return false

		case '?':
			// ? matches any single character except empty
			if ni >= len(name) {
				return false
			}
			pi++
			ni++

		case '[':
			// Character class [abc], [a-z], [!abc]
			if ni >= len(name) {
				return false
			}

			nextClose := strings.IndexByte(pattern[pi:], ']')
			if nextClose == -1 {
				return false // Invalid pattern: unclosed [
			}

			charClass := pattern[pi+1 : pi+nextClose]
			negate := false
			if len(charClass) > 0 && charClass[0] == '!' {
				negate = true
				charClass = charClass[1:]
			}

			matched := matchCharClass(charClass, name[ni])
			if negate {
				matched = !matched
			}
			if !matched {
				return false
			}

			pi += nextClose + 1
			ni++

		case '\\':
			// Escape next character
			if pi+1 >= len(pattern) {
				return false
			}
			pi++
			if ni >= len(name) || pattern[pi] != name[ni] {
				return false
			}
			pi++
			ni++

		default:
			// Regular character must match exactly
			if ni >= len(name) || pattern[pi] != name[ni] {
				return false
			}
			pi++
			ni++
		}
	}

	return ni == len(name)
}

func matchCharClass(charClass string, char byte) bool {
	i := 0
	for i < len(charClass) {
		// Check for range (e.g., a-z)
		if i+2 < len(charClass) && charClass[i+1] == '-' {
			if char >= charClass[i] && char <= charClass[i+2] {
				return true
			}
			i += 3
			continue
		}

		if char == charClass[i] {
			return true
		}
		i++
	}
	return false
}

func (m *Matcher) IsExact() bool {
	return m.isExact
}

func (m *Matcher) IsRegex() bool {
	return m.isRegex
}

func (m *Matcher) IsGlob() bool {
	return m.isGlob
}

func (m *Matcher) Pattern() string {
	return m.pattern
}
