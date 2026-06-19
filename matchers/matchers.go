package matchers

import (
	routing_api "code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/models"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func MatchTcpRoute(target models.TcpRouteMapping) types.GomegaMatcher {
	return gomega.SatisfyAll(
		gomega.WithTransform(func(t models.TcpRouteMapping) string {
			return t.RouterGroupGuid
		}, gomega.Equal(target.RouterGroupGuid)),
		gomega.WithTransform(func(t models.TcpRouteMapping) string {
			return t.HostIP
		}, gomega.Equal(target.HostIP)),
		gomega.WithTransform(func(t models.TcpRouteMapping) uint16 {
			return t.HostPort
		}, gomega.Equal(target.HostPort)),
		gomega.WithTransform(func(t models.TcpRouteMapping) uint16 {
			return t.ExternalPort
		}, gomega.Equal(target.ExternalPort)),
		gomega.WithTransform(func(t models.TcpRouteMapping) string {
			return t.IsolationSegment
		}, gomega.Equal(target.IsolationSegment)),
	)
}

func MatchRouterGroup(target models.RouterGroup) types.GomegaMatcher {
	return gomega.SatisfyAll(
		gomega.WithTransform(func(t models.RouterGroup) string {
			return t.Guid
		}, gomega.Equal(target.Guid)),
		gomega.WithTransform(func(t models.RouterGroup) string {
			return t.Name
		}, gomega.Equal(target.Name)),
		gomega.WithTransform(func(t models.RouterGroup) models.RouterGroupType {
			return t.Type
		}, gomega.Equal(target.Type)),
		gomega.WithTransform(func(t models.RouterGroup) models.ReservablePorts {
			return t.ReservablePorts
		}, gomega.Equal(target.ReservablePorts)),
	)
}

func MatchHttpRoute(target models.Route) types.GomegaMatcher {
	return gomega.SatisfyAll(
		gomega.WithTransform(func(t models.Route) string {
			return t.Route
		}, gomega.Equal(target.Route)),
		gomega.WithTransform(func(t models.Route) uint16 {
			return t.Port
		}, gomega.Equal(target.Port)),
		gomega.WithTransform(func(t models.Route) string {
			return t.IP
		}, gomega.Equal(target.IP)),
		gomega.WithTransform(func(t models.Route) string {
			return t.LogGuid
		}, gomega.Equal(target.LogGuid)),
		gomega.WithTransform(func(t models.Route) string {
			return t.RouteServiceUrl
		}, gomega.Equal(target.RouteServiceUrl)),
	)
}

func MatchHttpEvent(target routing_api.Event) types.GomegaMatcher {
	return gomega.SatisfyAll(
		gomega.WithTransform(func(t routing_api.Event) string {
			return t.Action
		}, gomega.Equal(target.Action)),
		gomega.WithTransform(func(t routing_api.Event) models.Route {
			return t.Route
		}, MatchHttpRoute(target.Route)),
	)
}
