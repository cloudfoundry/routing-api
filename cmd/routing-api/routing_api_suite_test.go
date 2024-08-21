package main_test

import (
	testHelpers "code.cloudfoundry.org/routing-api/test_helpers"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	tlsHelpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	routingAPI "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/config"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"google.golang.org/grpc/grpclog"
)

var (
	defaultConfig         testHelpers.RoutingAPITestConfig
	client                routingAPI.Client
	locketBinPath         string
	routingAPIBinPath     string
	routingAPIPort        uint16
	routingAPIMTLSPort    uint16
	routingAPIAdminPort   int
	oAuthServer           *ghttp.Server
	oAuthServerPort       string
	locketPort            uint16
	locketProcess         ifrit.Process
	databaseName          string
	dbAllocator           testHelpers.DbAllocator
	sqlDBConfig           *config.SqlDB
	uaaCACertsPath        string
	mTLSAPIServerKeyPath  string
	mTLSAPIServerCertPath string
	apiCAPath             string
	mTLSAPIClientCert     tls.Certificate
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

		dbAllocator = testHelpers.NewDbAllocator()

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

		oAuthServer, oAuthServerPort = testHelpers.SetupOauthServer(uaaServerCert)
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
	routingAPIPort = uint16(testHelpers.NextAvailPort())
	routingAPIMTLSPort = uint16(testHelpers.NextAvailPort())

	client = testHelpers.RoutingApiClientWithPort(routingAPIPort, testHelpers.RoutingAPIIP)
	err := dbAllocator.Reset()
	Expect(err).NotTo(HaveOccurred())

	locketPort = uint16(testHelpers.NextAvailPort())
	locketProcess = testHelpers.StartLocket(
		locketPort,
		locketBinPath,
		databaseName,
		sqlDBConfig.CACert,
	)

	oAuthServerPort, err := strconv.ParseInt(oAuthServerPort, 10, 0)
	Expect(err).NotTo(HaveOccurred())

	locketAddress := fmt.Sprintf("%s:%d", testHelpers.Host, locketPort)
	locketConfig := locketrunner.ClientLocketConfig()
	locketConfig.LocketAddress = locketAddress

	routingAPIAdminPort = testHelpers.NextAvailPort()
	defaultConfig = testHelpers.GetRoutingAPITestConfig(
		routingAPIPort,
		routingAPIAdminPort,
		routingAPIMTLSPort,
		oAuthServerPort,
		uaaCACertsPath,
		databaseName,
		mTLSAPIServerCertPath,
		mTLSAPIServerKeyPath,
		apiCAPath,
		locketConfig,
	)
})

var _ = AfterEach(func() {
	testHelpers.StopLocket(locketProcess)
})
