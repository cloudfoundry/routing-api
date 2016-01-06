package authentication_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/routing-api/authentication"
	"github.com/cloudfoundry-incubator/routing-api/metrics"
	"github.com/pivotal-golang/lager/lagertest"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("UaaKeyFetcher", func() {

	const (
		TOKEN_KEY_ENDPOINT = "/token_key"
	)
	var (
		server        *ghttp.Server
		uaaKeyFetcher authentication.UaaKeyFetcher
		logger        *lagertest.TestLogger
		keyValue      string
		err           error
	)
	Context("FetchKey", func() {
		BeforeEach(func() {
			server = ghttp.NewServer()
			logger = lagertest.NewTestLogger("uaa-key-fetcher-test")
			uaaKeyFetcher = authentication.NewUaaKeyFetcher(logger, server.URL()+TOKEN_KEY_ENDPOINT)
		})

		AfterEach(func() {
			server.Close()
		})

		JustBeforeEach(func() {
			keyValue, err = uaaKeyFetcher.FetchKey()
		})

		Context("when UAA is available and responsive", func() {

			Context("and http request succeeds", func() {
				var (
					currentKeyRefreshCount int64
				)

				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{}`),
						),
					)
					currentKeyRefreshCount = metrics.GetKeyVerificationRefreshCount()
				})
				It("increments the KeyRefresh metric", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(metrics.GetKeyVerificationRefreshCount()).To(Equal(currentKeyRefreshCount + 1))
				})
			})

			Context("and returns a valid uaa key", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{"alg":"alg", "value": "AABBCC" }`),
						),
					)
				})

				It("returns the key value", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(keyValue).NotTo(BeNil())
					Expect(keyValue).Should(Equal("AABBCC"))
				})

				It("logs success message", func() {
					Expect(logger).Should(gbytes.Say("fetch-key-successful"))
				})
			})

			Context("and returns a invalid json uaa key", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{"alg":"alg", "value": "ooooppps }`),
						),
					)
				})

				It("returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(keyValue).To(BeEmpty())
				})

				It("logs error message", func() {
					Expect(logger).Should(gbytes.Say("error-in-unmarshaling-key"))
				})
			})

			Context("and returns an http error", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusInternalServerError, `{}`),
						),
					)
				})

				It("returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).Should(Equal(errors.New("http-error-fetching-key")))
					Expect(keyValue).To(BeEmpty())
				})

				It("logs error message", func() {
					Expect(logger).Should(gbytes.Say("http-error-fetching-key"))
				})
			})
		})

		Context("when UAA is unavailable", func() {

			BeforeEach(func() {
				uaaKeyFetcher = authentication.NewUaaKeyFetcher(logger, "http://127.0.0.1:1111/token_key")
			})

			It("returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(keyValue).To(BeEmpty())
			})

			It("logs error message", func() {
				Expect(logger).Should(gbytes.Say("error-in-fetching-key"))
			})
		})
	})
})
