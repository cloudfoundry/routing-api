package db_test

import (
	"encoding/json"
	"errors"
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
				Route:           "post_here",
				Port:            7000,
				IP:              "1.2.3.4",
				TTL:             50,
				LogGuid:         "my-guid",
				RouteServiceUrl: "https://rs.com",
			}
			fakeKeysAPI = &fakes.FakeKeysAPI{}
			fakeEtcd = setupFakeEtcd(fakeKeysAPI)

			tcpRouteMapping1 = models.NewTcpRouteMapping("router-group-guid-002", 52002, "2.3.4.5", 60002)
		})
		Describe("Http Routes", func() {
			Describe("ReadRoutes", func() {
				var routes []models.Route
				var err error

				JustBeforeEach(func() {
					routes, err = fakeEtcd.ReadRoutes()
					Expect(err).ToNot(HaveOccurred())
				})

				Context("when the route key is missing", func() {
					BeforeEach(func() {
						fakeKeysAPI.GetReturns(nil, errors.New("key missing error"))
					})

					It("gives empty list of routes", func() {
						Expect(routes).To(HaveLen(0))
					})
				})
				Context("when there are no routes", func() {
					BeforeEach(func() {
						fakeKeysAPI.GetReturns(&client.Response{Node: &client.Node{}}, nil)
					})

					It("returns an empty list of routes", func() {
						Expect(routes).To(HaveLen(0))
					})
				})

				Context("when there are multiple routes", func() {
					var route2 models.Route

					BeforeEach(func() {
						route2 = models.Route{
							Route:           "some-route/path",
							Port:            5500,
							IP:              "3.1.5.7",
							TTL:             1000,
							LogGuid:         "your-guid",
							RouteServiceUrl: "https://my-rs.com",
						}
						routeJson, err := json.Marshal(route)
						Expect(err).NotTo(HaveOccurred())
						var route2Json []byte
						route2Json, err = json.Marshal(route2)
						Expect(err).NotTo(HaveOccurred())
						node1 := client.Node{Value: string(routeJson)}
						node2 := client.Node{Value: string(route2Json)}
						nodes := []*client.Node{&node1, &node2}
						fakeKeysAPI.GetReturns(&client.Response{Node: &client.Node{Nodes: nodes}}, nil)
					})

					It("returns multiple routes", func() {
						Expect(routes).To(HaveLen(2))
						Expect(routes).To(ContainElement(route))
						Expect(routes).To(ContainElement(route2))
					})
				})
			})

			Describe("SaveRoute", func() {
				Context("when there's no existing entry", func() {
					BeforeEach(func() {
						keyNotFoundError := client.Error{Code: client.ErrorCodeKeyNotFound}
						fakeKeysAPI.GetReturns(nil, keyNotFoundError)
					})

					It("Creates a route if none exist", func() {
						err := fakeEtcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeKeysAPI.GetCallCount()).To(Equal(1))
						Expect(fakeKeysAPI.SetCallCount()).To(Equal(1))
						_, _, json, _ := fakeKeysAPI.SetArgsForCall(0)
						Expect(json).To(ContainSubstring("\"index\":0"))
					})
				})

				Context("when an entry already exists", func() {
					BeforeEach(func() {
						route.ModificationTag = models.ModificationTag{"guid", 5}
						routeJson, err := json.Marshal(&route)
						Expect(err).ToNot(HaveOccurred())
						fakeResp := &client.Response{Node: &client.Node{Value: string(routeJson)}}
						fakeKeysAPI.GetReturns(fakeResp, nil)
					})

					It("Updates the route and increments the tag index", func() {
						err := fakeEtcd.SaveRoute(route)
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeKeysAPI.GetCallCount()).To(Equal(1))
						Expect(fakeKeysAPI.SetCallCount()).To(Equal(1))
						_, _, json, _ := fakeKeysAPI.SetArgsForCall(0)
						Expect(json).To(ContainSubstring("\"index\":6"))
					})

					Context("when Set operation fails with a compare error", func() {

						BeforeEach(func() {
							count := 0
							fakeKeysAPI.SetStub = func(ctx context.Context, key, value string, opts *client.SetOptions) (*client.Response, error) {
								if count == 0 {
									count++
									return nil, client.Error{Code: client.ErrorCodeTestFailed}
								}

								return &client.Response{}, nil
							}
						})

						It("retries successfully", func() {
							err := fakeEtcd.SaveRoute(route)
							Expect(err).NotTo(HaveOccurred())
							Expect(fakeKeysAPI.GetCallCount()).To(Equal(2))
							Expect(fakeKeysAPI.SetCallCount()).To(Equal(2))
						})

						Context("when the number of retries exceeded the max retries", func() {
							BeforeEach(func() {
								fakeKeysAPI.SetReturns(nil, client.Error{Code: client.ErrorCodeTestFailed})
							})

							It("returns a conflict error", func() {
								err := fakeEtcd.SaveRoute(route)
								Expect(err).To(HaveOccurred())
								Expect(err).To(Equal(db.ErrorConflict))
								Expect(fakeKeysAPI.GetCallCount()).To(Equal(4))
								Expect(fakeKeysAPI.SetCallCount()).To(Equal(4))
							})
						})

						Context("when a delete happens in between a read and a set", func() {
							BeforeEach(func() {
								fakeKeysAPI.SetReturns(nil, client.Error{Code: client.ErrorCodeTestFailed})
								count := 0
								fakeKeysAPI.GetStub = func(ctx context.Context, key string, opts *client.GetOptions) (*client.Response, error) {
									if count == 0 {
										count++
										routeJson, err := json.Marshal(&route)
										Expect(err).ToNot(HaveOccurred())
										return &client.Response{Node: &client.Node{Value: string(routeJson)}}, nil
									}
									return nil, client.Error{Code: client.ErrorCodeKeyNotFound}
								}
							})

							It("returns a conflict error", func() {
								err := fakeEtcd.SaveRoute(route)
								Expect(err).To(HaveOccurred())
								Expect(err).To(Equal(db.ErrorConflict))
								Expect(fakeKeysAPI.GetCallCount()).To(Equal(2))
								Expect(fakeKeysAPI.SetCallCount()).To(Equal(1))
							})
						})
					})

					Context("when Set operation fails with a network error", func() {
						BeforeEach(func() {
							fakeKeysAPI.SetReturns(nil, errors.New("some network error"))
						})

						It("returns the network error", func() {
							err := fakeEtcd.SaveRoute(route)
							Expect(err).To(HaveOccurred())
							Expect(err).To(Equal(errors.New("some network error")))
							Expect(fakeKeysAPI.GetCallCount()).To(Equal(1))
							Expect(fakeKeysAPI.SetCallCount()).To(Equal(1))
						})
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
						Expect(event.Type).To(Equal(db.CreateEvent))

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
				var err error

				JustBeforeEach(func() {
					err = fakeEtcd.DeleteRoute(route)
				})

				Context("when a route exists", func() {
					BeforeEach(func() {
						fakeKeysAPI.DeleteReturns(nil, nil)
					})

					It("Deletes the route", func() {
						Expect(err).NotTo(HaveOccurred())
						Expect(fakeKeysAPI.DeleteCallCount()).To(Equal(1))
					})
				})

				Context("when route does not exist", func() {
					BeforeEach(func() {
						fakeKeysAPI.DeleteReturns(nil, client.Error{Code: client.ErrorCodeKeyNotFound})
					})

					It("returns route could not be found error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("The specified route could not be found."))
						Expect(fakeKeysAPI.DeleteCallCount()).To(Equal(1))
					})
				})

				Context("when etcd client returns a network error", func() {
					BeforeEach(func() {
						fakeKeysAPI.DeleteReturns(nil, errors.New("some network error"))
					})

					It("returns route could not be found error", func() {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("some network error"))
						Expect(fakeKeysAPI.DeleteCallCount()).To(Equal(1))
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
