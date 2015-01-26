package authentication

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter -o fakes/fake_token.go . Token
type Token interface {
	DecodeToken(userToken string) error
}

type accessToken struct {
	uaaPublicKey string `json:"value"`
	logger       lager.Logger
}

func NewAccessToken(uaaPublicKey string, logger lager.Logger) accessToken {
	return accessToken{
		uaaPublicKey: uaaPublicKey,
		logger:       logger,
	}
}

type p struct {
	uaaPublicKey string `json:"value"`
}

func (accessToken accessToken) DecodeToken(userToken string) error {
	token, err := jwt.Parse(userToken, func(t *jwt.Token) (interface{}, error) {
		return []byte(accessToken.uaaPublicKey), nil
	})

	if err != nil {
		accessToken.logger.Error("error", err)
		return err
	}

	hasPermission := false
	permissions := token.Claims["scope"]

	a := permissions.([]interface{})

	for _, permission := range a {
		if permission.(string) == "route.advertise" {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return errors.New("route.advertise permissions missing")
	}

	return nil
}
