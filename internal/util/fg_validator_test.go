// Copyright 2024 Google LLC
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/kube-startup-cpu-boost/internal/mock"
	"github.com/google/kube-startup-cpu-boost/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	fakerest "k8s.io/client-go/rest/fake"
)

const (
	K8S_FG_METRIC_HELP = "# HELP kubernetes_feature_enabled [BETA] This metric records the data about the stage and enablement of a k8s feature."
	K8S_FG_METRIC_TYPE = "# TYPE kubernetes_feature_enabled gauge"
)

var _ = Describe("Feature Gate Validator", func() {
	Describe("Retrieves feature gates from metrics", func() {
		var (
			mockCtrl           *gomock.Controller
			mockClient         *mock.MockInterface
			mockCall           *gomock.Call
			respData           map[string]map[string]int
			metricsFgValidator util.FeatureGateValidator
			featureGates       util.FeatureGates
			err                error
		)
		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockClient = mock.NewMockInterface(mockCtrl)
			metricsFgValidator = util.NewMetricsFeatureGateValidator(context.TODO(), mockClient)
		})
		JustBeforeEach(func() {
			fakeClient := fakerest.RESTClient{
				Resp: &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(
						bytes.NewReader([]byte(featureGateString(respData))),
					),
				},
			}
			mockCall = mockClient.EXPECT().Get().Return(fakeClient.Request())
			featureGates, err = metricsFgValidator.GetFeatureGates()
		})
		When("REST client returns error", func() {
			JustBeforeEach(func() {
				fakeClient := fakerest.RESTClient{
					Err: errors.New("fake error"),
				}
				mockCall = mockClient.EXPECT().Get().Return(fakeClient.Request())
				featureGates, err = metricsFgValidator.GetFeatureGates()
			})
			It("errors", func() {
				Expect(err).To(HaveOccurred())
			})
		})
		When("The kubernetes_feature_enabled metric is missing", func() {
			BeforeEach(func() {
				respData = make(map[string]map[string]int)
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("calls GET on metrics endpoint", func() {
				mockCall.Times(1)
			})
			It("returns zero feature gates", func() {
				Expect(len(featureGates)).To(Equal(0))
			})
		})
		When("The kubernetes_feature_enabled is present with three features", func() {
			BeforeEach(func() {
				respData = map[string]map[string]int{
					"InPlacePodVerticalScaling": {
						"ALPHA": 0,
						"BETA":  1,
					},
					"MinDomainsInPodTopologySpread": {
						"": 1,
					},
					"NodeLogQuery": {
						"BETA": 0,
					},
				}
			})
			It("doesn't error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("calls GET on metrics endpoint", func() {
				mockCall.Times(1)
			})
			It("returns feature gates", func() {
				Expect(len(featureGates)).To(Equal(3))
			})
			It("returns InPlacePodVerticalScaling with two states", func() {
				Expect(len(featureGates["InPlacePodVerticalScaling"])).To(Equal(2))
			})
			It("returns InPlacePodVerticalScaling ALPHA disabled", func() {
				Expect(featureGates.IsEnabled("InPlacePodVerticalScaling", "ALPHA")).To(BeFalse())
			})
			It("returns InPlacePodVerticalScaling BETA enabled", func() {
				Expect(featureGates.IsEnabled("InPlacePodVerticalScaling", "BETA")).To(BeTrue())
			})
			It("returns InPlacePodVerticalScaling anyStage enabled", func() {
				Expect(featureGates.IsEnabledAnyStage("InPlacePodVerticalScaling")).To(BeTrue())
			})
			It("returns MinDomainsInPodTopologySpread GA enabled", func() {
				Expect(featureGates.IsEnabled("MinDomainsInPodTopologySpread", "")).To(BeTrue())
			})
			It("returns NodeLogQuery BETA disabled", func() {
				Expect(featureGates.IsEnabled("NodeLogQuery", "BETA")).To(BeFalse())
			})
			It("returns NonExisting BETA disabled", func() {
				Expect(featureGates.IsEnabled("NonExisting", "BETA")).To(BeFalse())
			})
		})
	})
})

func featureGateString(data map[string]map[string]int) string {
	builder := strings.Builder{}
	builder.WriteString(K8S_FG_METRIC_HELP)
	builder.WriteString("\n")
	builder.WriteString(K8S_FG_METRIC_TYPE)
	builder.WriteString("\n")
	if data == nil {
		return builder.String()
	}
	for fg, stageValues := range data {
		for stage, value := range stageValues {
			featureStr := fmt.Sprintf("kubernetes_feature_enabled{name=\"%s\",stage=\"%s\"} %d", fg, stage, value)
			builder.WriteString(featureStr)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
