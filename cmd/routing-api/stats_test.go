package main_test

import (
	"net"
	"time"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routes API", func() {
	var (
		err    error
		route1 db.Route
		addr   *net.UDPAddr
	)

	BeforeEach(func() {
		routingAPIProcess = ginkgomon.Invoke(routingAPIRunner)
		addr, err = net.ResolveUDPAddr("udp", "localhost:8125")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		ginkgomon.Kill(routingAPIProcess)
	})

	Describe("Stats for event subscribers", func() {
		Context("Subscribe", func() {
			var fakeStatsdServer *net.UDPConn

			BeforeEach(func() {
				var err error
				fakeStatsdServer, err = net.ListenUDP("udp", addr)
				fakeStatsdServer.SetReadDeadline(time.Now().Add(15 * time.Second))
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				err := fakeStatsdServer.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have two subscriptions", func() {
				eventStream1, err := client.SubscribeToEvents()
				Expect(err).NotTo(HaveOccurred())

				eventStream2, err := client.SubscribeToEvents()
				Expect(err).NotTo(HaveOccurred())
				defer eventStream1.Close()
				defer eventStream2.Close()

				var line []byte = make([]byte, 64)
				Eventually(func() []byte {
					n, err := fakeStatsdServer.Read(line)
					Expect(err).ToNot(HaveOccurred())
					return line[:n]
				}).Should(BeEquivalentTo("routing_api.total_subscriptions:2|g"))

			})
		})
	})

	Describe("Stats for total routes", func() {
		var fakeStatsdServer *net.UDPConn
		var fakeStatsdChan chan []byte

		BeforeEach(func() {
			var err error
			fakeStatsdServer, err = net.ListenUDP("udp", addr)
			Expect(err).ToNot(HaveOccurred())

			fakeStatsdServer.SetReadDeadline(time.Now().Add(20 * time.Second))

			route1 = db.Route{
				Route:   "a.b.c",
				Port:    33,
				IP:      "1.1.1.1",
				TTL:     55,
				LogGuid: "potato",
			}

			fakeStatsdChan = make(chan []byte, 1)

			go func() {
				defer GinkgoRecover()
				for {
					buffer := make([]byte, 1024)
					n, err := fakeStatsdServer.Read(buffer)
					if err != nil {
						close(fakeStatsdChan)
						return
					}

					select {
					case fakeStatsdChan <- buffer[:n]:
					default:
					}
				}
			}()
		})

		AfterEach(func() {
			err := fakeStatsdServer.Close()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("periodically receives total routes", func() {
			It("Gets statsd messages for existing routes", func() {
				//The first time is because we get the event of adding the self route
				Eventually(fakeStatsdChan).Should(Receive(BeEquivalentTo("routing_api.total_routes:1|g")))
				//Do it again to make sure it's not because of events
				Eventually(fakeStatsdChan).Should(Receive(BeEquivalentTo("routing_api.total_routes:1|g")))
			})
		})

		Context("when creating and updating a new route", func() {
			It("Gets statsd messages for new routes", func() {
				client.UpsertRoutes([]db.Route{route1})

				Eventually(fakeStatsdChan).Should(Receive(BeEquivalentTo("routing_api.total_routes:2|g")))
			})
		})

		Context("when deleting a route", func() {
			It("gets statsd messages for deleted routes", func() {
				client.UpsertRoutes([]db.Route{route1})

				client.DeleteRoutes([]db.Route{route1})

				Eventually(fakeStatsdChan).Should(Receive(BeEquivalentTo("routing_api.total_routes:1|g")))
			})
		})

		Context("when expiring a route", func() {
			It("gets statsd messages for expired routes", func() {
				routeExpire := db.Route{
					Route:   "z.a.k",
					Port:    63,
					IP:      "42.42.42.42",
					TTL:     1,
					LogGuid: "Tomato",
				}

				client.UpsertRoutes([]db.Route{routeExpire})

				Eventually(fakeStatsdChan).Should(Receive(BeEquivalentTo("routing_api.total_routes:1|g")))
			})
		})
	})
})
