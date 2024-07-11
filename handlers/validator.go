package handlers

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/models"
)

//go:generate counterfeiter -o fakes/fake_validator.go . RouteValidator
type RouteValidator interface {
	ValidateCreate(routes []models.Route, maxTTL int) *routing_api.Error
	ValidateDelete(routes []models.Route) *routing_api.Error

	ValidateCreateTcpRouteMapping(tcpRouteMappings []models.TcpRouteMapping, routerGroups models.RouterGroups, maxTTL int) *routing_api.Error
	ValidateDeleteTcpRouteMapping(tcpRouteMappings []models.TcpRouteMapping) *routing_api.Error
}

type Validator struct{}

func NewValidator() Validator {
	return Validator{}
}

func (v Validator) ValidateCreate(routes []models.Route, maxTTL int) *routing_api.Error {
	for _, route := range routes {
		err := requiredValidation(route)
		if err != nil {
			return err
		}

		if *route.TTL > maxTTL {
			err := routing_api.NewError(routing_api.RouteInvalidError, fmt.Sprintf("Max ttl is %d", maxTTL))
			return &err
		}

		if *route.TTL <= 0 {
			err := routing_api.NewError(routing_api.RouteInvalidError, "Request requires a ttl greater than 0")
			return &err
		}
	}
	return nil
}

func (v Validator) ValidateDelete(routes []models.Route) *routing_api.Error {
	for _, route := range routes {
		err := requiredValidation(route)
		if err != nil {
			return err
		}
	}
	return nil
}

func requiredValidation(route models.Route) *routing_api.Error {
	err := validateRouteUrl(route.Route)
	if err != nil {
		return err
	}

	err = validateRouteServiceUrl(route.RouteServiceUrl)
	if err != nil {
		return err
	}

	if route.Port <= 0 {
		err := routing_api.NewError(routing_api.RouteInvalidError, "Each route request requires a port greater than 0")
		return &err
	}

	if route.Route == "" {
		err := routing_api.NewError(routing_api.RouteInvalidError, "Each route request requires a valid route")
		return &err
	}

	if route.IP == "" {
		err := routing_api.NewError(routing_api.RouteInvalidError, "Each route request requires an IP")
		return &err
	}

	return nil
}

func validateRouteUrl(route string) *routing_api.Error {
	err := validateUrl(route)
	if err != nil {
		err := routing_api.NewError(routing_api.RouteInvalidError, err.Error())
		return &err
	}

	return nil
}

func validateRouteServiceUrl(routeService string) *routing_api.Error {
	if routeService == "" {
		return nil
	}

	if !strings.HasPrefix(routeService, "https://") {
		err := routing_api.NewError(routing_api.RouteServiceUrlInvalidError, "Route service url must use HTTPS.")
		return &err
	}

	err := validateUrl(routeService)
	if err != nil {
		err := routing_api.NewError(routing_api.RouteServiceUrlInvalidError, err.Error())
		return &err
	}

	return nil
}

func validateUrl(urlToValidate string) error {
	if strings.ContainsAny(urlToValidate, "?#") {
		return errors.New("Url cannot contain any of [?, #]")
	}

	parsedURL, err := url.Parse(urlToValidate)

	if err != nil {
		return err
	}

	if parsedURL.String() != urlToValidate {
		return errors.New("Url cannot contain invalid characters")
	}

	return nil
}

func (v Validator) ValidateCreateTcpRouteMapping(tcpRouteMappings []models.TcpRouteMapping, routerGroups models.RouterGroups, maxTTL int) *routing_api.Error {
	for _, tcpRouteMapping := range tcpRouteMappings {
		err := validateTcpRouteMapping(tcpRouteMapping, true, maxTTL)
		if err != nil {
			return err
		}

		validGuid := false
		for _, routerGroup := range routerGroups {
			if tcpRouteMapping.RouterGroupGuid == routerGroup.Guid {
				validGuid = true
				break
			}
		}

		if !validGuid {
			err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
				"router_group_guid: "+tcpRouteMapping.RouterGroupGuid+" not found")
			return &err
		}
	}
	return nil
}

func (v Validator) ValidateDeleteTcpRouteMapping(tcpRouteMappings []models.TcpRouteMapping) *routing_api.Error {
	for _, tcpRouteMapping := range tcpRouteMappings {
		err := validateTcpRouteMapping(tcpRouteMapping, false, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateTcpRouteMapping(tcpRouteMapping models.TcpRouteMapping, checkTTL bool, maxTTL int) *routing_api.Error {
	if tcpRouteMapping.RouterGroupGuid == "" {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp mapping requires a non empty router group guid. RouteMapping=["+tcpRouteMapping.String()+"]")
		return &err
	}

	if tcpRouteMapping.ExternalPort <= 0 {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp mapping requires a positive external port. RouteMapping=["+tcpRouteMapping.String()+"]")
		return &err
	}

	if tcpRouteMapping.HostIP == "" {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp mapping requires a non empty backend ip. RouteMapping=["+tcpRouteMapping.String()+"]")
		return &err
	}

	if tcpRouteMapping.HostPort <= 0 {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp mapping requires a positive backend port. RouteMapping=["+tcpRouteMapping.String()+"]")
		return &err
	}

	if checkTTL && *tcpRouteMapping.TTL > maxTTL {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp mapping requires TTL to be less than or equal to "+strconv.Itoa(int(maxTTL))+". RouteMapping=["+tcpRouteMapping.String()+"]")
		return &err
	}

	if checkTTL && *tcpRouteMapping.TTL <= 0 {
		err := routing_api.NewError(routing_api.TcpRouteMappingInvalidError,
			"Each tcp route mapping requires a ttl greater than 0")
		return &err
	}

	return nil
}
