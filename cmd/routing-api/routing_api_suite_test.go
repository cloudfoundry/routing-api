package main_test

import (
	"fmt"
	"net/url"
	"os"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/cmd/routing-api/testrunner"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
	"time"
)

var etcdPort int
var etcdUrl string
var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdAdapter storeadapter.StoreAdapter

var consulScheme string
var consulDatacenter string
var consulRunner *consuladapter.ClusterRunner
var consulSession *consuladapter.Session

var client routing_api.Client
var routingAPIBinPath string
var routingAPIAddress string
var routingAPIArgs testrunner.Args
var routingAPIRunner *ginkgomon.Runner
var routingAPIProcess ifrit.Process
var routingAPIPort int
var routingAPIIP string
var routingAPISystemDomain string

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = SynchronizedBeforeSuite(
	func() []byte {
		routingAPIBin, err := gexec.Build("github.com/cloudfoundry-incubator/routing-api/cmd/routing-api", "-race")
		Expect(err).NotTo(HaveOccurred())
		return []byte(routingAPIBin)
	},
	func(routingAPIBin []byte) {
		routingAPIBinPath = string(routingAPIBin)
		SetDefaultEventuallyTimeout(15 * time.Second)

		consulScheme = "http"
		consulDatacenter = "dc"
		consulRunner = consuladapter.NewClusterRunner(
			9001+GinkgoParallelNode()*consuladapter.PortOffsetLength,
			1,
			consulScheme,
		)

		consulRunner.Start()
		consulRunner.WaitUntilReady()
	},
)

var _ = SynchronizedAfterSuite(func() {
	consulRunner.Stop()
}, func() {
	gexec.CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	etcdPort = 4001 + GinkgoParallelNode()
	etcdUrl = fmt.Sprintf("http://127.0.0.1:%d", etcdPort)
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(etcdPort, 1)
	etcdRunner.Start()

	etcdAdapter = etcdRunner.Adapter()
	routingAPIPort = 6900 + GinkgoParallelNode()
	routingAPIIP = "127.0.0.1"
	routingAPISystemDomain = "example.com"
	routingAPIAddress = fmt.Sprintf("%s:%d", routingAPIIP, routingAPIPort)

	routingAPIURL := &url.URL{
		Scheme: "http",
		Host:   routingAPIAddress,
	}

	client = routing_api.NewClient(routingAPIURL.String())
	workingDir, _ := os.Getwd()

	routingAPIArgs = testrunner.Args{
		Port:          routingAPIPort,
		IP:            routingAPIIP,
		SystemDomain:  routingAPISystemDomain,
		ConfigPath:    workingDir + "/../../example_config/example.yml",
		EtcdCluster:   etcdUrl,
		DevMode:       true,
		ConsulCluster: consulRunner.ConsulCluster(),
	}
	routingAPIRunner = testrunner.New(routingAPIBinPath, routingAPIArgs)
})

var _ = AfterEach(func() {
	etcdAdapter.Disconnect()
	etcdRunner.Stop()

	consulRunner.Reset()
	consulSession = consulRunner.NewSession("a-session")
})
