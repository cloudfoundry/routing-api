package authentication

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pivotal-golang/lager"
)

//go:generate  counterfeiter -o fakes/fake_cloud_controller_api.go . CloudControllerApi
type CloudControllerApi interface {
	GetInfo() *Info
}

type Info struct {
	TokenEndpoint string `json:"token_endpoint"`
}

type CCApi struct {
	CCEndpoint string
	logger     lager.Logger
}

func NewCCApi(CCEndpoint string, logger lager.Logger) CCApi {
	return CCApi{
		CCEndpoint: CCEndpoint,
		logger:     logger,
	}
}

func (cc CCApi) GetInfo() *Info {
	url := cc.CCEndpoint + "/v2/info"

	client := http.Client{}

	resp, _ := client.Get(url)
	if resp.StatusCode == 404 {
		cc.logger.Error("Error", errors.New("Cloud Controller could not be reached"))
		return nil
	}

	body, _ := ioutil.ReadAll(resp.Body)

	var info *Info
	json.Unmarshal(body, &info)

	info.TokenEndpoint = strings.Replace(info.TokenEndpoint, "https", "http", 1)

	return info
}
