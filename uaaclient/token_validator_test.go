package uaaclient_test

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/routing-api/uaaclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

const (
	pemHeaderPrivateKey = "RSA PRIVATE KEY"
	pemHeaderPublicKey  = "PUBLIC KEY"
)

var tokenEncoding = base64.RawURLEncoding

var _ = Describe("UaaClient", func() {
	var (
		publicKeyPEM   []byte
		privateKey     *rsa.PrivateKey
		cfg            uaaclient.Config
		server         *ghttp.Server
		serverCertFile *os.File
		logger         *lagertest.TestLogger
	)

	BeforeEach(func() {
		var err error
		var publicKey *rsa.PublicKey

		privateKey, publicKey, err = generateRSAKeyPair()
		Expect(err).NotTo(HaveOccurred())

		publicKeyPEM, err = publicKeyToPEM(publicKey)
		Expect(err).NotTo(HaveOccurred())

		var uaaResponseStruct = struct {
			Alg   string `json:"alg"`
			Value string `json:"value"`
		}{"alg", string(publicKeyPEM)}
		server = ghttp.NewTLSServer()
		serverCertFile, err = ioutil.TempFile("", "routing-api-uaa-client-test")
		Expect(err).NotTo(HaveOccurred())

		certPem := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: server.HTTPTestServer.Certificate().Raw,
		})
		_, err = serverCertFile.Write(certPem)
		Expect(err).NotTo(HaveOccurred())

		url, err := url.Parse(server.URL())
		Expect(err).ToNot(HaveOccurred())

		hostParts := strings.Split(url.Host, ":")
		Expect(hostParts).To(HaveLen(2))
		port, err := strconv.Atoi(hostParts[1])
		Expect(err).NotTo(HaveOccurred())

		cfg = uaaclient.Config{
			SkipSSLValidation: true,
			TokenEndpoint:     hostParts[0],
			Port:              port,
			ClientName:        "client-name",
			ClientSecret:      "client-secret",
		}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", OpenIDConfigEndpoint),
				ghttp.RespondWith(http.StatusOK, "{\"issuer\":\"https://uaa.domain.com\"}"),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", TokenKeyEndpoint),
				ghttp.RespondWithJSONEncoded(
					http.StatusOK,
					uaaResponseStruct,
				),
			),
		)
		logger = lagertest.NewTestLogger("test")
	})

	AfterEach(func() {
		err := os.Remove(serverCertFile.Name())
		Expect(err).NotTo(HaveOccurred())
		server.Close()
	})

	Describe("NewTokenValidator", func() {
		It("returns a uaa client", func() {
			client, err := uaaclient.NewTokenValidator(false, cfg, logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(client).NotTo(BeNil())
			Expect(reflect.TypeOf(client).String()).To(Equal("*uaaclient.tokenValidator"))
		})

		Context("in dev mode", func() {
			It("returns a noOpUaaClient", func() {
				client, err := uaaclient.NewTokenValidator(true, cfg, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(client).NotTo(BeNil())
				Expect(reflect.TypeOf(client).String()).To(Equal("*uaaclient.noOpTokenValidator"))
			})
		})

		Context("when OAuth port is -1", func() {
			BeforeEach(func() {
				cfg.Port = -1
			})

			It("returns an error that UAA client requires TLS", func() {
				_, err := uaaclient.NewTokenValidator(false, cfg, logger)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("tls-not-enabled: UAA client requires TLS enabled"))
			})
		})

		Context("when the OAuth config includes CA certs", func() {
			BeforeEach(func() {
				cfg.CACerts = serverCertFile.Name()
				cfg.SkipSSLValidation = false
			})

			It("uses certificates to validate the server", func() {
				client, err := uaaclient.NewTokenValidator(false, cfg, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(server.ReceivedRequests()).To(HaveLen(2))
			})

			Context("when there is an error reading the cert file", func() {
				BeforeEach(func() {
					cfg.CACerts = "non-existing-cert-file"
				})

				It("returns an error", func() {
					client, err := uaaclient.NewTokenValidator(false, cfg, logger)
					Expect(client).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Failed to read ca cert file"))
				})
			})

			Context("when there is an error parsing the PEM file", func() {
				var corruptedCertFile *os.File

				BeforeEach(func() {
					var err error
					corruptedCertFile, err = ioutil.TempFile("", "routing-api-uaa-client-test")
					Expect(err).NotTo(HaveOccurred())

					_, err = corruptedCertFile.Write([]byte("definitely-not-a-pem"))
					Expect(err).NotTo(HaveOccurred())

					cfg.CACerts = corruptedCertFile.Name()
				})

				AfterEach(func() {
					err := os.Remove(corruptedCertFile.Name())
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					client, err := uaaclient.NewTokenValidator(false, cfg, logger)
					Expect(client).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Unable to load caCert"))
				})
			})
		})

		Context("when it fails to get the issuer", func() {
			BeforeEach(func() {
				server.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", OpenIDConfigEndpoint),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("returns an error", func() {
				client, err := uaaclient.NewTokenValidator(false, cfg, logger)
				Expect(client).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`An error occurred while calling https://.*/\.well-known/openid-configuration`)))
				Expect(logger).To(gbytes.Say("Failed to get issuer"))
			})
		})

		Context("when it fails to get the token", func() {
			BeforeEach(func() {
				server.SetHandler(1,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", TokenKeyEndpoint),
						ghttp.RespondWith(http.StatusInternalServerError, ""),
					),
				)
			})

			It("returns an error", func() {
				client, err := uaaclient.NewTokenValidator(false, cfg, logger)
				Expect(client).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(MatchRegexp(`An error occurred while calling https://.*/token_key`)))
				Expect(logger).To(gbytes.Say("Failed to get verification key"))
			})
		})

		Context("when received token is not valid", func() {
			BeforeEach(func() {
				var uaaResponseStruct = struct {
					Alg   string `json:"alg"`
					Value string `json:"value"`
				}{"alg", "invalid"}
				server.SetHandler(1,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", TokenKeyEndpoint),
						ghttp.RespondWithJSONEncoded(http.StatusOK, uaaResponseStruct),
					),
				)
			})

			It("returns an error", func() {
				client, err := uaaclient.NewTokenValidator(false, cfg, logger)
				Expect(client).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Public uaa token must be PEM encoded"))
			})
		})
	})

	Describe("ValidateToken", func() {
		var (
			uaaClient uaaclient.TokenValidator
		)

		BeforeEach(func() {
			var err error
			uaaClient, err = uaaclient.NewTokenValidator(false, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(uaaClient).NotTo(BeNil())
		})

		It("returns nil, indicating that the token is valid and includes one of the desired permissions", func() {
			validToken, err := makeValidToken(privateKey)
			Expect(err).NotTo(HaveOccurred())

			err = uaaClient.ValidateToken(validToken, "some.scope")
			Expect(err).ToNot(HaveOccurred())

			err = uaaClient.ValidateToken(validToken, "another.scope", "some.scope")
			Expect(err).ToNot(HaveOccurred())

			validMultiscopeToken, err := makeValidMultiscopeToken(privateKey)
			Expect(err).NotTo(HaveOccurred())
			err = uaaClient.ValidateToken(validMultiscopeToken, "some.scope")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the token does not contain desired permissions", func() {
			It("fails", func() {
				validToken, err := makeValidToken(privateKey)
				Expect(err).NotTo(HaveOccurred())

				err = uaaClient.ValidateToken(validToken, "another.scope")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Token does not have 'another.scope' scope"))
			})
		})

		Context("when passed token has invalid format", func() {
			It("fails", func() {
				err := uaaClient.ValidateToken("invalid", "some.scope")
				Expect(err).To(MatchError("Invalid token format"))
			})
		})

		Context("when passed token type is not bearer", func() {
			It("fails", func() {
				err := uaaClient.ValidateToken("invalid token", "some.scope")
				Expect(err).To(MatchError("Invalid token type: invalid"))
			})
		})

		Context("when the token algorithm doesn't match the key type", func() {
			It("fails", func() {
				spoofedToken, err := makeSpoofedToken(publicKeyPEM)
				Expect(err).NotTo(HaveOccurred())

				err = uaaClient.ValidateToken(spoofedToken, "some.scope")
				Expect(err).To(MatchError("invalid signing method"))
			})
		})

		Context("when the token's issuer does not match the issuer saved on the UAA client", func() {
			It("fails", func() {
				otherIssuerToken, err := makeInvalidIssuerToken(privateKey)
				Expect(err).NotTo(HaveOccurred())

				err = uaaClient.ValidateToken(otherIssuerToken, "some.scope")
				Expect(err).To(MatchError("invalid issuer"))
			})
		})

		Context("when the token has an invalid Issued At date", func() {
			It("ignores the error and continues verifying the token", func() {
				invalidIssuedAtToken, err := makeInvalidIssuedAtToken(privateKey)
				Expect(err).NotTo(HaveOccurred())

				err = uaaClient.ValidateToken(invalidIssuedAtToken, "some.scope")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the token has been signed by a newer UAA token key than the one stored on the client", func() {
			var (
				newPrivateKey *rsa.PrivateKey
				newPublicKey  *rsa.PublicKey
			)

			BeforeEach(func() {
				var err error
				newPrivateKey, newPublicKey, err = generateRSAKeyPair()
				Expect(err).NotTo(HaveOccurred())
			})

			Context("on the first try", func() {
				BeforeEach(func() {
					newPublicKeyPEM, err := publicKeyToPEM(newPublicKey)
					Expect(err).NotTo(HaveOccurred())

					var uaaResponseStruct = struct {
						Alg   string `json:"alg"`
						Value string `json:"value"`
					}{"RS256", string(newPublicKeyPEM)}

					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TokenKeyEndpoint),
							ghttp.RespondWithJSONEncoded(
								http.StatusOK,
								uaaResponseStruct,
							),
						),
					)
				})

				It("refetches the UAA token key and makes another attempt to parse the token", func() {
					validToken, err := makeValidToken(newPrivateKey)
					Expect(err).NotTo(HaveOccurred())

					err = uaaClient.ValidateToken(validToken, "some.scope")
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).To(HaveLen(3))
				})
			})

			Context("when it keeps getting invalid token key", func() {
				BeforeEach(func() {
					var uaaResponseStruct = struct {
						Alg   string `json:"alg"`
						Value string `json:"value"`
					}{"RS256", string(publicKeyPEM)}

					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TokenKeyEndpoint),
							ghttp.RespondWithJSONEncoded(
								http.StatusOK,
								uaaResponseStruct,
							),
						),
					)
				})

				It("fails", func() {
					validToken, err := makeValidToken(newPrivateKey)
					Expect(err).NotTo(HaveOccurred())

					err = uaaClient.ValidateToken(validToken, "some.scope")
					Expect(err).To(HaveOccurred())
					Expect(server.ReceivedRequests()).To(HaveLen(3))
				})
			})
		})
	})
})

func generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}
	public := private.Public().(*rsa.PublicKey)
	return private, public, nil
}

// PublicKeyToPEM serializes an RSA Public key into PEM format.
func publicKeyToPEM(publicKey *rsa.PublicKey) ([]byte, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return []byte{}, err
	}

	return encodePEM(keyBytes, pemHeaderPublicKey), nil
}

func encodePEM(keyBytes []byte, keyType string) []byte {
	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	return pem.EncodeToMemory(block)
}

func jwtHeader(alg, kid string) string {
	return fmt.Sprintf(`{ "alg": "%s", "kid": "%s", "typ": "JWT" }`, alg, kid)
}

func makeValidToken(privateKey *rsa.PrivateKey) (string, error) {
	header := jwtHeader("RS256", "some-key-id")
	signingString := fmt.Sprintf("%s.%s",
		tokenEncoding.EncodeToString([]byte(header)),
		tokenEncoding.EncodeToString([]byte(TokenPayload)),
	)
	signature, err := signWithRS256(signingString, privateKey)
	if err != nil {
		return "", err
	}
	fullToken := fmt.Sprintf("bearer %s.%s", signingString, signature)
	return fullToken, nil
}

func makeInvalidIssuerToken(privateKey *rsa.PrivateKey) (string, error) {
	tokenPayload := `{
	  "scope": [
		"some.scope"
	  ],
	  "iat": 1481253086,
	  "exp": 2491253686,
	  "iss": "https://other.issuer"
	}`
	header := jwtHeader("RS256", "some-key-id")
	signingString := fmt.Sprintf("%s.%s",
		tokenEncoding.EncodeToString([]byte(header)),
		tokenEncoding.EncodeToString([]byte(tokenPayload)),
	)
	signature, err := signWithRS256(signingString, privateKey)
	if err != nil {
		return "", err
	}
	fullToken := fmt.Sprintf("bearer %s.%s", signingString, signature)
	return fullToken, nil
}

