package main_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"google.golang.org/grpc/grpclog"

	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"
	"code.cloudfoundry.org/routing-api/test_helpers"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"

	"testing"
	"time"
)

var (
	defaultConfig          customConfig
	client                 routing_api.Client
	locketBinPath          string
	routingAPIBinPath      string
	routingAPIAddress      string
	routingAPIPort         uint16
	routingAPIAdminPort    int
	routingAPIIP           string
	routingAPISystemDomain string
	oauthServer            *ghttp.Server
	oauthServerPort        string

	sqlDBName    string
	gormDB       *gorm.DB
	consulRunner *consulrunner.ClusterRunner

	mysqlAllocator testrunner.DbAllocator
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
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
		grpclog.SetLogger(log.New(ioutil.Discard, "", 0))

		var err error
		path := string(binPaths)
		routingAPIBinPath = strings.Split(path, ",")[0]
		locketBinPath = strings.Split(path, ",")[1]

		SetDefaultEventuallyTimeout(15 * time.Second)

		mysqlAllocator = testrunner.NewMySQLAllocator()

		sqlDBName, err = mysqlAllocator.Create()
		Expect(err).NotTo(HaveOccurred())

		setupConsul()
		setupOauthServer()
	},
)

var _ = SynchronizedAfterSuite(func() {
	err := mysqlAllocator.Delete()
	Expect(err).NotTo(HaveOccurred())

	teardownConsul()
	oauthServer.Close()
}, func() {
	gexec.CleanupBuildArtifacts()
})

var _ = BeforeEach(func() {
	client = routingApiClient()
	err := mysqlAllocator.Reset()
	Expect(err).NotTo(HaveOccurred())
	resetConsul()

	caCertsPath, err := filepath.Abs(filepath.Join("..", "..", "fixtures", "uaa-certs", "uaa-ca.pem"))
	Expect(err).NotTo(HaveOccurred())

	oauthSrvPort, err := strconv.ParseInt(oauthServerPort, 10, 0)
	Expect(err).NotTo(HaveOccurred())

	routingAPIAdminPort = test_helpers.NextAvailPort()
	defaultConfig = customConfig{
		StatsdPort:  8125 + GinkgoParallelNode(),
		AdminPort:   routingAPIAdminPort,
		UAAPort:     int(oauthSrvPort),
		CACertsPath: caCertsPath,
		Schema:      sqlDBName,
		ConsulUrl:   consulRunner.URL(),
		UseSQL:      true,
	}
})

type customConfig struct {
	Port        int
	StatsdPort  int
	UAAPort     int
	AdminPort   int
	CACertsPath string
	Schema      string
	ConsulUrl   string
	UseSQL      bool
}

func getRoutingAPIConfig(c customConfig) *config.Config {
	rapiConfig := &config.Config{
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
		ConsulCluster: config.ConsulCluster{
			Servers:       c.ConsulUrl,
			RetryInterval: 50 * time.Millisecond,
		},
		UUID: "fake-uuid",
	}
	rapiConfig.SqlDB = config.SqlDB{
		Host:     "localhost",
		Port:     3306,
		Schema:   c.Schema,
		Type:     "mysql",
		Username: "root",
		Password: "password",
	}
	return rapiConfig
}

func writeConfigToTempFile(c *config.Config) string {
	d, err := yaml.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	tmpfile, err := ioutil.TempFile("", "routing_api_config.yml")
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(tmpfile.Close()).To(Succeed())
	}()

	_, err = tmpfile.Write(d)
	Expect(err).ToNot(HaveOccurred())

	return tmpfile.Name()
}

func routingApiClient() routing_api.Client {
	routingAPIPort = uint16(test_helpers.NextAvailPort())
	routingAPIIP = "127.0.0.1"
	routingAPISystemDomain = "example.com"
	routingAPIAddress = fmt.Sprintf("%s:%d", routingAPIIP, routingAPIPort)

	routingAPIURL := &url.URL{
		Scheme: "http",
		Host:   routingAPIAddress,
	}

	return routing_api.NewClient(routingAPIURL.String(), false)
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

func setupOauthServer() {
	oauthServer = ghttp.NewUnstartedServer()
	basePath, err := filepath.Abs(path.Join("..", "..", "fixtures", "uaa-certs"))
	Expect(err).ToNot(HaveOccurred())

	cert, err := tls.LoadX509KeyPair(filepath.Join(basePath, "server.pem"), filepath.Join(basePath, "server.key"))
	Expect(err).ToNot(HaveOccurred())
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	oauthServer.HTTPTestServer.TLS = tlsConfig
	oauthServer.AllowUnhandledRequests = true
	oauthServer.UnhandledRequestStatusCode = http.StatusOK
	oauthServer.HTTPTestServer.StartTLS()

	oauthServerPort = getServerPort(oauthServer.URL())
}

func setupConsul() {
	consulRunner = consulrunner.NewClusterRunner(consulrunner.ClusterRunnerConfig{
		StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
		NumNodes:     1,
		Scheme:       "http",
	})
	consulRunner.Start()
	consulRunner.WaitUntilReady()
}

func teardownConsul() {
	consulRunner.Stop()
}

func resetConsul() {
	err := consulRunner.Reset()
	Expect(err).ToNot(HaveOccurred())
}

func getServerPort(url string) string {
	endpoints := strings.Split(url, ":")
	Expect(endpoints).To(HaveLen(3))
	return endpoints[2]
}
