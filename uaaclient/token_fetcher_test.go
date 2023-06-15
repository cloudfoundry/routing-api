package uaaclient_test

import (
	"bytes"
	"context"
	"encoding/pem"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/oauth2"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/routing-api/trace"
	"code.cloudfoundry.org/routing-api/uaaclient"
)

var _ = Describe("TokenFetcher", func() {
	var (
		config         uaaclient.Config
		ctx            context.Context
		tokenFetcher   uaaclient.TokenFetcher
		clock          *fakeclock.FakeClock
		server         *ghttp.Server
		logger         lager.Logger
		forceUpdate    bool
		serverCertFile *os.File
	)

	// from oauth2/internal/token.go
	type tokenJSON struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}

	BeforeEach(func() {
		config = uaaclient.Config{
			SkipSSLValidation: true,
			ClientName:        "client-name",
			ClientSecret:      "client-secret",
		}
		clock = fakeclock.NewFakeClock(time.Now())
		logger = lagertest.NewTestLogger("test")
		server = ghttp.NewTLSServer()

		url, err := url.Parse(server.URL())
		Expect(err).ToNot(HaveOccurred())

		addr := strings.Split(url.Host, ":")

		config.TokenEndpoint = addr[0]

		port, err := strconv.Atoi(addr[1])
		Expect(err).ToNot(HaveOccurred())

		config.Port = port

		serverCertFile, err = ioutil.TempFile("", "routing-api-uaa-client-test")
		Expect(err).NotTo(HaveOccurred())

		certPem := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: server.HTTPTestServer.Certificate().Raw,
		})
		_, err = serverCertFile.Write(certPem)
		Expect(err).NotTo(HaveOccurred())

		tokenFetcher, err = uaaclient.NewTokenFetcher(
			false,
			config,
			clock,
			0,
			15*time.Second,
			30,
			logger,
		)
		Expect(err).ToNot(HaveOccurred())
		ctx = context.Background()
		forceUpdate = false
	})

	AfterEach(func() {
		err := os.Remove(serverCertFile.Name())
		Expect(err).NotTo(HaveOccurred())
		server.Close()
	})

	var verifyBody = func(expectedBody string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(body)).To(Equal(expectedBody))
		}
	}

	var verifyLogs = func(reqMessage, resMessage string) {
		Expect(logger).To(gbytes.Say(reqMessage))
		Expect(logger).To(gbytes.Say(resMessage))
	}

	var getOauthHandlerFunc = func(status int, response *tokenJSON, optionalHeader ...http.Header) http.HandlerFunc {
		return ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/oauth/token"),
			ghttp.VerifyBasicAuth("client-name", "client-secret"),
			ghttp.VerifyContentType("application/x-www-form-urlencoded"),
			verifyBody("grant_type=client_credentials&token_format=jwt"),
			ghttp.RespondWithJSONEncoded(status, response, optionalHeader...),
		)
	}

	Context("when the respose body is malformed", func() {
		It("returns an error and doesn't retry", func() {
			server.AppendHandlers(
				ghttp.RespondWithJSONEncoded(http.StatusOK, "broken garbage response"),
			)

			_, err := tokenFetcher.FetchToken(ctx, forceUpdate)
			Expect(err).To(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))

			verifyLogs("test", "test")
		})
	})

	Context("when OAuth server cannot be reached", func() {
		It("retries number of times and finally returns an error", func() {
			done := make(chan interface{})
			go func() {
				var err error

				defer close(done)
				config.TokenEndpoint = "bogus.url"
				tokenFetcher, err = uaaclient.NewTokenFetcher(
					false,
					config,
					clock,
					3,
					5*time.Millisecond,
					30,
					logger,
				)
				Expect(err).ToNot(HaveOccurred())
				wg := sync.WaitGroup{}
				wg.Add(1)
				go func(wg *sync.WaitGroup) {
					defer GinkgoRecover()
					defer wg.Done()
					_, err := tokenFetcher.FetchToken(ctx, forceUpdate)
					Expect(err).To(HaveOccurred())
				}(&wg)

				for i := 0; i < 3; i++ {
					Eventually(logger, 2*time.Second).Should(gbytes.Say("error-fetching-token.*bogus.url"))
					clock.WaitForWatcherAndIncrement(DefaultRetryInterval + 10*time.Second)
				}
				wg.Wait()
			}()
			Eventually(done, 5*time.Second).Should(BeClosed())
		})
	})

	Context("when OAuth server returns 200 OK", func() {
		It("returns a new token and trace the request response", func() {
			stdout := bytes.NewBuffer([]byte{})
			trace.SetStdout(stdout)
			trace.NewLogger("true")

			responseBody := &tokenJSON{
				AccessToken: "the token",
			}

			server.AppendHandlers(
				getOauthHandlerFunc(http.StatusOK, responseBody),
			)

			token, err := tokenFetcher.FetchToken(ctx, forceUpdate)
			Expect(err).NotTo(HaveOccurred())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(token.AccessToken).To(Equal("the token"))
		})
	})

	Context("when OAuth server returns a 3xx http status code", func() {
		var header http.Header
		BeforeEach(func() {
			header = make(http.Header)
			header.Add("Location", server.URL())
		})

		It("returns an error and doesn't retry", func() {
			server.AppendHandlers(
				ghttp.RespondWith(http.StatusMovedPermanently, "moved", header),
			)

			_, err := tokenFetcher.FetchToken(ctx, forceUpdate)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot fetch token: 301"))
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
		})
	})

	Context("when multiple goroutines fetch a token", func() {
		It("contacts oauth server only once and returns cached token", func() {
			server.AppendHandlers(
				getOauthHandlerFunc(http.StatusOK, &tokenJSON{
					AccessToken: "the token",
					ExpiresIn:   60,
				}),
				getOauthHandlerFunc(http.StatusOK, &tokenJSON{
					AccessToken: "the new token",
					ExpiresIn:   60,
				}))

			var token *oauth2.Token
			var err error
			forceUpdate = false
			for i := 0; i < 2; i++ {
				token, err = tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(token.AccessToken).To(Equal("the token"))
		})
		Context("and the token is about to expire", func() {
			It("contacts the oauth server again, and returns a new token", func() {
				server.AppendHandlers(
					getOauthHandlerFunc(http.StatusOK, &tokenJSON{
						AccessToken: "the token",
						ExpiresIn:   60,
					}),
					getOauthHandlerFunc(http.StatusOK, &tokenJSON{
						AccessToken: "the new token",
						ExpiresIn:   60,
					}))

				var token *oauth2.Token
				var err error
				forceUpdate = false
				for i := 0; i < 2; i++ {
					token, err = tokenFetcher.FetchToken(ctx, forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					clock.IncrementBySeconds(35) // push us into the expiry buffer window
				}
				Expect(server.ReceivedRequests()).Should(HaveLen(2))
				Expect(token.AccessToken).To(Equal("the new token"))

			})
		})
	})

	Context("when fetching token from Cache", func() {
		Context("when cached token is expired", func() {
			It("returns a new token and logs request response", func() {
				firstResponseBody := &tokenJSON{
					AccessToken: "the token",
					ExpiresIn:   3600,
				}
				secondResponseBody := &tokenJSON{
					AccessToken: "another token",
					ExpiresIn:   3600,
				}

				server.AppendHandlers(
					getOauthHandlerFunc(http.StatusOK, firstResponseBody),
					getOauthHandlerFunc(http.StatusOK, secondResponseBody),
				)

				token, err := tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(1))
				Expect(token.AccessToken).To(Equal("the token"))
				// oauth2 package translates ExpiresIn to Expiry
				clock.Increment((3600 - DefaultExpirationBufferTime) * 5 * time.Second)

				token, err = tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(2))
				Expect(token.AccessToken).To(Equal("another token"))
			})
		})

		Context("when a cached token can be used", func() {
			It("returns the cached token", func() {
				firstResponseBody := &tokenJSON{
					AccessToken: "the token",
					ExpiresIn:   3600,
				}
				secondResponseBody := &tokenJSON{
					AccessToken: "another token",
					ExpiresIn:   3600,
				}

				server.AppendHandlers(
					getOauthHandlerFunc(http.StatusOK, firstResponseBody),
					getOauthHandlerFunc(http.StatusOK, secondResponseBody),
				)

				token, err := tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(1))
				Expect(token.AccessToken).To(Equal("the token"))
				clock.Increment(3000 * time.Second)

				token, err = tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(1))
				Expect(token.AccessToken).To(Equal("the token"))
			})
		})

		Context("when forcing token refresh", func() {
			It("returns a new token", func() {
				firstResponseBody := &tokenJSON{
					AccessToken: "the token",
					ExpiresIn:   3600,
				}
				secondResponseBody := &tokenJSON{
					AccessToken: "another token",
					ExpiresIn:   3600,
				}

				server.AppendHandlers(
					getOauthHandlerFunc(http.StatusOK, firstResponseBody),
					getOauthHandlerFunc(http.StatusOK, secondResponseBody),
				)

				token, err := tokenFetcher.FetchToken(ctx, forceUpdate)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(1))
				Expect(token.AccessToken).To(Equal("the token"))

				token, err = tokenFetcher.FetchToken(ctx, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(server.ReceivedRequests()).Should(HaveLen(2))
				Expect(token.AccessToken).To(Equal("another token"))
			})
		})
	})
})
