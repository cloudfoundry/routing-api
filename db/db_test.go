package db_test

import (
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/db/fakes"
	"github.com/cloudfoundry-incubator/routing-api/models"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/client"
	"github.com/nu7hatch/gouuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB", func() {
	Context("when no URLs are passed in", func() {
		var (
			etcd db.DB
			err  error
		)

		BeforeEach(func() {
			etcd, err = db.NewETCD([]string{})
		})

		It("should not return an etcd instance", func() {
			Expect(err).To(HaveOccurred())
			Expect(etcd).To(BeNil())
		})
	})

	Context("when connect fails", func() {
		var (
			etcd db.DB
			err  error
		)

		BeforeEach(func() {
			etcd, err = db.NewETCD([]string{"im-not-really-running"})
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error", func() {
			Expect(etcd.Connect()).To(HaveOccurred())
		})
	})

	Describe("etcd", func() {
		var (
			etcd             db.DB
			fakeEtcd         db.DB
			fakeKeysAPI      *fakes.FakeKeysAPI
			err              error
			route            models.Route
			tcpRouteMapping1 models.TcpRouteMapping
		)

		BeforeEach(func() {
			etcd, err = db.NewETCD(etcdRunner.NodeURLS())
			Expect(err).NotTo(HaveOccurred())
			route = models.Route{
				Route:   "post_here",
				Port:    7000,
				IP:      "1.2.3.4",
				TTL:     50,
				LogGuid: "my-guid",
			}
			fakeKeysAPI = &fakes.FakeKeysAPI{}
			fakeEtcd = setupFakeEtcd(fakeKeysAPI)

			tcpRouteMapping1 = models.NewTcpRouteMapping("router-group-guid-002", 52002, "2.3.4.5", 60002)
		})
		Describe("Http Routes", func() {
			Describe("ReadRoutes", func() {
				It("Returns a empty list of routes", func() {
					routes, err := etcd.ReadRoutes()
					Expect(err).NotTo(HaveOccurred())
					Expect(routes).To(Equal([]models.Route{}))
				})

				Context("when only one entry is present", func() {
					BeforeEach(func() {
						route.Route = "next-route"
						route.IP = "9.8.7.6"
						route.Port = 12345

						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
					})

					It("Returns a list with one route", func() {
						routes, err := etcd.ReadRoutes()
						Expect(err).NotTo(HaveOccurred())

						Expect(routes).To(ContainElement(route))
					})
				})

				Context("when the route contains a path", func() {
					BeforeEach(func() {
						route.Route = "route/path"
						route.IP = "9.8.7.6"
						route.Port = 12345

						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
					})

					It("returns the route", func() {
						routes, err := etcd.ReadRoutes()
						Expect(err).NotTo(HaveOccurred())

						Expect(routes).To(ContainElement(route))
					})
				})

				Context("when multiple entries present", func() {
					var (
						route2 models.Route
						route3 models.Route
					)

					BeforeEach(func() {
						route.Route = "next-route"
						route.IP = "9.8.7.6"
						route.Port = 12345

						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())

						route2 = models.Route{
							Route:           "some-route",
							Port:            5500,
							IP:              "3.1.5.7",
							TTL:             1000,
							LogGuid:         "your-guid",
							RouteServiceUrl: "https://my-rs.com",
						}
						err = etcd.SaveRoute(route2)
						Expect(err).NotTo(HaveOccurred())

						route3 = models.Route{
							Route:   "some-other-route",
							Port:    5500,
							IP:      "3.1.5.7",
							TTL:     1000,
							LogGuid: "your-guid",
						}
						err = etcd.SaveRoute(route3)
						Expect(err).NotTo(HaveOccurred())
					})

					It("Returns a list with multiple routes", func() {
						routes, err := etcd.ReadRoutes()
						Expect(err).NotTo(HaveOccurred())

						Expect(routes).To(ContainElement(route))
						Expect(routes).To(ContainElement(route2))
						Expect(routes).To(ContainElement(route3))
					})
				})
			})

			Describe("SaveRoute", func() {
				It("Creates a route if none exist", func() {
					err := etcd.SaveRoute(route)
					Expect(err).NotTo(HaveOccurred())

					node, err := etcdClient.Get(`/routes/post_here,1.2.3.4:7000`)
					Expect(err).NotTo(HaveOccurred())
					Expect(node.TTL).To(Equal(uint64(50)))
					Expect(node.Value).To(MatchJSON(`{
							"ip": "1.2.3.4",
							"route": "post_here",
							"port": 7000,
							"ttl": 50,
							"log_guid": "my-guid"
						}`))
				})

				Context("when a route has a route_service_url", func() {
					BeforeEach(func() {
						route = models.Route{
							Route:           "post_here",
							Port:            7000,
							IP:              "1.2.3.4",
							TTL:             50,
							LogGuid:         "my-guid",
							RouteServiceUrl: "https://my-rs.com",
						}
					})

					It("Creates a route if none exist", func() {
						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())

						node, err := etcdClient.Get(`/routes/post_here,1.2.3.4:7000`)
						Expect(err).NotTo(HaveOccurred())
						Expect(node.TTL).To(Equal(uint64(50)))
						Expect(node.Value).To(MatchJSON(`{
							"ip": "1.2.3.4",
							"route": "post_here",
							"port": 7000,
							"ttl": 50,
							"log_guid": "my-guid",
							"route_service_url":"https://my-rs.com"
						}`))
					})
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

						node, err := etcdClient.Get(`/routes/next-route,9.8.7.6:12345`)
						Expect(err).NotTo(HaveOccurred())
						Expect(node.TTL).To(Equal(uint64(47)))
						Expect(node.Value).To(MatchJSON(`{
							"ip": "9.8.7.6",
							"route": "next-route",
							"port": 12345,
							"ttl": 47,
							"log_guid": "new-guid"
						}`))
					})
				})
			})

			Describe("WatchRouteChanges with http events", func() {
				It("does return an error when canceled", func() {
					_, errors, cancel := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)
					cancel()
					Consistently(errors).ShouldNot(Receive())
					Eventually(errors).Should(BeClosed())
				})

				Context("Cancel Watches", func() {
					It("cancels any in-flight watches", func() {
						results, err, _ := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)
						results2, err2, _ := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)

						etcd.CancelWatches()

						Eventually(err).Should(BeClosed())
						Eventually(err2).Should(BeClosed())
						Eventually(results).Should(BeClosed())
						Eventually(results2).Should(BeClosed())
					})
				})

				Context("with wrong event type", func() {
					BeforeEach(func() {
						fakeresp := &client.Response{Action: "some-action"}
						fakeWatcher := &fakes.FakeWatcher{}
						fakeWatcher.NextReturns(fakeresp, nil)
						fakeKeysAPI.WatcherReturns(fakeWatcher)
					})

					It("throws an error", func() {
						event, err, _ := fakeEtcd.WatchRouteChanges("some-random-key")
						Eventually(err).Should(Receive())
						Eventually(err).Should(BeClosed())

						Consistently(event).ShouldNot(Receive())
						Eventually(event).Should(BeClosed())
					})
				})

				Context("and have outdated index", func() {
					var outdatedIndex = true

					BeforeEach(func() {
						fakeWatcher := &fakes.FakeWatcher{}
						fakeWatcher.NextStub = func(context.Context) (*client.Response, error) {
							if outdatedIndex {
								outdatedIndex = false
								return nil, client.Error{Code: client.ErrorCodeEventIndexCleared}
							} else {
								return &client.Response{Action: "create"}, nil
							}
						}

						fakeKeysAPI.WatcherReturns(fakeWatcher)
					})

					It("resets the index", func() {
						_, err, _ := fakeEtcd.WatchRouteChanges("some-key")
						Expect(err).NotTo(Receive())
						Expect(fakeKeysAPI.WatcherCallCount()).To(Equal(2))

						_, resetOpts := fakeKeysAPI.WatcherArgsForCall(1)
						Expect(resetOpts.AfterIndex).To(BeZero())
						Expect(resetOpts.Recursive).To(BeTrue())
					})

					It("does not throws an error", func() {
						_, err, _ := fakeEtcd.WatchRouteChanges("some-key")
						Expect(err).NotTo(Receive())
					})
				})

				Context("when a route is upserted", func() {
					It("should return an update watch event", func() {
						results, _, _ := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)

						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())

						var event db.Event
						Eventually(results).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.UpdateEvent))

						By("when tcp route is upserted")
						err = etcd.SaveTcpRouteMapping(tcpRouteMapping1)
						Expect(err).NotTo(HaveOccurred())
						Consistently(results).ShouldNot(Receive())
					})
				})

				Context("when a route is deleted", func() {
					It("should return an delete watch event", func() {
						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())

						results, _, _ := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)

						err = etcd.DeleteRoute(route)
						Expect(err).NotTo(HaveOccurred())

						var event db.Event
						Eventually(results).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.DeleteEvent))
					})
				})

				Context("when a route is expired", func() {
					It("should return an expire watch event", func() {
						route.TTL = 1
						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
						results, _, _ := etcd.WatchRouteChanges(db.HTTP_ROUTE_BASE_KEY)

						time.Sleep(1 * time.Second)
						var event db.Event
						Eventually(results).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.ExpireEvent))
					})
				})
			})

			Describe("DeleteRoute", func() {
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
					It("passes the error through", func() {
						etcd, err = db.NewETCD([]string{"im-not-really-running"})
						Expect(err).NotTo(HaveOccurred())

						err := etcd.DeleteRoute(route)
						Expect(err).To(HaveOccurred())
					})

					It("returns a key not found error if the key does not exists", func() {
						err := etcd.DeleteRoute(route)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("The specified route could not be found."))
					})
				})
			})
		})

		Describe("Tcp Mappings", func() {
			var (
				tcpMapping models.TcpRouteMapping
			)

			BeforeEach(func() {
				tcpMapping = models.NewTcpRouteMapping("router-group-guid-001", 52000, "1.2.3.4", 60000)
			})

			Describe("SaveTcpRouteMapping", func() {
				It("Creates a mapping if none exist", func() {
					err := etcd.SaveTcpRouteMapping(tcpMapping)

					Expect(err).NotTo(HaveOccurred())

					key := fmt.Sprintf("%s/%s/%d/%s:%d", db.TCP_MAPPING_BASE_KEY, "router-group-guid-001", 52000, "1.2.3.4", 60000)

					node, err := etcdClient.Get(key)
					Expect(err).NotTo(HaveOccurred())
					Expect(node.Value).To(MatchJSON(`{
							"router_group_guid":"router-group-guid-001",
							"port":52000,
							"backend_ip": "1.2.3.4",
							"backend_port": 60000
						}`))
				})
			})

			Describe("ReadTcpRouteMappings", func() {
				It("Returns a empty list of routes", func() {
					tcpMappings, err := etcd.ReadTcpRouteMappings()
					Expect(err).NotTo(HaveOccurred())
					Expect(tcpMappings).To(Equal([]models.TcpRouteMapping{}))
				})

				Context("when only one entry is present", func() {
					BeforeEach(func() {
						err := etcd.SaveTcpRouteMapping(tcpMapping)
						Expect(err).NotTo(HaveOccurred())
					})

					It("Returns a list with one route", func() {
						tcpMappings, err := etcd.ReadTcpRouteMappings()
						Expect(err).NotTo(HaveOccurred())
						Expect(tcpMappings).To(ContainElement(tcpMapping))
					})
				})
			})

			Describe("WatchRouteChanges with tcp events", func() {
				Context("when a tcp route is upserted", func() {
					It("should return an update watch event", func() {
						results, _, _ := etcd.WatchRouteChanges(db.TCP_MAPPING_BASE_KEY)

						err = etcd.SaveTcpRouteMapping(tcpRouteMapping1)
						Expect(err).NotTo(HaveOccurred())

						var event db.Event
						Eventually(results).Should((Receive(&event)))
						Expect(event).NotTo(BeNil())
						Expect(event.Type).To(Equal(db.UpdateEvent))

						By("when http route is upserted")
						err := etcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
						Consistently(results).ShouldNot(Receive())
					})
				})
			})

			Describe("DeleteTcpRouteMapping", func() {
				Context("when a tcp route mapping exists", func() {
					BeforeEach(func() {
						err := etcd.SaveTcpRouteMapping(tcpMapping)
						Expect(err).NotTo(HaveOccurred())
					})

					It("Deletes the tcp route mapping", func() {
						err := etcd.DeleteTcpRouteMapping(tcpMapping)
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("when deleting a tcp route mapping returns an error", func() {
					It("returns a key not found error if the key does not exists", func() {
						nonExistingTcpMapping := models.NewTcpRouteMapping("router-group-guid-009", 53000, "1.2.3.4", 60000)
						err := etcd.DeleteTcpRouteMapping(nonExistingTcpMapping)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("The specified route (router-group-guid-009:53000<->1.2.3.4:60000) could not be found."))
					})
				})
			})
		})

		Context("RouterGroup", func() {
			Context("Save", func() {
				Context("when router group is missing a guid", func() {
					It("does not save the router group", func() {
						routerGroup := models.RouterGroup{
							Name:            "router-group-1",
							Type:            "tcp",
							ReservablePorts: "10-20,25",
						}
						err = etcd.SaveRouterGroup(routerGroup)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("missing guid"))
					})
				})

				Context("when router group does not exist", func() {
					It("saves the router group", func() {
						g, err := uuid.NewV4()
						Expect(err).NotTo(HaveOccurred())
						guid := g.String()

						routerGroup := models.RouterGroup{
							Name:            "router-group-1",
							Type:            "tcp",
							Guid:            guid,
							ReservablePorts: "10-20,25",
						}
						err = etcd.SaveRouterGroup(routerGroup)
						Expect(err).NotTo(HaveOccurred())

						node, err := etcdClient.Get(db.ROUTER_GROUP_BASE_KEY + "/" + guid)
						Expect(err).NotTo(HaveOccurred())
						Expect(node.TTL).To(Equal(uint64(0)))
						expected := `{
							"name": "router-group-1",
							"type": "tcp",
							"guid": "` + guid + `",
							"reservable_ports": "10-20,25"
						}`
						Expect(node.Value).To(MatchJSON(expected))
					})
				})

				Context("when router group does exist", func() {
					var (
						guid        string
						routerGroup models.RouterGroup
					)

					BeforeEach(func() {
						g, err := uuid.NewV4()
						Expect(err).NotTo(HaveOccurred())
						guid = g.String()

						routerGroup = models.RouterGroup{
							Name:            "router-group-1",
							Type:            "tcp",
							Guid:            guid,
							ReservablePorts: "10-20,25",
						}
						err = etcd.SaveRouterGroup(routerGroup)
						Expect(err).NotTo(HaveOccurred())
					})

					It("can list the router groups", func() {
						rg, err := etcd.ReadRouterGroups()
						Expect(err).NotTo(HaveOccurred())
						Expect(len(rg)).To(Equal(1))
						Expect(rg[0]).Should(Equal(routerGroup))
					})

					It("updates the router group", func() {
						routerGroup.Type = "http"
						routerGroup.ReservablePorts = "10-20,25,30"

						err := etcd.SaveRouterGroup(routerGroup)
						Expect(err).NotTo(HaveOccurred())

						node, err := etcdClient.Get(db.ROUTER_GROUP_BASE_KEY + "/" + guid)
						Expect(err).NotTo(HaveOccurred())
						Expect(node.TTL).To(Equal(uint64(0)))
						expected := `{
							"name": "router-group-1",
							"type": "http",
							"guid": "` + guid + `",
							"reservable_ports": "10-20,25,30"
						}`
						Expect(node.Value).To(MatchJSON(expected))
					})

					It("does not allow name to be updated", func() {
						routerGroup.Name = "not-updatable-name"
						err := etcd.SaveRouterGroup(routerGroup)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Name cannot be updated"))
					})

					It("does not allow duplicate router groups with same name", func() {
						guid, err := uuid.NewV4()
						Expect(err).ToNot(HaveOccurred())
						routerGroup.Guid = guid.String()
						err = etcd.SaveRouterGroup(routerGroup)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("already exists"))
					})

					It("does not allow name to be empty", func() {
						routerGroup.Name = ""
						err := etcd.SaveRouterGroup(routerGroup)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Name cannot be updated"))
					})
				})
			})

		})
	})
})

func setupFakeEtcd(keys client.KeysAPI) db.DB {
	nodeURLs := []string{"127.0.0.1:5000"}

	cfg := client.Config{
		Endpoints: nodeURLs,
		Transport: client.DefaultTransport,
	}

	client, err := client.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	ctx, cancel := context.WithCancel(context.Background())
	return db.New(client, keys, ctx, cancel)
}
