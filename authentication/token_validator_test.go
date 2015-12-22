package authentication_test

import (
	"errors"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/routing-api/authentication"
	"github.com/cloudfoundry-incubator/routing-api/authentication/fakes"
	"github.com/dgrijalva/jwt-go"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token", func() {
	const (
		validUaaPEMKey   = "-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d\nKVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX\nqHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug\nspULZVNRxq7veq/fzwIDAQAB\n-----END PUBLIC KEY-----"
		invalidUaaPEMKey = "-----BEGIN PUBLIC KEY-----\nMIHfMA0SCSqGSIb3EQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d\nKVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX\nqHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug\nspULZVNRxq7veq/fzwIDAQAB\n-----END PUBLIC KEY-----"
	)
	var (
		accessTokenValidator authentication.TokenValidator
		fakeSigningMethod    *fakes.FakeSigningMethod
		fakeUaaKeyFetcher    *fakes.FakeUaaKeyFetcher
		signedKey            string
		UserPrivateKey       string
		UAAPublicKey         string
		logger               lager.Logger

		token *jwt.Token
		err   error
	)

	verifyErrorType := func(err error, errorType uint32, message string) {
		validationError, ok := err.(*jwt.ValidationError)
		Expect(ok).To(BeTrue())
		Expect(validationError.Errors & errorType).To(Equal(errorType))
		Expect(err.Error()).To(Equal(message))
	}

	BeforeEach(func() {
		UserPrivateKey = "UserPrivateKey"
		UAAPublicKey = "UAAPublicKey"
		logger = lagertest.NewTestLogger("test")

		fakeSigningMethod = &fakes.FakeSigningMethod{}
		fakeSigningMethod.AlgStub = func() string {
			return "FAST"
		}
		fakeSigningMethod.SignStub = func(signingString string, key interface{}) (string, error) {
			signature := jwt.EncodeSegment([]byte(signingString + "SUPERFAST"))
			return signature, nil
		}
		fakeSigningMethod.VerifyStub = func(signingString, signature string, key interface{}) (err error) {
			if signature != jwt.EncodeSegment([]byte(signingString+"SUPERFAST")) {
				return errors.New("Signature is invalid")
			}

			return nil
		}

		jwt.RegisterSigningMethod("FAST", func() jwt.SigningMethod {
			return fakeSigningMethod
		})

		header := map[string]interface{}{
			"alg": "FAST",
		}

		alg := "FAST"
		signingMethod := jwt.GetSigningMethod(alg)
		token = jwt.New(signingMethod)
		token.Header = header

		fakeUaaKeyFetcher = &fakes.FakeUaaKeyFetcher{}
		accessTokenValidator = authentication.NewAccessTokenValidator(logger, UAAPublicKey, fakeUaaKeyFetcher)
	})

	Describe(".DecodeToken", func() {
		Context("when the token is valid", func() {
			BeforeEach(func() {
				claims := map[string]interface{}{
					"exp":   3404281214,
					"scope": []string{"route.advertise"},
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())

				signedKey = "bearer " + signedKey
			})

			It("does not return an error", func() {
				err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when a token is not valid", func() {
			It("returns an error if the user token is not signed", func() {
				err = accessTokenValidator.DecodeToken("bearer not-a-signed-key", "not a permission")
				Expect(err).To(HaveOccurred())
				verifyErrorType(err, jwt.ValidationErrorMalformed, "token contains an invalid number of segments")
				Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(0))
			})

			It("returns an invalid token format when there is no token type", func() {
				err = accessTokenValidator.DecodeToken("has-no-token-type", "not a permission")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Invalid token format"))
				Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(0))
			})

			It("returns an invalid token type when type is not bearer", func() {
				err = accessTokenValidator.DecodeToken("basic some-auth", "not a permission")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Invalid token type: basic"))
				Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(0))
			})
		})

		Context("when signature is invalid", func() {
			BeforeEach(func() {
				fakeSigningMethod.VerifyReturns(errors.New("invalid signature"))
				fakeUaaKeyFetcher.FetchKeyReturns(validUaaPEMKey, nil)
				claims := map[string]interface{}{
					"exp":   3404281214,
					"scope": []string{"route.advertise"},
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())
				signedKey = "bearer " + signedKey
			})

			Context("uaa returns a verification key", func() {
				It("refreshes the key and returns an invalid signature error", func() {
					err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
					Expect(err).To(HaveOccurred())
					Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(1))
					verifyErrorType(err, jwt.ValidationErrorSignatureInvalid, "invalid signature")
				})
			})

			Context("when uaa returns an error", func() {
				BeforeEach(func() {
					fakeUaaKeyFetcher.FetchKeyReturns("", errors.New("booom"))
				})

				It("tries to refresh key and returns the uaa error", func() {
					err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("booom"))
					Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(1))
				})
			})
		})

		Context("when verification key needs to be refreshed to validate the signature", func() {
			BeforeEach(func() {
				fakeSigningMethod.VerifyStub = func(signingString string, signature string, key interface{}) error {
					switch k := key.(type) {
					case []byte:
						if string(k) == validUaaPEMKey {
							return nil
						}
						return errors.New("invalid signature")
					default:
						return errors.New("invalid signature")
					}
				}

				fakeUaaKeyFetcher.FetchKeyReturns(validUaaPEMKey, nil)
				claims := map[string]interface{}{
					"exp":   3404281214,
					"scope": []string{"route.advertise"},
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())
				signedKey = "bearer " + signedKey
			})

			It("fetches new key and then validates the token", func() {
				err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(1))
			})

			Context("with multiple concurrent clients", func() {
				Context("when new key applies to all clients", func() {
					It("fetches new key and then validates the token", func() {
						wg := sync.WaitGroup{}
						for i := 0; i < 2; i++ {
							wg.Add(1)
							go func(wg *sync.WaitGroup) {
								defer GinkgoRecover()
								defer wg.Done()
								err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
								Expect(err).NotTo(HaveOccurred())
							}(&wg)
						}
						wg.Wait()
						Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(BeNumerically(">=", 1))
					})
				})

				Context("when new key applies to only one client and not others", func() {
					var (
						keyChannel    chan string
						resultChannel chan bool
					)

					BeforeEach(func() {
						keyChannel = make(chan string)
						resultChannel = make(chan bool)
						fakeUaaKeyFetcher.FetchKeyStub = func() (string, error) {
							key := <-keyChannel
							return key, nil
						}
					})

					AfterEach(func() {
						close(keyChannel)
						close(resultChannel)
					})

					It("fetches new key and validates the token", func() {
						wg := sync.WaitGroup{}
						for i := 0; i < 2; i++ {
							wg.Add(1)
							go func(wg *sync.WaitGroup) {
								defer GinkgoRecover()
								defer wg.Done()
								err := accessTokenValidator.DecodeToken(signedKey, "route.advertise")
								select {
								case fail := <-resultChannel:
									if fail {
										Expect(err).To(HaveOccurred())
										verifyErrorType(err, jwt.ValidationErrorSignatureInvalid, "invalid signature")
									} else {
										Expect(err).NotTo(HaveOccurred())
									}
								}
							}(&wg)
						}
						keyChannel <- invalidUaaPEMKey
						resultChannel <- true
						keyChannel <- validUaaPEMKey
						resultChannel <- false
						wg.Wait()
						Expect(fakeUaaKeyFetcher.FetchKeyCallCount()).To(Equal(2))
					})
				})
			})
		})

		Context("expired time", func() {
			BeforeEach(func() {
				claims := map[string]interface{}{
					"exp": time.Now().Unix() - 5,
				}
				token.Claims = claims

				signedKey, err = token.SignedString([]byte(UserPrivateKey))
				Expect(err).NotTo(HaveOccurred())

				signedKey = "bearer " + signedKey
			})

			It("returns an error if the token is expired", func() {
				err = accessTokenValidator.DecodeToken(signedKey, "route.advertise")
				Expect(err).To(HaveOccurred())
				verifyErrorType(err, jwt.ValidationErrorExpired, "token is expired")
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

				signedKey = "bearer " + signedKey
			})

			It("returns an error if the the user does not have requested permissions", func() {
				err = accessTokenValidator.DecodeToken(signedKey, "route.my-permissions", "some.other.scope")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Token does not have 'route.my-permissions', 'some.other.scope' scope"))
			})
		})

	})

	Describe(".CheckPublicToken", func() {
		BeforeEach(func() {
			accessTokenValidator = authentication.NewAccessTokenValidator(logger, "not a valid pem string", fakeUaaKeyFetcher)
		})

		It("returns an error if the public token is malformed", func() {
			err = accessTokenValidator.CheckPublicToken()
			Expect(err).To(HaveOccurred())
		})
	})
})
