package uaaclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/routing-api/config"
	uaa "github.com/cloudfoundry-community/go-uaa"
	jwt "github.com/golang-jwt/jwt/v4"
)

//go:generate counterfeiter -o fakes/uaa_client.go . Client
type Client interface {
	ValidateToken(uaaToken string, desiredPermissions ...string) error
}

func NewClient(devMode bool, cfg config.Config, logger lager.Logger) (Client, error) {
	if devMode {
		return &noOpUaaClient{}, nil
	}

	if cfg.OAuth.Port == -1 {
		err := errors.New("GoRouter requires TLS enabled to get OAuth token")
		logger.Fatal("tls-not-enabled", err, lager.Data{
			"token-endpoint": cfg.OAuth.TokenEndpoint,
			"port":           cfg.OAuth.Port,
		})
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: cfg.OAuth.SkipSSLValidation}
	if cfg.OAuth.CACerts != "" {
		certBytes, err := ioutil.ReadFile(cfg.OAuth.CACerts)
		if err != nil {
			return nil, fmt.Errorf("Failed to read ca cert file: %s", err.Error())
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
			return nil, errors.New("Unable to load caCert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := &http.Client{Transport: tr}
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	tokenURL := fmt.Sprintf("https://%s:%d", cfg.OAuth.TokenEndpoint, cfg.OAuth.Port)
	api, err := uaa.New(tokenURL, uaa.WithNoAuthentication(), uaa.WithClient(httpClient), uaa.WithSkipSSLValidation(cfg.OAuth.SkipSSLValidation))
	if err != nil {
		logger.Error("Failed to create UAA client", err)
		return nil, err
	}

	issuer, err := api.Issuer()
	if err != nil {
		logger.Error("Failed to get issuer configuration from UAA", err)
		return nil, err
	}

	logger.Info("received-issuer", lager.Data{"issuer": issuer})

	jwk, err := api.TokenKey()
	if err != nil {
		logger.Error("Failed to get verification key from UAA", err)
		return nil, err
	}

	if err := checkPublicKey(jwk.Value); err != nil {
		return nil, err
	}

	return &uaaClient{api: api, issuer: issuer, logger: logger, uaaPublicKey: jwk.Value}, nil
}

func checkPublicKey(key string) error {
	var block *pem.Block
	if block, _ = pem.Decode([]byte(key)); block == nil {
		return errors.New("Public uaa token must be PEM encoded")
	}
	return nil
}

type noOpUaaClient struct {
}

func (c *noOpUaaClient) ValidateToken(uaaToken string, desiredPermissions ...string) error {
	return nil
}

type uaaClient struct {
	api          *uaa.API
	issuer       string
	logger       lager.Logger
	uaaPublicKey string
	rwlock       sync.RWMutex
}

func (c *uaaClient) ValidateToken(uaaToken string, desiredPermissions ...string) error {
	logger := c.logger.Session("uaa-client")
	logger.Debug("decode-token-started")
	defer logger.Debug("decode-token-completed")
	var err error
	jwtToken, err := checkTokenFormat(uaaToken)
	if err != nil {
		return err
	}

	var (
		token            *jwt.Token
		uaaKey           string
		forceUaaKeyFetch bool
	)

	for i := 0; i < 2; i++ {
		uaaKey, err = c.getUaaTokenKey(logger, forceUaaKeyFetch)
		if err != nil {
			return err
		}

		token, err = jwt.Parse(jwtToken, func(t *jwt.Token) (interface{}, error) {
			if !c.isValidSigningMethod(t) {
				return nil, errors.New("invalid signing method")
			}
			if !c.isValidIssuer(t) {
				return nil, errors.New("invalid issuer")
			}

			pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(uaaKey))
			if err != nil {
				return nil, err
			}

			return pubKey, nil
		})

		if err != nil {
			logger.Error("decode-token-failed", err)
			if matchesError(err, jwt.ValidationErrorSignatureInvalid) {
				forceUaaKeyFetch = true
				continue
			}

			if matchesError(err, jwt.ValidationErrorIssuedAt) {
				logger.Info("decode-token-ignoring-issued-at-validation")
				err = nil
				break
			}
		}

		break
	}

	if err != nil {
		return err
	}

	permissions := extractPermissionsFromToken(token)
	for _, permission := range permissions {
		for _, desiredPermission := range desiredPermissions {
			if permission == desiredPermission {
				return nil
			}
		}
	}

	return errors.New("Token does not have '" + strings.Join(desiredPermissions, "', '") + "' scope")
}

func extractPermissionsFromToken(token *jwt.Token) []string {
	claims := token.Claims.(jwt.MapClaims)
	scopes := claims["scope"].([]interface{})

	var permissions []string
	for _, scope := range scopes {
		permissions = append(permissions, scope.(string))
	}

	return permissions
}

func checkTokenFormat(token string) (string, error) {
	tokenParts := strings.Split(token, " ")
	if len(tokenParts) != 2 {
		return "", errors.New("Invalid token format")
	}

	tokenType, userToken := tokenParts[0], tokenParts[1]
	if !strings.EqualFold(tokenType, "bearer") {
		return "", errors.New("Invalid token type: " + tokenType)
	}

	return userToken, nil
}

func matchesError(err error, errorType uint32) bool {
	if validationError, ok := err.(*jwt.ValidationError); ok {
		return validationError.Errors&errorType == errorType
	}
	return false
}

func (c *uaaClient) getUaaTokenKey(logger lager.Logger, forceFetch bool) (string, error) {
	if c.getUaaPublicKey() == "" || forceFetch {
		logger.Debug("fetching-new-uaa-key")
		key, err := c.api.TokenKey()
		if err != nil {
			return "", err
		}

		if err = checkPublicKey(key.Value); err != nil {
			return "", err
		}
		logger.Info("fetch-key-successful")

		if c.getUaaPublicKey() == key.Value {
			logger.Debug("Fetched the same verification key from UAA")
		} else {
			logger.Debug("Fetched a different verification key from UAA")
		}
		c.rwlock.Lock()
		defer c.rwlock.Unlock()
		c.uaaPublicKey = key.Value

		return key.Value, nil
	}

	return c.getUaaPublicKey(), nil
}

func (c *uaaClient) getUaaPublicKey() string {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()
	return c.uaaPublicKey
}

func (c *uaaClient) isValidIssuer(token *jwt.Token) bool {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims.VerifyIssuer(c.issuer, true)
	}
	return false
}

func (u *uaaClient) isValidSigningMethod(token *jwt.Token) bool {
	switch token.Method {
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		return true
	default:
		return false
	}
}
