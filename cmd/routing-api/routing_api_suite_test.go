package main_test

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	tlsHelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	loggingclient "code.cloudfoundry.org/diego-logging-client"

	"code.cloudfoundry.org/diego-logging-client/testhelpers"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	routingAPI "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/test_helpers"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"google.golang.org/grpc/grpclog"
)

var (
	defaultConfig                                           testrunner.RoutingAPITestConfig
	client                                                  routingAPI.Client
	locketBinPath                                           string
	routingAPIBinPath                                       string
	routingAPIPort                                          uint16
	routingAPIMTLSPort                                      uint16
	routingAPIAdminPort                                     uint16
	oAuthServer                                             *ghttp.Server
	oAuthServerPort                                         string
	locketPort                                              uint16
	locketProcess                                           ifrit.Process
	databaseName                                            string
	dbAllocator                                             testrunner.DbAllocator
	sqlDBConfig                                             *config.SqlDB
	uaaCACertsPath                                          string
	mTLSAPIServerKeyPath                                    string
	mTLSAPIServerCertPath                                   string
	apiCAPath                                               string
	mTLSAPIClientCert                                       tls.Certificate
	testMetricsChan                                         chan *loggregator_v2.Envelope
	signalMetricsChan                                       chan struct{}
	testIngressServer                                       *testhelpers.TestIngressServer
	metronCAFile, metronServerCertFile, metronServerKeyFile string
	metricsPort                                             int
)

func TestRoutingAPI(test *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(test, "Routing API Test Suite")
}

var _ = SynchronizedBeforeSuite(
	func() []byte {
		routingAPIBin, err := gexec.Build("code.cloudfoundry.org/routing-api/cmd/routing-api", "-race")
		Expect(err).NotTo(HaveOccurred())

		locketPath, err := gexec.Build("code.cloudfoundry.org/locket/cmd/locket", "-race")
		Expect(err).NotTo(HaveOccurred())

		return []byte(strings.Join([]string{routingAPIBin, locketPath}, ","))
	},
	func(binPaths []byte) {
		grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))

		path := string(binPaths)
		routingAPIBinPath = strings.Split(path, ",")[0]
		locketBinPath = strings.Split(path, ",")[1]

		SetDefaultEventuallyTimeout(15 * time.Second)

		dbAllocator = testrunner.NewDbAllocator()

		var err error
		sqlDBConfig, err = dbAllocator.Create()
		Expect(err).NotTo(HaveOccurred(), "error occurred starting database client, is the database running?")
		databaseName = sqlDBConfig.Schema

		caCert, caPrivateKey, err := createCA()
		Expect(err).ToNot(HaveOccurred())

		f, err := os.CreateTemp("", "routing-api-uaa-ca")
		Expect(err).ToNot(HaveOccurred())

		uaaCACertsPath = f.Name()

		err = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
		Expect(err).ToNot(HaveOccurred())

		err = f.Close()
		Expect(err).ToNot(HaveOccurred())

		uaaServerCert, err := createCertificate(caCert, caPrivateKey, isCA)
		Expect(err).ToNot(HaveOccurred())

		apiCAPath, mTLSAPIServerCertPath, mTLSAPIServerKeyPath, mTLSAPIClientCert = tlsHelpers.GenerateCaAndMutualTlsCerts()

		oAuthServer, oAuthServerPort = testrunner.SetupOauthServer(uaaServerCert)
	},
)

var _ = SynchronizedAfterSuite(func() {
	err := dbAllocator.Delete()
	Expect(err).NotTo(HaveOccurred())

	oAuthServer.Close()

	err = os.Remove(uaaCACertsPath)
	Expect(err).NotTo(HaveOccurred())
}, func() {
	gexec.CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	routingAPIPort = uint16(test_helpers.NextAvailPort())
	routingAPIMTLSPort = uint16(test_helpers.NextAvailPort())

	fixturesPath := "fixtures"

	var err error
	metronCAFile = path.Join(fixturesPath, "metron", "CA.crt")
	metronServerCertFile = path.Join(fixturesPath, "metron", "metron.crt")
	metronServerKeyFile = path.Join(fixturesPath, "metron", "metron.key")
	testIngressServer, err = testhelpers.NewTestIngressServer(metronServerCertFile, metronServerKeyFile, metronCAFile)
	Expect(err).NotTo(HaveOccurred())
	receiversChan := testIngressServer.Receivers()
	testIngressServer.Start()
	metricsPort, _ = testIngressServer.Port()
	fmt.Println(" metric port :", metricsPort)

	testMetricsChan, signalMetricsChan = testhelpers.TestMetricChan(receiversChan)

	client = testrunner.RoutingApiClientWithPort(routingAPIPort, testrunner.RoutingAPIIP)
	err = dbAllocator.Reset()
	Expect(err).NotTo(HaveOccurred())

	locketPort = uint16(test_helpers.NextAvailPort())
	loggregatorConfig := loggingclient.Config{
		APIPort:    metricsPort,
		CACertPath: path.Join(fixturesPath, "metron", "CA.crt"),
		CertPath:   path.Join(fixturesPath, "metron", "client.crt"),
		KeyPath:    path.Join(fixturesPath, "metron", "client.key"),
	}
	locketProcess = testrunner.StartLocket(
		locketPort,
		locketBinPath,
		databaseName,
		sqlDBConfig.CACert,
		loggregatorConfig,
	)

	oAuthServerPort, err := strconv.ParseUint(oAuthServerPort, 10, 16)
	Expect(err).NotTo(HaveOccurred())

	locketAddress := fmt.Sprintf("%s:%d", testrunner.Host, locketPort)
	locketConfig := locketrunner.ClientLocketConfig()
	locketConfig.LocketAddress = locketAddress
	routingAPIAdminPort = test_helpers.NextAvailPort()
	defaultConfig = testrunner.GetRoutingAPITestConfig(
		routingAPIPort,
		routingAPIAdminPort,
		routingAPIMTLSPort,
		uint16(oAuthServerPort),
		uaaCACertsPath,
		databaseName,
		mTLSAPIServerCertPath,
		mTLSAPIServerKeyPath,
		apiCAPath,
		locketConfig,
	)
})

var _ = AfterEach(func() {
	testIngressServer.Stop()
	close(signalMetricsChan)
	testrunner.StopLocket(locketProcess)
})
