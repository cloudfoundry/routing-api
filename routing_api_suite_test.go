package routing_api_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRoutingApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RoutingApi Suite")
}
