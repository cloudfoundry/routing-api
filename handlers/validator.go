package handlers

import (
	"fmt"
	"net/url"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
)

//go:generate counterfeiter -o fakes/fake_validator.go . RouteValidator
type RouteValidator interface {
	ValidateCreate(routes []db.Route, maxTTL int) *routing_api.Error
	ValidateDelete(routes []db.Route) *routing_api.Error
}

type Validator struct{}

func NewValidator() Validator {
	return Validator{}
}

func (v Validator) ValidateCreate(routes []db.Route, maxTTL int) *routing_api.Error {
	for _, route := range routes {
		err := requiredValidation(route)
		if err != nil {
			return err
		}

		if route.TTL > maxTTL {
			return &routing_api.Error{routing_api.RouteInvalidError, fmt.Sprintf("Max ttl is %d", maxTTL)}
		}

		if route.TTL <= 0 {
			return &routing_api.Error{routing_api.RouteInvalidError, "Request requires a ttl greater than 0"}
		}
	}
	return nil
}

func (v Validator) ValidateDelete(routes []db.Route) *routing_api.Error {
	for _, route := range routes {
		err := requiredValidation(route)
		if err != nil {
			return err
		}
	}
	return nil
}

func requiredValidation(route db.Route) *routing_api.Error {
	err := validateRouteParses(route.Route)
	if err != nil {
		return err
	}

	if route.Port <= 0 {
		return &routing_api.Error{routing_api.RouteInvalidError, "Each route request requires a port greater than 0"}
	}

	if route.Route == "" {
		return &routing_api.Error{routing_api.RouteInvalidError, "Each route request requires a valid route"}
	}

	if route.IP == "" {
		return &routing_api.Error{routing_api.RouteInvalidError, "Each route request requires an IP"}
	}

	return nil
}

func validateRouteParses(route string) *routing_api.Error {
	parsedURL, err := url.Parse(route)

	if err != nil {
		return &routing_api.Error{routing_api.RouteInvalidError, err.Error()}
	}

	if parsedURL.String() != route {
		return &routing_api.Error{routing_api.RouteInvalidError, "Route cannot contain invalid characters"}
	}

	return nil
}