func makeValidMultiscopeToken(privateKey *rsa.PrivateKey) (string, error) {
	tokenPayload := `{
	  "scope": [
		"another.scope",
		"some.scope"
	  ],
	  "iat": 1481253086,
	  "exp": 2491253686,
	  "iss": "https://uaa.domain.com"
	}`
	header := jwtHeader("RS256", "some-key-id")
	signingString := fmt.Sprintf("%s.%s",
		tokenEncoding.EncodeToString([]byte(header)),
		tokenEncoding.EncodeToString([]byte(tokenPayload)),
	)
	signature, err := signWithRS256(signingString, privateKey)
	if err != nil {
		return "", err
	}
	fullToken := fmt.Sprintf("bearer %s.%s", signingString, signature)
	return fullToken, nil
}

func makeInvalidIssuedAtToken(privateKey *rsa.PrivateKey) (string, error) {
	invalidIssuedAtTime := time.Now().Add(24 * time.Hour)
	tokenPayload := fmt.Sprintf(`{
	  "scope": [
		"some.scope"
	  ],
	  "iat": %d,
	  "exp": 2491253686,
	  "iss": "https://uaa.domain.com"
	}`, invalidIssuedAtTime.Unix())
	header := jwtHeader("RS256", "some-key-id")
	signingString := fmt.Sprintf("%s.%s",
		tokenEncoding.EncodeToString([]byte(header)),
		tokenEncoding.EncodeToString([]byte(tokenPayload)),
	)
	signature, err := signWithRS256(signingString, privateKey)
	if err != nil {
		return "", err
	}
	fullToken := fmt.Sprintf("bearer %s.%s", signingString, signature)
	return fullToken, nil
}

func makeSpoofedToken(publicKeyPEM []byte) (string, error) {
	header := jwtHeader("HS256", "some-key-id")
	signingString := fmt.Sprintf("%s.%s",
		tokenEncoding.EncodeToString([]byte(header)),
		tokenEncoding.EncodeToString([]byte(TokenPayload)),
	)
	signature := signWithHS256(signingString, string(publicKeyPEM))
	fullToken := fmt.Sprintf("bearer %s.%s", signingString, signature)
	return fullToken, nil
}

func signWithHS256(signingString string, key string) string {
	hasher := hmac.New(sha256.New, []byte(key))
	hasher.Write([]byte(signingString))
	return tokenEncoding.EncodeToString(hasher.Sum(nil))
}

func signWithRS256(signingString string, privateKey *rsa.PrivateKey) (string, error) {
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(signingString))

	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hasher.Sum(nil))
	if err != nil {
		return "", err
	}
	return tokenEncoding.EncodeToString(sigBytes), nil
}
