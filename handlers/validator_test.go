package handlers_test

import (
	"fmt"

	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/handlers"
	"code.cloudfoundry.org/routing-api/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validator", func() {
	var (
		validator handlers.Validator
		routes    []models.Route
		maxTTL    int
	)

	BeforeEach(func() {
		validator = handlers.NewValidator()
		maxTTL = 50

		route := models.NewRoute("http://127.0.0.1/a/valid/route", 8080, "127.0.0.1", "log_guid", "https://my-rs.example.com", maxTTL)
		routes = []models.Route{route}
	})

	Describe(".ValidateCreate", func() {
		It("does not return an error if all route inputs are valid", func() {
			err := validator.ValidateCreate(routes, maxTTL)
			Expect(err).To(BeNil())
		})

		Context("when any route has an invalid value", func() {
			BeforeEach(func() {
				routes = append(routes, routes[0])
			})

			It("returns an error if any ttl is greater than max ttl", func() {
				*routes[1].TTL = maxTTL + 1

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal(fmt.Sprintf("Max ttl is %d", maxTTL)))
			})

			It("returns an error if any ttl is less than 1", func() {
				*routes[1].TTL = 0

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Request requires a ttl greater than 0"))
			})

			It("returns an error if any request does not have a route", func() {
				routes[0].Route = ""

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires a valid route"))
			})

			It("returns an error if any port is less than 1", func() {
				routes[0].Port = 0

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires a port greater than 0"))
			})

			It("returns an error if the path contains invalid characters", func() {
				routes[0].Route = "/foo/b ar"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Url cannot contain invalid characters"))
			})

			It("returns an error if the path is not valid", func() {
				routes[0].Route = "/foo/bar%"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(ContainSubstring("invalid URL"))
			})

			It("returns an error if the path contains a question mark", func() {
				routes[0].Route = "/foo/bar?a"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(ContainSubstring("cannot contain any of [?, #]"))
			})

			It("returns an error if the path contains a hash mark", func() {
				routes[0].Route = "/foo/bar#a"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(ContainSubstring("cannot contain any of [?, #]"))
			})

			It("returns an error if the route service url is not https", func() {
				routes[0].RouteServiceUrl = "http://my-rs.com/ab"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(Equal("Route service url must use HTTPS."))
			})

			It("returns an error if the route service url contains invalid characters", func() {
				routes[0].RouteServiceUrl = "https://my-rs.com/a  b"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(Equal("Url cannot contain invalid characters"))
			})

			It("returns an error if the route service url host is not valid", func() {
				routes[0].RouteServiceUrl = "https://my-rs%.com"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(ContainSubstring("invalid URL escape"))
			})

			It("returns an error if the route service url path is not valid", func() {
				routes[0].RouteServiceUrl = "https://my-rs.com/ad%"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(ContainSubstring("invalid URL"))
			})

			It("returns an error if the route service url contains a question mark", func() {
				routes[0].RouteServiceUrl = "https://foo/bar?a"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(ContainSubstring("cannot contain any of [?, #]"))
			})

			It("returns an error if the route service url contains a hash mark", func() {
				routes[0].RouteServiceUrl = "https://foo/bar#a"

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.RouteServiceUrlInvalidError))
				Expect(err.Error()).To(ContainSubstring("cannot contain any of [?, #]"))
			})

			It("returns an error if any request does not have an IP", func() {
				routes[1].IP = ""

				err := validator.ValidateCreate(routes, maxTTL)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires an IP"))
			})
		})
	})

	Describe(".ValidateDelete", func() {
		It("does not return an error if all route inputs are valid", func() {
			err := validator.ValidateDelete(routes)
			Expect(err).To(BeNil())
		})

		Context("when any route has an invalid value", func() {
			BeforeEach(func() {
				routes = append(routes, routes[0])
			})

			It("returns an error if any request does not have a route", func() {
				routes[0].Route = ""

				err := validator.ValidateDelete(routes)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires a valid route"))
			})

			It("returns an error if any port is less than 1", func() {
				routes[0].Port = 0

				err := validator.ValidateDelete(routes)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires a port greater than 0"))
			})

			It("returns an error if any request does not have an IP", func() {
				routes[1].IP = ""

				err := validator.ValidateDelete(routes)
				Expect(err.Type).To(Equal(routing_api.RouteInvalidError))
				Expect(err.Error()).To(Equal("Each route request requires an IP"))
			})
		})
	})

	Describe("ValidateCreateTcpRouteMapping", func() {
		var (
			tcpMapping   models.TcpRouteMapping
			routerGroups models.RouterGroups
		)

		BeforeEach(func() {
			routerGroups = models.RouterGroups{
				{
					Guid:            DefaultRouterGroupGuid,
					Name:            "default-tcp",
					Type:            "tcp",
					ReservablePorts: "1024-65535",
				},
			}
			tcpMapping = models.NewTcpRouteMapping(DefaultRouterGroupGuid, 52000, "1.2.3.4", 60000, 60001, "instanceId", nil, 60, models.ModificationTag{})
		})

		Context("when valid tcp mapping is passed", func() {
			It("does not return error", func() {
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).To(BeNil())
			})
		})

		Context("when invalid tcp route mappings are passed", func() {

			It("blows up when a backend port is zero", func() {
				tcpMapping.HostPort = 0
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a positive backend port"))
			})

			It("blows up when a external port is zero", func() {
				tcpMapping.ExternalPort = 0
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a positive external port"))
			})

			It("blows up when backend ip empty", func() {
				tcpMapping.HostIP = ""
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a non empty backend ip"))
			})

			It("blows up when group guid is empty", func() {
				tcpMapping.RouterGroupGuid = ""
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a non empty router group guid"))
			})

			It("blows up when group guid is unknown", func() {
				tcpMapping.RouterGroupGuid = "unknown-router-group-guid"
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("router_group_guid: unknown-router-group-guid not found"))
			})

			It("blows up when TTL is greater than 120", func() {
				*tcpMapping.TTL = 200
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires TTL to be less than or equal to 120"))
			})

			It("blows up when TTL is equal to 0", func() {
				*tcpMapping.TTL = 0
				err := validator.ValidateCreateTcpRouteMapping([]models.TcpRouteMapping{tcpMapping}, routerGroups, 120)
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp route mapping requires a ttl greater than 0"))
			})
		})
	})

	Describe("ValidateDeleteTcpRouteMapping", func() {
		var (
			tcpMapping models.TcpRouteMapping
		)

		BeforeEach(func() {
			tcpMapping = models.NewTcpRouteMapping(DefaultRouterGroupGuid, 52000, "1.2.3.4", 60000, 60001, "instanceId", nil, 60, models.ModificationTag{})
		})

		Context("when valid tcp mapping is passed", func() {
			It("does not return error", func() {
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).To(BeNil())
			})
		})

		Context("when invalid tcp route mappings are passed", func() {

			It("blows up when a backend port is zero", func() {
				tcpMapping.HostPort = 0
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a positive backend port"))
			})

			It("blows up when a external port is zero", func() {
				tcpMapping.ExternalPort = 0
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a positive external port"))
			})

			It("blows up when backend ip empty", func() {
				tcpMapping.HostIP = ""
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a non empty backend ip"))
			})

			It("blows up when group guid is empty", func() {
				tcpMapping.RouterGroupGuid = ""
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).ToNot(BeNil())
				Expect(err.Type).To(Equal(routing_api.TcpRouteMappingInvalidError))
				Expect(err.Error()).To(ContainSubstring("Each tcp mapping requires a non empty router group guid"))
			})

			It("does not blow up when group guid is unknown", func() {
				tcpMapping.RouterGroupGuid = "unknown-router-group-guid"
				err := validator.ValidateDeleteTcpRouteMapping([]models.TcpRouteMapping{tcpMapping})
				Expect(err).To(BeNil())
			})
		})
	})
})
