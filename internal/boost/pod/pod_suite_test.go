package pod_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPod(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pod Suite")
}
