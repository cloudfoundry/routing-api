package main_test

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.cloudfoundry.org/locket"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"

	tls_helpers "code.cloudfoundry.org/cf-routing-test-helpers/tls"
	locketconfig "code.cloudfoundry.org/locket/cmd/locket/config"
	locketrunner "code.cloudfoundry.org/locket/cmd/locket/testrunner"
	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"
	"code.cloudfoundry.org/routing-api/test_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"google.golang.org/grpc/grpclog"
	yaml "gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

var (
	defaultConfig          customConfig
	client                 routing_api.Client
	locketBinPath          string
	routingAPIBinPath      string
	routingAPIAddress      string
	routingAPIPort         uint16
	routingAPIMTLSPort     uint16
	routingAPIAdminPort    int
	routingAPIIP           string
	routingAPISystemDomain string
	oauthServer            *ghttp.Server
	oauthServerPort        string
	locketPort             uint16
	locketProcess          ifrit.Process

	sqlDBName string

	dbAllocator testrunner.DbAllocator
	sqlDBConfig *config.SqlDB

	uaaCACertsPath string

	mtlsAPIServerKeyPath  string
	mtlsAPIServerCertPath string
	apiCAPath             string
	mtlsAPIClientCert     tls.Certificate
)

func TestMainSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Main Suite", suiteConfig, reporterConfig)
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
		sqlDBName = sqlDBConfig.Schema

		caCert, caPrivKey, err := createCA()
		Expect(err).ToNot(HaveOccurred())

		f, err := os.CreateTemp("", "routing-api-uaa-ca")
		Expect(err).ToNot(HaveOccurred())

		uaaCACertsPath = f.Name()

		err = pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
		Expect(err).ToNot(HaveOccurred())

		err = f.Close()
		Expect(err).ToNot(HaveOccurred())

		uaaServerCert, err := createCertificate(caCert, caPrivKey, isCA)
		Expect(err).ToNot(HaveOccurred())

		apiCAPath, mtlsAPIServerCertPath, mtlsAPIServerKeyPath, mtlsAPIClientCert = tls_helpers.GenerateCaAndMutualTlsCerts()

		setupOauthServer(uaaServerCert)
	},
)

var _ = SynchronizedAfterSuite(func() {
	err := dbAllocator.Delete()
	Expect(err).NotTo(HaveOccurred())

	oauthServer.Close()

	err = os.Remove(uaaCACertsPath)
	Expect(err).NotTo(HaveOccurred())
}, func() {
	gexec.CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	routingAPIPort = uint16(test_helpers.NextAvailPort())
	routingAPIMTLSPort = uint16(test_helpers.NextAvailPort())

	client = routingApiClientWithPort(routingAPIPort)
	err := dbAllocator.Reset()
	Expect(err).NotTo(HaveOccurred())

	startLocket()

	oauthSrvPort, err := strconv.ParseInt(oauthServerPort, 10, 0)
	Expect(err).NotTo(HaveOccurred())

	locketAddress := fmt.Sprintf("localhost:%d", locketPort)
	locketConfig := locketrunner.ClientLocketConfig()
	locketConfig.LocketAddress = locketAddress

	routingAPIAdminPort = test_helpers.NextAvailPort()
	defaultConfig = customConfig{
		APIServerHTTPEnabled: true,
		Port:                 int(routingAPIPort),
		StatsdPort:           8125 + GinkgoParallelProcess(),
		AdminPort:            routingAPIAdminPort,
		UAAPort:              int(oauthSrvPort),
		CACertsPath:          uaaCACertsPath,
		Schema:               sqlDBName,

		UseSQL: true,

		LocketConfig: locketConfig,

		// mTLS API
		APIServerMTLSPort: int(routingAPIMTLSPort),
		APIServerCertPath: mtlsAPIServerCertPath,
		APIServerKeyPath:  mtlsAPIServerKeyPath,
		APICAPath:         apiCAPath,
	}
})

var _ = AfterEach(func() {
	stopLocket()
})

type customConfig struct {
	Port         int
	StatsdPort   int
	UAAPort      int
	AdminPort    int
	LocketConfig locket.ClientLocketConfig
	CACertsPath  string
	Schema       string
	UseSQL       bool

	APIServerHTTPEnabled bool
	APIServerMTLSPort    int
	APIServerCertPath    string
	APIServerKeyPath     string
	APICAPath            string
}

