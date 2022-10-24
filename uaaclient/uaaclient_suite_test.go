package uaaclient_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	OpenIDConfigEndpoint        = "/.well-known/openid-configuration"
	TokenKeyEndpoint            = "/token_key"
	DefaultMaxNumberOfRetries   = 3
	DefaultRetryInterval        = 15 * time.Second
	DefaultRequestTimeout       = 1 * time.Second
	DefaultExpirationBufferTime = 30
	TokenPayload                = `{
	  "scope": [
		"some.scope"
	  ],
	  "iat": 1481253086,
	  "exp": 2491253686,
	  "iss": "https://uaa.domain.com"
	}`
)

func TestUaaclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Uaaclient Suite")
}
