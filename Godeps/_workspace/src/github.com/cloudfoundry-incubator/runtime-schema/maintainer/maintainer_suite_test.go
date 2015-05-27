package maintainer_test

import (
	"github.com/cloudfoundry-incubator/consuladapter"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"

	"testing"
)

var (
	consulStartingPort int
	consulRunner       *consuladapter.ClusterRunner
)

const (
	defaultScheme = "http"
)

func TestMaintainer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Maintainer Suite")
}

var _ = BeforeSuite(func() {
	consulStartingPort = 5001 + config.GinkgoConfig.ParallelNode*consuladapter.PortOffsetLength
	consulRunner = consuladapter.NewClusterRunner(consulStartingPort, 1, defaultScheme)

	consulRunner.Start()
	consulRunner.WaitUntilReady()
})

var _ = BeforeEach(func() {
	consulRunner.Reset()
})

var _ = AfterSuite(func() {
	consulRunner.Stop()
})
