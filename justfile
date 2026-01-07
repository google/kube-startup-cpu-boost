# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Variables
localbin := env_var_or_default("PWD", ".") + "/bin"
envtest_k8s_version := "1.30.x"
coverprofile := "cover.out"
coverage_dir := "coverage"

# Default recipe
default:
    @just --list

# Run all tests with coverage
test:
    @just coverage

coverage:
    #!/usr/bin/env bash
    set -e
    LOCALBIN="$(pwd)/bin"
    mkdir -p {{coverage_dir}}
    if [ ! -f "$LOCALBIN/setup-envtest" ]; then
        echo "Installing envtest..."
        GOBIN="$LOCALBIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    export KUBEBUILDER_ASSETS="$($LOCALBIN/setup-envtest use {{envtest_k8s_version}} --bin-dir "$LOCALBIN" -p path)"
    go test ./... -coverprofile={{coverage_dir}}/{{coverprofile}} -covermode=atomic
    echo "Coverage profile written to {{coverage_dir}}/{{coverprofile}}"

# Run tests without coverage (faster)
test-fast:
    #!/usr/bin/env bash
    set -e
    LOCALBIN="$(pwd)/bin"
    if [ ! -f "$LOCALBIN/setup-envtest" ]; then
        echo "Installing envtest..."
        GOBIN="$LOCALBIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    export KUBEBUILDER_ASSETS="$($LOCALBIN/setup-envtest use {{envtest_k8s_version}} --bin-dir "$LOCALBIN" -p path)"
    go test ./... -short

# Run tests for a specific package
test-package package:
    #!/usr/bin/env bash
    set -e
    LOCALBIN="$(pwd)/bin"
    if [ ! -f "$LOCALBIN/setup-envtest" ]; then
        echo "Installing envtest..."
        GOBIN="$LOCALBIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    export KUBEBUILDER_ASSETS="$($LOCALBIN/setup-envtest use {{envtest_k8s_version}} --bin-dir "$LOCALBIN" -p path)"
    go test ./{{package}} -v

# Run tests with verbose output
test-verbose:
    #!/usr/bin/env bash
    set -e
    LOCALBIN="$(pwd)/bin"
    if [ ! -f "$LOCALBIN/setup-envtest" ]; then
        echo "Installing envtest..."
        GOBIN="$LOCALBIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    export KUBEBUILDER_ASSETS="$($LOCALBIN/setup-envtest use {{envtest_k8s_version}} --bin-dir "$LOCALBIN" -p path)"
    go test ./... -v -coverprofile={{coverage_dir}}/{{coverprofile}} -covermode=atomic

# Generate HTML coverage report
coverage-html:
    #!/usr/bin/env bash
    set -e
    if [ ! -f {{coverage_dir}}/{{coverprofile}} ]; then
        echo "Coverage profile not found. Running tests first..."
        just coverage
    fi
    mkdir -p {{coverage_dir}}
    go tool cover -html={{coverage_dir}}/{{coverprofile}} -o {{coverage_dir}}/coverage.html
    echo "HTML coverage report generated: {{coverage_dir}}/coverage.html"
    echo "Open with: open {{coverage_dir}}/coverage.html"

# Generate text coverage report
coverage-text:
    #!/usr/bin/env bash
    set -e
    if [ ! -f {{coverage_dir}}/{{coverprofile}} ]; then
        echo "Coverage profile not found. Running tests first..."
        just coverage
    fi
    mkdir -p {{coverage_dir}}
    go tool cover -func={{coverage_dir}}/{{coverprofile}} > {{coverage_dir}}/coverage.txt
    cat {{coverage_dir}}/coverage.txt
    echo ""
    echo "Text coverage report saved to: {{coverage_dir}}/coverage.txt"

# Generate coverage report for a specific package
coverage-package package:
    #!/usr/bin/env bash
    set -e
    LOCALBIN="$(pwd)/bin"
    PACKAGE_NAME=$(echo {{package}} | tr '/' '_')
    mkdir -p {{coverage_dir}}
    if [ ! -f "$LOCALBIN/setup-envtest" ]; then
        echo "Installing envtest..."
        GOBIN="$LOCALBIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    export KUBEBUILDER_ASSETS="$($LOCALBIN/setup-envtest use {{envtest_k8s_version}} --bin-dir "$LOCALBIN" -p path)"
    go test ./{{package}} -coverprofile={{coverage_dir}}/${PACKAGE_NAME}_cover.out -covermode=atomic
    go tool cover -html={{coverage_dir}}/${PACKAGE_NAME}_cover.out -o {{coverage_dir}}/${PACKAGE_NAME}_coverage.html
    go tool cover -func={{coverage_dir}}/${PACKAGE_NAME}_cover.out
    echo "Coverage report for {{package}} generated: {{coverage_dir}}/${PACKAGE_NAME}_coverage.html"

# Show coverage summary (percentage only)
coverage-summary:
    #!/usr/bin/env bash
    set -e
    if [ ! -f {{coverage_dir}}/{{coverprofile}} ]; then
        echo "Coverage profile not found. Running tests first..."
        just coverage
    fi
    go tool cover -func={{coverage_dir}}/{{coverprofile}} | tail -1

# Generate all coverage reports (HTML and text)
coverage-all:
    @just coverage-html
    @just coverage-text
    @just coverage-summary

# Clean coverage artifacts
coverage-clean:
    rm -rf {{coverage_dir}}

# Run tests and generate all coverage reports
test-coverage-all:
    @just test
    @just coverage-all

