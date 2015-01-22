package authentication

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/pivotal-golang/lager"
)

type Authentication interface {
	GetVerificationToken() *AccessToken // No
	DecodeToken()
	CheckScope()
	CheckExperation()
}

type UAA struct {
	CCApi  CloudControllerApi
	logger lager.Logger
}

type AccessToken struct {
	Value string `json:"value"`
}

func NewUAA(CCApi CloudControllerApi, logger lager.Logger) UAA {
	return UAA{
		CCApi:  CCApi,
		logger: logger,
	}
}

func (uaa UAA) GetVerificationToken() *AccessToken {
	info := uaa.CCApi.GetInfo()
	if info == nil {
		return nil
	}

	url := info.TokenEndpoint + "/token_key"

	client := http.Client{}

	resp, _ := client.Get(url)
	if resp.StatusCode == 404 {
		uaa.logger.Error("error", errors.New("UAA could not be reached"))
		return nil
	}

	body, _ := ioutil.ReadAll(resp.Body)

	accessToken := &AccessToken{}
	json.Unmarshal(body, accessToken)

	return accessToken
}
