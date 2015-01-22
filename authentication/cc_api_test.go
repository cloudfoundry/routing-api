package authentication_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/pivotal-cf-experimental/routing-api/authentication"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CCApi", func() {
	var (
		server      *httptest.Server
		uaaEndpoint string
		ccApi       authentication.CCApi
		logger      *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("cc-api-test")
	})

	Describe(".GetInfo", func() {
		Context("when we can successfully reach CC", func() {
			JustBeforeEach(func() {
				successBody := fmt.Sprintf(`{"token_endpoint":"%s"}`, uaaEndpoint)
				successHandler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(successBody))
				}

				server = httptest.NewServer(http.HandlerFunc(successHandler))
				ccApi = authentication.NewCCApi(server.URL, logger)
			})

			Context("when the uaa endpoint is http", func() {
				BeforeEach(func() {
					uaaEndpoint = "http://uaa.example.com"
				})

				It("returns the token_endpoint from /v2/info", func() {
					info := ccApi.GetInfo()
					Expect(info.TokenEndpoint).To(Equal(uaaEndpoint))
				})
			})

			Context("when the uaa endpoint is https", func() {
				BeforeEach(func() {
					uaaEndpoint = "https://uaa.example.com"
				})

				It("returns the token_endpoint from /v2/info as http", func() {
					info := ccApi.GetInfo()
					Expect(info.TokenEndpoint).To(Equal("http://uaa.example.com"))
				})
			})
		})

		Context("when we cannot reach CC", func() {
			BeforeEach(func() {
				successHandler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}

				server = httptest.NewServer(http.HandlerFunc(successHandler))
				ccApi = authentication.NewCCApi(server.URL, logger)
			})

			It("logs the error", func() {
				info := ccApi.GetInfo()
				Expect(info).To(BeNil())
				Expect(logger.Logs()[0].Data["error"]).To(ContainSubstring("Cloud Controller could not be reached"))
			})
		})
	})
})
