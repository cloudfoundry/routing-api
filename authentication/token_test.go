package authentication_test

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pivotal-cf-experimental/routing-api/authentication"
	"github.com/pivotal-cf-experimental/routing-api/authentication/fakes"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	var (
		accessToken authentication.Token
		logger      *lagertest.TestLogger

		signedKey      string
		UserPrivateKey string
		UAAPublicKey   string

		token *jwt.Token
		err   error
	)

	BeforeEach(func() {
		UserPrivateKey = "UserPrivateKey"
		UAAPublicKey = "UAAPublicKey"
		logger = lagertest.NewTestLogger("routing-api-test")

		fakes.RegisterFastTokenSigningMethod()

		header := map[string]interface{}{
			"alg": "FAST",
		}

		alg := "FAST"
		signingMethod := jwt.GetSigningMethod(alg)
		token = jwt.New(signingMethod)
		token.Header = header

		accessToken = authentication.NewAccessToken(UAAPublicKey, logger)
	})

	Describe(".DecodeToken", func() {
		Context("when the token is valid", func() {
			BeforeEach(func() {
				claims := map[string]interface{}{
					// "jti":       "c5f6a266-5cf0-4ae2-9647-2615e7d28fa1",
					// "client_id": "mister-client",
					// "cid":       "mister-client",
					"exp":   3404281214,
					"scope": []string{"route.advertise"},
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not return an error", func() {
				err := accessToken.DecodeToken(signedKey)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the token is not valid", func() {
			BeforeEach(func() {
				err = accessToken.DecodeToken("not a signed key")
			})

			It("returns an error if the token is malformed", func() {
				Expect(err).To(HaveOccurred())
			})

			It("logs the error", func() {
				Expect(logger.Logs()[0].Message).To(ContainSubstring("error"))
				Expect(logger.Logs()[0].Data["error"]).To(ContainSubstring(err.Error()))
			})
		})

		Context("expired time", func() {
			BeforeEach(func() {
				claims := map[string]interface{}{
					"exp": time.Now().Unix() - 5,
					// "exp": time.Now(),
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error if the token is expired", func() {
				err = accessToken.DecodeToken(signedKey)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("token is expired"))
			})
		})

		Context("permissions", func() {
			BeforeEach(func() {
				claims := map[string]interface{}{
					"exp":   time.Now().Unix() + 50000000,
					"scope": []string{"route.foo"},
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error if the the user does not have route.advertise permissions", func() {
				err = accessToken.DecodeToken(signedKey)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("route.advertise permissions missing"))
			})
		})

	})
})
