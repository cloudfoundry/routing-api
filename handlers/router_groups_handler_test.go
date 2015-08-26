package handlers_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/handlers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("RouterGroupsHandler", func() {

	var (
		routerGroupHandler *handlers.RouterGroupsHandler
		request            *http.Request
		responseRecorder   *httptest.ResponseRecorder
		logger             *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test-router-group")
		routerGroupHandler = handlers.NewRouteGroupsHandler(logger)
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
					"name": "default_tcp",
					"features": [
										        "tcp"
											]
			}]`))
		})

	})

})
