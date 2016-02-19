package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/handlers"
	"github.com/cloudfoundry-incubator/routing-api/metrics"
	fake_client "github.com/cloudfoundry-incubator/uaa-go-client/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("RouterGroupsHandler", func() {

	var (
		routerGroupHandler *handlers.RouterGroupsHandler
		request            *http.Request
		responseRecorder   *httptest.ResponseRecorder
		fakeClient         *fake_client.FakeClient
		logger             *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-router-group")
		fakeClient = &fake_client.FakeClient{}
		routerGroupHandler = handlers.NewRouteGroupsHandler(fakeClient, logger)
		responseRecorder = httptest.NewRecorder()
	})

	Describe("ListRouterGroups", func() {
		It("responds with 200 OK and returns default router group details", func() {
			var err error
			request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
			Expect(err).NotTo(HaveOccurred())
			routerGroupHandler.ListRouterGroups(responseRecorder, request)
			Expect(responseRecorder.Code).To(Equal(http.StatusOK))
			payload := responseRecorder.Body.String()
			Expect(payload).To(MatchJSON(`[
			{
				  "guid": "bad25cff-9332-48a6-8603-b619858e7992",
					"name": "default-tcp",
					"type": "tcp"
			}]`))
		})

		It("checks for routing.router_groups.read scope", func() {
			var err error
			request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
			Expect(err).NotTo(HaveOccurred())
			routerGroupHandler.ListRouterGroups(responseRecorder, request)
			_, permission := fakeClient.DecodeTokenArgsForCall(0)
			Expect(permission).To(ConsistOf(handlers.RouterGroupsReadScope))
		})

		Context("when authorization token is invalid", func() {
			var (
				currentCount int64
			)
			BeforeEach(func() {
				currentCount = metrics.GetTokenErrors()
				fakeClient.DecodeTokenReturns(errors.New("kaboom"))
			})

			It("returns Unauthorized error", func() {
				var err error
				request, err = http.NewRequest("GET", routing_api.ListRouterGroups, nil)
				Expect(err).NotTo(HaveOccurred())
				routerGroupHandler.ListRouterGroups(responseRecorder, request)
				Expect(responseRecorder.Code).To(Equal(http.StatusUnauthorized))
				Expect(metrics.GetTokenErrors()).To(Equal(currentCount + 1))
			})
		})

	})

})
