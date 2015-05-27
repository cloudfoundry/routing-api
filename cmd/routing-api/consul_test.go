package main_test

import (
	"net"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connecting to consul", func() {
	var (
		err  error
		addr *net.UDPAddr
	)

	BeforeEach(func() {
		addr, err = net.ResolveUDPAddr("udp", "localhost:8125")
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		routingAPIProcess = ginkgomon.Invoke(routingAPIRunner)
	})

	AfterEach(func() {
		ginkgomon.Kill(routingAPIProcess)
	})

	Describe("Stats for total routes", func() {
		var fakeStatsdServer *net.UDPConn
		var fakeStatsdChan chan []byte

		BeforeEach(func() {
			var err error
			fakeStatsdServer, err = net.ListenUDP("udp", addr)
			Expect(err).ToNot(HaveOccurred())

			fakeStatsdServer.SetReadDeadline(time.Now().Add(10 * time.Second))

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

		Context("when the metrics server initially does not have the lock", func() {
			var otherSession *consuladapter.Session

			BeforeEach(func() {
				otherSession = consulRunner.NewSession("other-session")
				err := otherSession.AcquireLock("v1/locks/routing-api", []byte("something-else"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not emit any metrics", func() {
				Consistently(fakeStatsdChan, 1*time.Second, 10*time.Millisecond).ShouldNot(Receive())
			})

			Context("when the lock becomes available", func() {
				BeforeEach(func() {
					otherSession.Destroy()
				})
				It("starts emitting metrics", func() {
					Eventually(fakeStatsdChan).Should(Receive())
				})
			})
		})
	})
})