func getRoutingAPIConfig(c customConfig) *config.Config {
	rapiConfig := &config.Config{
		API: config.APIConfig{
			ListenPort:         c.Port,
			HTTPEnabled:        c.APIServerHTTPEnabled,
			MTLSListenPort:     c.APIServerMTLSPort,
			MTLSClientCAPath:   c.APICAPath,
			MTLSServerCertPath: c.APIServerCertPath,
			MTLSServerKeyPath:  c.APIServerKeyPath,
		},
		AdminPort:    c.AdminPort,
		DebugAddress: "1.2.3.4:1234",
		LogGuid:      "my_logs",
		MetronConfig: config.MetronConfig{
			Address: "1.2.3.4",
			Port:    "4567",
		},
		SystemDomain:                    "example.com",
		MetricsReportingIntervalString:  "500ms",
		StatsdEndpoint:                  fmt.Sprintf("localhost:%d", c.StatsdPort),
		StatsdClientFlushIntervalString: "10ms",
		OAuth: config.OAuthConfig{
			TokenEndpoint:     "127.0.0.1",
			Port:              c.UAAPort,
			SkipSSLValidation: false,
			CACerts:           c.CACertsPath,
		},
		RouterGroups: models.RouterGroups{
			models.RouterGroup{
				Name:            "default-tcp",
				Type:            "tcp",
				ReservablePorts: "1024-65535",
			},
		},
		RetryInterval: 50 * time.Millisecond,
		UUID:          "fake-uuid",
		Locket:        c.LocketConfig,
	}
	switch os.Getenv("DB") {
	case "postgres":
		rapiConfig.SqlDB = config.SqlDB{
			Host:              "localhost",
			Port:              5432,
			Schema:            c.Schema,
			Type:              "postgres",
			Username:          "postgres",
			Password:          "",
			CACert:            os.Getenv("SQL_SERVER_CA_CERT"),
			SkipSSLValidation: os.Getenv("DB_SKIP_SSL_VALIDATION") == "true",
		}
	default:
		rapiConfig.SqlDB = config.SqlDB{
			Host:              "localhost",
			Port:              3306,
			Schema:            c.Schema,
			Type:              "mysql",
			Username:          "root",
			Password:          "password",
			CACert:            os.Getenv("SQL_SERVER_CA_CERT"),
			SkipSSLValidation: os.Getenv("DB_SKIP_SSL_VALIDATION") == "true",
		}
	}
	return rapiConfig
}

func writeConfigToTempFile(c *config.Config) string {
	d, err := yaml.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	tmpfile, err := os.CreateTemp("", "routing_api_config.yml")
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(tmpfile.Close()).To(Succeed())
	}()

	_, err = tmpfile.Write(d)
	Expect(err).ToNot(HaveOccurred())

	return tmpfile.Name()
}

func routingApiClientWithPort(routingAPIPort uint16) routing_api.Client {
	routingAPIIP = "127.0.0.1"
	routingAPISystemDomain = "example.com"
	routingAPIAddress = fmt.Sprintf("%s:%d", routingAPIIP, routingAPIPort)

	routingAPIURL := &url.URL{
		Scheme: "http",
		Host:   routingAPIAddress,
	}

	return routing_api.NewClient(routingAPIURL.String(), false)
}

func setupOauthServer(uaaServerCert tls.Certificate) {
	oauthServer = ghttp.NewUnstartedServer()

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{uaaServerCert},
	}
	oauthServer.HTTPTestServer.TLS = tlsConfig
	oauthServer.AllowUnhandledRequests = true
	oauthServer.UnhandledRequestStatusCode = http.StatusOK
	oauthServer.HTTPTestServer.StartTLS()

	oauthServerPort = getServerPort(oauthServer.URL())
}

func startLocket() {
	locketPort = uint16(test_helpers.NextAvailPort())
	locketAddress := fmt.Sprintf("localhost:%d", locketPort)

	locketRunner := locketrunner.NewLocketRunner(locketBinPath, func(cfg *locketconfig.LocketConfig) {
		switch os.Getenv("DB") {
		case "postgres":
			cfg.DatabaseConnectionString = "user=postgres password= host=localhost dbname=" + sqlDBName
			cfg.DatabaseDriver = "postgres"
		default:
			connStr := "root:password@/"
			cfg.DatabaseConnectionString = connStr + sqlDBName
			cfg.DatabaseDriver = "mysql"
		}
		if sqlDBConfig.CACert != "" {
			caFile, err := os.CreateTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(caFile.Name(), []byte(sqlDBConfig.CACert), 0400)).To(Succeed())
			cfg.SQLCACertFile = caFile.Name()
		}
		cfg.ListenAddress = locketAddress
	})
	locketProcess = ginkgomon.Invoke(locketRunner)
}

func stopLocket() {
	ginkgomon.Interrupt(locketProcess)
	locketProcess.Wait()
}

func getServerPort(url string) string {
	endpoints := strings.Split(url, ":")
	Expect(endpoints).To(HaveLen(3))
	return endpoints[2]
}
