package authentication

import (
	"errors"

	"github.com/dgrijalva/jwt-go"
)

//go:generate counterfeiter -o fakes/fake_token.go . Token
type Token interface {
	DecodeToken(userToken string) error
}

type accessToken struct {
	uaaPublicKey string `json:"value"`
}

func NewAccessToken(uaaPublicKey string) accessToken {
	return accessToken{
		uaaPublicKey: uaaPublicKey,
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
		err = errors.New("Token does not have 'route.advertise' scope")
		return err
	}

	return nil
}
