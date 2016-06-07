package db_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var etcdClient storeadapter.StoreAdapter
var etcdPort int
var etcdUrl string
var etcdRunner *etcdstorerunner.ETCDClusterRunner
var etcdVersion = "etcdserver\":\"2.1.1"
var routingAPIBinPath string
var basePath = path.Join(os.Getenv("GOPATH"), "src", "github.com", "cloudfoundry-incubator", "routing-api", "fixtures", "etcd-certs")

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	serverSSLConfig := &etcdstorerunner.SSLConfig{
		CertFile: filepath.Join(basePath, "server.crt"),
		KeyFile:  filepath.Join(basePath, "server.key"),
		CAFile:   filepath.Join(basePath, "etcd-ca.crt"),
	}

	etcdPort = 4001 + GinkgoParallelNode()
	etcdUrl = fmt.Sprintf("https://127.0.0.1:%d", etcdPort)
	etcdRunner = etcdstorerunner.NewETCDClusterRunner(etcdPort, 1, serverSSLConfig)
	etcdRunner.Start()

	clientSSLConfig := &etcdstorerunner.SSLConfig{
		filepath.Join(basePath, "client.crt"),
		filepath.Join(basePath, "client.key"),
		filepath.Join(basePath, "etcd-ca.crt"),
	}
	etcdClient = etcdRunner.Adapter(clientSSLConfig)

	RunSpecs(t, "DB Suite")

	etcdRunner.Stop()
}

var _ = BeforeSuite(func() {
	Expect(len(etcdRunner.NodeURLS())).Should(BeNumerically(">=", 1))

	tlsConfig, err := cf_http.NewTLSConfig(
		filepath.Join(basePath, "client.crt"),
		filepath.Join(basePath, "client.key"),
		filepath.Join(basePath, "etcd-ca.crt"))
	Expect(err).ToNot(HaveOccurred())

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}

	etcdVersionUrl := etcdRunner.NodeURLS()[0] + "/version"
	resp, err := client.Get(etcdVersionUrl)
	Expect(err).ToNot(HaveOccurred())

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())

	// response body: {"etcdserver":"2.1.1","etcdcluster":"2.1.0"}
	Expect(string(body)).To(ContainSubstring(etcdVersion))
})

var _ = BeforeEach(func() {
	etcdRunner.Reset()
})
