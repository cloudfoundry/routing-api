package consuladapter

import (
	"errors"
	"net/url"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/hashicorp/consul/api"
)

func Parse(urlArg string) (string, string, error) {
	u, err := url.Parse(urlArg)
	if err != nil {
		return "", "", err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", errors.New("scheme must be http or https")
	}

	if u.Host == "" {
		return "", "", errors.New("missing address")
	}

	return u.Scheme, u.Host, nil
}

func NewClient(urlString string) (*api.Client, error) {
	scheme, address, err := Parse(urlString)
	if err != nil {
		return nil, err
	}

	return api.NewClient(&api.Config{
		Address:    address,
		Scheme:     scheme,
		HttpClient: cf_http.NewStreamingClient(),
	})
}
