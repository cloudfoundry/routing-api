package testrunner

import (
	"code.cloudfoundry.org/routing-api/config"
	"crypto/tls"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"strings"
)

func WriteConfigToTempFile(conf *config.Config) string {
	bytes, err := yaml.Marshal(conf)
	Expect(err).ToNot(HaveOccurred())

	tempFile, err := os.CreateTemp("", "routing_api_config.yml")
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(tempFile.Close()).To(Succeed())
	}()

	_, err = tempFile.Write(bytes)
	Expect(err).ToNot(HaveOccurred())

	return tempFile.Name()
}

func getServerPort(url string) string {
	endpoints := strings.Split(url, ":")
	Expect(endpoints).To(HaveLen(3))
	return endpoints[2]
}

func SetupOauthServer(uaaServerCert tls.Certificate) (*ghttp.Server, string) {
	oAuthServer := ghttp.NewUnstartedServer()

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{uaaServerCert},
	}

	oAuthServer.HTTPTestServer.TLS = tlsConfig
	oAuthServer.AllowUnhandledRequests = true
	oAuthServer.UnhandledRequestStatusCode = http.StatusOK
	oAuthServer.HTTPTestServer.StartTLS()

	oAuthServerPort := getServerPort(oAuthServer.URL())

	return oAuthServer, oAuthServerPort
}
