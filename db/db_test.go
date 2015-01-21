package db_test

import (
	"github.com/pivotal-cf-experimental/routing-api/db"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB", func() {
	Describe("etcd", func() {
		var (
			etcd  db.DB
			route db.Route
		)

		BeforeEach(func() {
			etcd = db.NewETCD(etcdRunner.NodeURLS())
			route = db.Route{
				Route:   "post_here",
				Port:    7000,
				IP:      "1.2.3.4",
				TTL:     50,
				LogGuid: "my-guid",
			}
		})

		Describe(".SaveRoute", func() {
			It("Creates a route if none exist", func() {
				err := etcd.SaveRoute(route)
				Expect(err).NotTo(HaveOccurred())

				response, err := etcdClient.Get(`/routes/post_here,1.2.3.4:7000`, false, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(response.Node.TTL).To(Equal(int64(50)))
				Expect(response.Node.Value).To(MatchJSON(`{
						"ip": "1.2.3.4",
						"route": "post_here",
						"port": 7000,
						"ttl": 50,
						"log_guid": "my-guid"
					}`))
			})

			It("Returns the ETCD error if bad inputs are given", func() {
				route.TTL = -1
				err := etcd.SaveRoute(route)
				Expect(err).To(HaveOccurred())
			})

			Context("when an entry already exists", func() {
				BeforeEach(func() {
					route.Route = "next-route"
					route.IP = "9.8.7.6"
					route.Port = 12345

					err := etcd.SaveRoute(route)
					Expect(err).NotTo(HaveOccurred())
				})

				It("Updates a route if one already exists", func() {
					route.TTL = 47
					route.LogGuid = "new-guid"

					err := etcd.SaveRoute(route)
					Expect(err).NotTo(HaveOccurred())

					response, err := etcdClient.Get(`/routes/next-route,9.8.7.6:12345`, false, false)
					Expect(err).NotTo(HaveOccurred())
					Expect(response.Node.TTL).To(Equal(int64(47)))
					Expect(response.Node.Value).To(MatchJSON(`{
						"ip": "9.8.7.6",
						"route": "next-route",
						"port": 12345,
						"ttl": 47,
						"log_guid": "new-guid"
					}`))
				})
			})
		})

		Describe(".DeleteRoute", func() {
			Context("when a route exists", func() {
				BeforeEach(func() {
					err := etcd.SaveRoute(route)
					Expect(err).NotTo(HaveOccurred())
				})

				It("Deletes the route", func() {
					err := etcd.DeleteRoute(route)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when deleting a route returns an error", func() {
				It("returns a key not found error if the key does not exists", func() {
					err := etcd.DeleteRoute(route)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
