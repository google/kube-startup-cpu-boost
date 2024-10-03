package resource_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	apiResource "k8s.io/apimachinery/pkg/api/resource"

	resource "github.com/google/kube-startup-cpu-boost/internal/boost/resource"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Auto Resource Policy", func() {
	var (
		policy       resource.ContainerPolicy
		newResources *corev1.ResourceRequirements
		container    *corev1.Container
		mockServer   *httptest.Server
		oldStdout    *os.File
		stdoutReader *os.File
		stdoutWriter *os.File
		outputBuffer *bytes.Buffer
	)

	BeforeEach(func() {
		container = containerTemplate.DeepCopy()

		// Set up pipe to capture stdout
		oldStdout = os.Stdout
		stdoutReader, stdoutWriter, _ = os.Pipe()
		os.Stdout = stdoutWriter

		outputBuffer = new(bytes.Buffer)
	})

	JustBeforeEach(func() {
		// Run the test case logic that triggers the fmt.Print statements.
		policy = resource.NewAutoPolicy(mockServer.URL)

		podName := "test-pod"
		podNamespace := "test-namespace"

		ctx := context.WithValue(context.TODO(), resource.ContextKey("podName"), podName)
		ctx = context.WithValue(ctx, resource.ContextKey("podNamespace"), podNamespace)

		newResources = policy.NewResources(ctx, container)

		fmt.Printf("newResources: %+v\n", newResources)

		// Ensure everything written to os.Stdout is captured
		stdoutWriter.Close()                       // Close the writer
		_, _ = io.Copy(outputBuffer, stdoutReader) // Copy the output to buffer
		os.Stdout = oldStdout                      // Restore stdout
	})

	AfterEach(func() {
		if mockServer != nil {
			mockServer.Close()
		}
		fmt.Fprintln(GinkgoWriter, "Captured Output:", outputBuffer.String()) // Print captured output to test log
	})

	Describe("AutoPolicy", func() {
		Context("when the API returns valid predictions", func() {
			BeforeEach(func() {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/cpu"))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					var pod corev1.Pod
					err := json.NewDecoder(r.Body).Decode(&pod)
					Expect(err).NotTo(HaveOccurred())

					prediction := resource.ResourcePrediction{
						CPURequests: "500m",
						CPULimits:   "1000m",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(prediction)
				}))

				cpuRequests, err := apiResource.ParseQuantity("300m")
				Expect(err).NotTo(HaveOccurred())
				cpuLimits, err := apiResource.ParseQuantity("500m")
				Expect(err).NotTo(HaveOccurred())

				container.Resources.Requests[corev1.ResourceCPU] = cpuRequests
				container.Resources.Limits[corev1.ResourceCPU] = cpuLimits
			})

			It("returns resources with valid CPU requests and limits", func() {
				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Requests).NotTo(BeNil())
				Expect(newResources.Requests).To(HaveKey(corev1.ResourceCPU))
				cpuRequest := newResources.Requests[corev1.ResourceCPU]
				// fmt.Fprintln(GinkgoWriter, "cpuRequest:", cpuRequest.String())
				Expect(cpuRequest.String()).To(Equal("500m"))

				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Requests).NotTo(BeNil())
				Expect(newResources.Limits).To(HaveKey(corev1.ResourceCPU))
				cpuLimit := newResources.Limits[corev1.ResourceCPU]
				// fmt.Fprintln(GinkgoWriter, "cpuLimit:", cpuLimit.String())
				Expect(cpuLimit.String()).To(Equal("1"))
			})
		})

		Context("when the API returns 400m requests and 600m limits", func() {
			BeforeEach(func() {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/cpu"))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					var pod corev1.Pod
					err := json.NewDecoder(r.Body).Decode(&pod)
					Expect(err).NotTo(HaveOccurred())

					prediction := resource.ResourcePrediction{
						CPURequests: "400m",
						CPULimits:   "600m",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(prediction)
				}))

				cpuRequests, err := apiResource.ParseQuantity("300m")
				Expect(err).NotTo(HaveOccurred())
				cpuLimits, err := apiResource.ParseQuantity("400m")
				Expect(err).NotTo(HaveOccurred())

				container.Resources.Requests[corev1.ResourceCPU] = cpuRequests
				container.Resources.Limits[corev1.ResourceCPU] = cpuLimits
			})

			It("returns resources with 400m CPU requests and 600m limits", func() {
				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Requests).NotTo(BeNil())
				Expect(newResources.Requests).To(HaveKey(corev1.ResourceCPU))
				cpuRequest := newResources.Requests[corev1.ResourceCPU]
				fmt.Fprintln(GinkgoWriter, "cpuRequest:", cpuRequest.String())
				Expect(cpuRequest.String()).To(Equal("400m"))

				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Limits).NotTo(BeNil())
				Expect(newResources.Limits).To(HaveKey(corev1.ResourceCPU))
				cpuLimit := newResources.Limits[corev1.ResourceCPU]
				fmt.Fprintln(GinkgoWriter, "cpuLimit:", cpuLimit.String())
				Expect(cpuLimit.String()).To(Equal("600m"))
			})
		})

		Context("when the API returns 600m requests and 800m limits", func() {
			BeforeEach(func() {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/cpu"))
					Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))

					var pod corev1.Pod
					err := json.NewDecoder(r.Body).Decode(&pod)
					Expect(err).NotTo(HaveOccurred())

					prediction := resource.ResourcePrediction{
						CPURequests: "600m",
						CPULimits:   "800m",
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(prediction)
				}))

				cpuRequests, err := apiResource.ParseQuantity("500m")
				Expect(err).NotTo(HaveOccurred())
				cpuLimits, err := apiResource.ParseQuantity("600m")
				Expect(err).NotTo(HaveOccurred())

				container.Resources.Requests[corev1.ResourceCPU] = cpuRequests
				container.Resources.Limits[corev1.ResourceCPU] = cpuLimits
			})

			It("returns resources with 600m CPU requests and 800m limits", func() {
				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Requests).NotTo(BeNil())
				Expect(newResources.Requests).To(HaveKey(corev1.ResourceCPU))
				cpuRequest := newResources.Requests[corev1.ResourceCPU]
				fmt.Fprintln(GinkgoWriter, "cpuRequest:", cpuRequest.String())
				Expect(cpuRequest.String()).To(Equal("600m"))

				Expect(newResources).NotTo(BeNil())
				Expect(newResources.Requests).NotTo(BeNil())
				Expect(newResources.Limits).To(HaveKey(corev1.ResourceCPU))
				cpuLimit := newResources.Limits[corev1.ResourceCPU]
				fmt.Fprintln(GinkgoWriter, "cpuLimit:", cpuLimit.String())
				Expect(cpuLimit.String()).To(Equal("800m"))
			})
		})
	})
})
