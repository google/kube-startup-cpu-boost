package resource

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAutoPolicy_getPrediction(t *testing.T) {
	// Define a custom type for the context key
	type contextKey string

	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cpu", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var pod corev1.Pod
		err := json.NewDecoder(r.Body).Decode(&pod)
		assert.NoError(t, err)

		prediction := ResourcePrediction{
			CPURequests: "500m",
			CPULimits:   "1000m",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prediction)
	}))
	defer mockServer.Close()

	// Create an instance of AutoPolicy with the mock server URL
	policy := NewAutoPolicy(mockServer.URL)

	// Create a sample pod
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	// Create a context with the pod information
	ctx := context.WithValue(context.Background(), contextKey("pod"), pod)

	// Call the getPrediction method on the policy instance
	prediction, err := policy.(*AutoPolicy).getPrediction(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "500m", prediction.CPURequests)
	assert.Equal(t, "1000m", prediction.CPULimits)
}

func TestAutoPolicy_getPrediction_MissingPod(t *testing.T) {
	// Create an instance of AutoPolicy with a dummy URL
	policy := NewAutoPolicy("http://dummy-url")

	// Create a context without the pod information
	ctx := context.Background()

	// Call the getPrediction method on the policy instance
	_, err := policy.(*AutoPolicy).getPrediction(ctx)
	assert.Error(t, err)
	assert.Equal(t, errors.New("pod information is missing in context"), err)
}

func TestAutoPolicy_getPrediction_InvalidJSON(t *testing.T) {
	// Define a custom type for the context key
	type contextKey string

	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer mockServer.Close()

	// Create an instance of AutoPolicy with the mock server URL
	policy := NewAutoPolicy(mockServer.URL)

	// Create a sample pod
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
				},
			},
		},
	}

	// Create a context with the pod information
	ctx := context.WithValue(context.Background(), contextKey("pod"), pod)

	// Call the getPrediction method on the policy instance
	_, err := policy.(*AutoPolicy).getPrediction(ctx)
	assert.Error(t, err)
}
