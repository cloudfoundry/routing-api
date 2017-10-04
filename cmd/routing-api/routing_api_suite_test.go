package main_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"google.golang.org/grpc/grpclog"

	"code.cloudfoundry.org/consuladapter/consulrunner"
	"code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/models"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"

	"testing"
	"time"
)

var (
	etcdPort int
	etcdUrl  string

	defaultConfig          customConfig
	client                 routing_api.Client
	locketBinPath          string
	routingAPIBinPath      string
	routingAPIAddress      string
	routingAPIPort         uint16
	routingAPIAdminSocket  string
	routingAPIIP           string
	routingAPISystemDomain string
	oauthServer            *ghttp.Server
	oauthServerPort        string

	sqlDBName    string
	gormDB       *gorm.DB
	consulRunner *consulrunner.ClusterRunner

	mysqlAllocator testrunner.DbAllocator
	etcdAllocator  testrunner.DbAllocator
)

var etcdVersion = "etcdserver\":\"2.1.1"

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
		etcdPort = 4001 + GinkgoParallelNode()

		mysqlAllocator = testrunner.NewMySQLAllocator()
		etcdAllocator = testrunner.NewEtcdAllocator(etcdPort)

		sqlDBName, err = mysqlAllocator.Create()
		Expect(err).NotTo(HaveOccurred())

		etcdUrl, err = etcdAllocator.Create()
		Expect(err).NotTo(HaveOccurred())

		setupConsul()
		setupOauthServer()
	},
)

var _ = SynchronizedAfterSuite(func() {
	err := mysqlAllocator.Delete()
	Expect(err).NotTo(HaveOccurred())
	err = etcdAllocator.Delete()
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
	err = etcdAllocator.Reset()
	Expect(err).NotTo(HaveOccurred())
	resetConsul()

	caCertsPath, err := filepath.Abs(filepath.Join("..", "..", "fixtures", "uaa-certs", "uaa-ca.pem"))
	Expect(err).NotTo(HaveOccurred())

	oauthSrvPort, err := strconv.ParseInt(oauthServerPort, 10, 0)
	Expect(err).NotTo(HaveOccurred())

	routingAPIAdminSocket = tempUnixSocket()
	defaultConfig = customConfig{
		StatsdPort:  8125 + GinkgoParallelNode(),
		AdminSocket: routingAPIAdminSocket,
		UAAPort:     int(oauthSrvPort),
		CACertsPath: caCertsPath,
		EtcdPort:    etcdPort,
		Schema:      sqlDBName,
		ConsulUrl:   consulRunner.URL(),
		UseSQL:      true,
		UseETCD:     true,
	}
})

type customConfig struct {
	EtcdPort    int
	Port        int
	StatsdPort  int
	UAAPort     int
	AdminSocket string
	CACertsPath string
	Schema      string
	ConsulUrl   string
	UseETCD     bool
	UseSQL      bool
}

func getRoutingAPIConfig(c customConfig) *config.Config {
	rapiConfig := &config.Config{
		AdminSocket:  c.AdminSocket,
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
	if c.UseETCD {
		rapiConfig.Etcd = config.Etcd{
			NodeURLS: []string{fmt.Sprintf("http://localhost:%d", c.EtcdPort)},
		}
	}
	if c.UseSQL {
		rapiConfig.SqlDB = config.SqlDB{
			Host:     "localhost",
			Port:     3306,
			Schema:   c.Schema,
			Type:     "mysql",
			Username: "root",
			Password: "password",
		}
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
	routingAPIPort = uint16(testPort())
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

var (
	lastPortUsed int
	portLock     sync.Mutex
	once         sync.Once
)

func testPort() int {
	portLock.Lock()
	defer portLock.Unlock()

	if lastPortUsed == 0 {
		once.Do(func() {
			const portRangeStart = 61000
			lastPortUsed = portRangeStart + GinkgoConfig.ParallelNode
		})
	}

	lastPortUsed += GinkgoConfig.ParallelTotal
	return lastPortUsed
}

func validatePort(port uint16) {
	Eventually(func() error {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if l != nil {
			_ = l.Close()
		}
		return err
	}, "60s", "1s").Should(BeNil())
}

func tempUnixSocket() string {
	tmpfile, err := ioutil.TempFile("", "admin.sock")
	Expect(err).ToNot(HaveOccurred())
	defer Expect(os.Remove(tmpfile.Name())).To(Succeed())

	err = tmpfile.Close()
	Expect(err).ToNot(HaveOccurred())
	return tmpfile.Name()
}
