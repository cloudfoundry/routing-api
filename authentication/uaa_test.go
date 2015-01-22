package authentication_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/routing-api/authentication"
	"github.com/pivotal-cf-experimental/routing-api/authentication/fakes"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("UAA", func() {
	var (
		authKey string
		ccApi   *fakes.FakeCloudControllerApi
		uaa     authentication.UAA
		logger  *lagertest.TestLogger
	)

	BeforeEach(func() {
		authKey = "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d\nKVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX\nqHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug\nspULZVNRxq7veq/fzwIDAQAB\n-----END PUBLIC KEY-----\n"

		logger = lagertest.NewTestLogger("cc-api-test")
		ccApi = &fakes.FakeCloudControllerApi{}
		uaa = authentication.NewUAA(ccApi, logger)
	})

	Describe(".GetVerificationToken", func() {
		var (
			server *httptest.Server
		)

		Context("when we can successfully reach both Cloud Controller and Uaa", func() {
			BeforeEach(func() {
				successBody := authentication.AccessToken{Value: authKey}

				body, err := json.Marshal(successBody)
				Expect(err).NotTo(HaveOccurred())

				successHandler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write(body)
				}

				server = httptest.NewServer(http.HandlerFunc(successHandler))
				ccApi.GetInfoReturns(&authentication.Info{server.URL})
			})

			It("Returns an AccessToken on success", func() {
				token := uaa.GetVerificationToken()
				Expect(token.Value).To(Equal(authKey))
			})
		})

		Context("when getting the verification token is unsuccessful", func() {
			Context("communication with Cloud Controller", func() {
				BeforeEach(func() {
					ccApi.GetInfoReturns(nil)
				})

				It("returns nil", func() {
					token := uaa.GetVerificationToken()
					Expect(token).To(BeNil())
				})
			})

			Context("communicating with UAA", func() {
				BeforeEach(func() {
					errorHandler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusNotFound)
					}

					server = httptest.NewServer(http.HandlerFunc(errorHandler))
					ccApi.GetInfoReturns(&authentication.Info{server.URL})
				})

				It("logs unable to communicate with UAA on failure", func() {
					token := uaa.GetVerificationToken()
					Expect(token).To(BeNil())
					Expect(logger.Logs()[0].Data["error"]).To(ContainSubstring("UAA could not be reached"))
				})
			})
		})
	})
})
