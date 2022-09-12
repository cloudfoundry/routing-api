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
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/routing-api/config"
	"code.cloudfoundry.org/routing-api/uaaclient"

	. "github.com/onsi/ginkgo"
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
		publicKeyPEM []byte
		privateKey   *rsa.PrivateKey
		cfg          config.Config
		server       *ghttp.Server
		logger       *lagertest.TestLogger
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

		url, err := url.Parse(server.URL())
		Expect(err).ToNot(HaveOccurred())

		hostParts := strings.Split(url.Host, ":")
		Expect(hostParts).To(HaveLen(2))
		port, err := strconv.Atoi(hostParts[1])
		Expect(err).NotTo(HaveOccurred())

		cfg = config.Config{
			OAuth: config.OAuthConfig{
				SkipSSLValidation: true,
				TokenEndpoint:     hostParts[0],
				Port:              port,
				ClientName:        "client-name",
				ClientSecret:      "client-secret",
			},
		}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", OpenIDConfigEndpoint),
				ghttp.RespondWith(http.StatusOK, fmt.Sprintf("{\"issuer\":\"https://uaa.domain.com\"}")),
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

	Describe("NewClient", func() {
		Context("in dev mode", func() {
			It("returns a noOpUaaClient", func() {
				client, err := uaaclient.NewClient(true, cfg, nil)
				Expect(err).NotTo(HaveOccurred())

				Expect(client).NotTo(BeNil())
				Expect(reflect.TypeOf(client).String()).To(Equal("*uaaclient.noOpUaaClient"))
			})
		})

		Context("when OAuth port is -1", func() {
			BeforeEach(func() {
				cfg.OAuth.Port = -1
			})

			It("logs a fatal message that TLS is required in order to get an OAuth token", func() {
				Expect(func() {
					uaaclient.NewClient(false, cfg, logger)
				}).Should(Panic())
				Expect(logger).To(gbytes.Say("tls-not-enabled"))
			})
		})
	})

	Describe("DecodeToken", func() {
		var (
			uaaClient uaaclient.Client
		)

		BeforeEach(func() {
			var err error
			uaaClient, err = uaaclient.NewClient(false, cfg, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(uaaClient).NotTo(BeNil())
		})

		AfterEach(func() {
			server.Close()
		})

		Context("when the token has been signed with the correct private key", func() {
			It("succeeds", func() {
				validToken, err := makeValidToken(privateKey)
				Expect(err).NotTo(HaveOccurred())
				err = uaaClient.DecodeToken(validToken, "some.scope")
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the token algorithm doesn't match the key type", func() {
			It("fails", func() {
				spoofedToken, err := makeSpoofedToken(publicKeyPEM)
				Expect(err).NotTo(HaveOccurred())
				err = uaaClient.DecodeToken(spoofedToken, "some.scope")
				Expect(err).To(MatchError("invalid signing method"))
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
