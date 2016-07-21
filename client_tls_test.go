package routing_api_test

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/models"
	"github.com/cloudfoundry-incubator/trace-logger"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Client", func() {
	const (
		ROUTES_API_URL = "/routing/v1/routes"
	)
	var server *ghttp.Server
	var client routing_api.Client
	var stdout *bytes.Buffer

	BeforeEach(func() {
		stdout = bytes.NewBuffer([]byte{})
		trace.SetStdout(stdout)
		trace.Logger = trace.NewLogger("true")
	})

	BeforeEach(func() {
		server = ghttp.NewTLSServer()
		data, _ := json.Marshal([]models.Route{})
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", ROUTES_API_URL),
				ghttp.RespondWith(http.StatusOK, data),
			),
		)
	})

	AfterEach(func() {
		server.Close()
	})

	Context("without skip SSL validation", func() {
		BeforeEach(func() {
			client = routing_api.NewClient(server.URL(), false)
		})

		It("fails to connect to the Routing API", func() {
			_, err := client.Routes()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("x509: certificate signed by unknown authority"))
		})
	})

	Context("with skip SSL validation", func() {
		BeforeEach(func() {
			client = routing_api.NewClient(server.URL(), true)
		})

		It("successfully connect to the Routing API", func() {
			_, err := client.Routes()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
