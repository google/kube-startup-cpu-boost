package duration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestAutoDurationPolicy_GetDuration(t *testing.T) {
	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/duration", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var pod corev1.Pod
		err := json.NewDecoder(r.Body).Decode(&pod)
		assert.NoError(t, err)

		prediction := DurationPrediction{
			Duration: "5m",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prediction)
	}))
	defer mockServer.Close()

	// Create an instance of AutoDurationPolicy with the mock server URL
	policy := NewAutoDurationPolicy(mockServer.URL)

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

	// Call the GetDuration method
	duration, err := policy.GetDuration(pod)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Minute, duration)
}

func TestAutoDurationPolicy_getPrediction(t *testing.T) {
	// Mock API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/duration", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var pod corev1.Pod
		err := json.NewDecoder(r.Body).Decode(&pod)
		assert.NoError(t, err)

		prediction := DurationPrediction{
			Duration: "5m",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(prediction)
	}))
	defer mockServer.Close()

	// Create an instance of AutoDurationPolicy with the mock server URL
	policy := NewAutoDurationPolicy(mockServer.URL)

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

	// Call the getPrediction method
	prediction, err := policy.getPrediction(pod)
	assert.NoError(t, err)
	assert.Equal(t, "5m", prediction.Duration)
}
