package main_test

import (
	"fmt"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/cloudfoundry-incubator/routing-api/cmd/routing-api/test_helpers"
	"github.com/cloudfoundry-incubator/routing-api/cmd/routing-api/testrunner"
	"github.com/cloudfoundry-incubator/routing-api/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes API", func() {
	var routingAPIProcess ifrit.Process

	BeforeEach(func() {
		routingAPIRunner := testrunner.New(routingAPIBinPath, routingAPIArgs)
		routingAPIProcess = ginkgomon.Invoke(routingAPIRunner)
	})

	AfterEach(func() {
		ginkgomon.Kill(routingAPIProcess)
	})

	Describe("Routes", func() {
		var routes []models.Route
		var getErr error
		var route1, route2 models.Route

		BeforeEach(func() {
			route1 = models.NewRoute("a.b.c", 33, "1.1.1.1", "potato", "", 55)
			route2 = models.NewRoute("d.e.f", 35, "1.1.1.1", "banana", "", 66)

			routesToInsert := []models.Route{route1, route2}
			upsertErr := client.UpsertRoutes(routesToInsert)
			Expect(upsertErr).NotTo(HaveOccurred())
			routes, getErr = client.Routes()
		})

		It("responds without an error", func() {
			Expect(getErr).NotTo(HaveOccurred())
		})

		It("fetches all of the routes", func() {
			routingAPIRoute := models.NewRoute(fmt.Sprintf("api.%s/routing", routingAPISystemDomain), routingAPIPort, routingAPIIP, "my_logs", "", 120)

			Expect(routes).To(HaveLen(3))
			Expect(Routes(routes).ContainsAll(route1, route2, routingAPIRoute)).To(BeTrue())
		})

		It("deletes a route", func() {
			err := client.DeleteRoutes([]models.Route{route1})

			Expect(err).NotTo(HaveOccurred())

			routes, err = client.Routes()
			Expect(err).NotTo(HaveOccurred())
			Expect(Routes(routes).Contains(route1)).To(BeFalse())
		})

		It("rejects bad routes", func() {
			route3 := models.NewRoute("foo/b ar", 35, "2.2.2.2", "banana", "", 66)

			err := client.UpsertRoutes([]models.Route{route3})
			Expect(err).To(HaveOccurred())

			routes, err = client.Routes()

			Expect(err).ToNot(HaveOccurred())
			Expect(Routes(routes).Contains(route1)).To(BeTrue())
			Expect(Routes(routes).Contains(route2)).To(BeTrue())
			Expect(Routes(routes).Contains(route3)).To(BeFalse())
		})

		Context("when a route has a context path", func() {
			var routeWithPath models.Route

			BeforeEach(func() {
				routeWithPath = models.NewRoute("host.com/path", 51480, "1.2.3.4", "logguid", "", 60)
				err := client.UpsertRoutes([]models.Route{routeWithPath})
				Expect(err).ToNot(HaveOccurred())
			})

			It("is present in the routes list", func() {
				var err error
				routes, err = client.Routes()
				Expect(err).ToNot(HaveOccurred())
				Expect(Routes(routes).Contains(routeWithPath)).To(BeTrue())
			})
		})
	})
})
