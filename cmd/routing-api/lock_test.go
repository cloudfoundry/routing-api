package main_test

import (
	"fmt"
	"os"
	"time"

	"code.cloudfoundry.org/routing-api"
	"code.cloudfoundry.org/routing-api/cmd/routing-api/testrunner"
	"code.cloudfoundry.org/routing-api/test_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Locking", func() {
	var args testrunner.Args
	BeforeEach(func() {
		rapiConfig := getRoutingAPIConfig(defaultConfig)
		args = testrunner.Args{
			IP:         routingAPIIP,
			ConfigPath: writeConfigToTempFile(rapiConfig),
			DevMode:    true,
		}
	})

	AfterEach(func() {
		err := os.RemoveAll(args.ConfigPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("vieing for the lock", func() {
		Context("when two long-lived processes try to run", func() {
			It("one waits for the other to exit and then grabs the lock", func() {
				session1 := RoutingApi(args.ArgSlice()...)
				Eventually(session1, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))

				session2 := RoutingApi(args.ArgSlice()...)

				defer func() {
					session1.Interrupt().Wait(5 * time.Second)
					session2.Interrupt().Wait(10 * time.Second)
				}()

				Eventually(session2, 10*time.Second).Should(gbytes.Say("acquiring-lock"))
				Consistently(session2).ShouldNot(gbytes.Say("acquire-lock-succeeded"))

				session1.Interrupt().Wait(10 * time.Second)

				Eventually(session1, 10*time.Second).Should(gbytes.Say("releasing-lock"))
				Eventually(session2, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))
			})
		})
	})

	Context("when the lock disappears", func() {
		Context("long-lived processes", func() {
			It("should exit 1", func() {
				session1 := RoutingApi(args.ArgSlice()...)
				defer func() {
					session1.Interrupt().Wait(5 * time.Second)
				}()

				Eventually(session1, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))

				err := consulRunner.Reset()
				Expect(err).ToNot(HaveOccurred())

				consulRunner.WaitUntilReady()
				Eventually(session1, 10*time.Second).Should(gbytes.Say("lost-lock"))
				Eventually(session1, 20*time.Second).Should(gexec.Exit(1))
			})
		})
	})
	Context("when a rolling deploy occurs", func() {
		It("ensures there is no downtime", func() {
			session1 := RoutingApi(args.ArgSlice()...)
			client1 := routingApiClientWithPort(routingAPIPort)
			Eventually(session1, 10*time.Second).Should(gbytes.Say("routing-api.started"))

			session2Port := uint16(test_helpers.NextAvailPort())
			apiConfig := getRoutingAPIConfig(defaultConfig)
			apiConfig.API.ListenPort = int(session2Port)
			apiConfig.API.MTLSListenPort = test_helpers.NextAvailPort()
			apiConfig.AdminPort = test_helpers.NextAvailPort()
			configFilePath := writeConfigToTempFile(apiConfig)
			session2Args := testrunner.Args{
				IP:         routingAPIIP,
				ConfigPath: configFilePath,
				DevMode:    true,
			}
			session2 := RoutingApi(session2Args.ArgSlice()...)
			defer func() { session2.Interrupt().Wait(10 * time.Second) }()
			Eventually(session2, 10*time.Second).Should(gbytes.Say("acquiring-lock"))

			done := make(chan struct{})
			goRoutineFinished := make(chan struct{})
			client2 := routing_api.NewClient(fmt.Sprintf("http://127.0.0.1:%d", session2Port), false)

			go func() {
				defer GinkgoRecover()

				var err1, err2 error

				ticker := time.NewTicker(time.Second)
				for range ticker.C {
					select {
					case <-done:
						close(goRoutineFinished)
						ticker.Stop()
						return
					default:
						_, err1 = client1.Routes()
						_, err2 = client2.Routes()
						Expect([]error{err1, err2}).To(ContainElement(Not(HaveOccurred())), "At least one of the errors should not have occurred")
					}
				}
			}()

			session1.Interrupt().Wait(10 * time.Second)

			Eventually(session2, 10*time.Second).Should(gbytes.Say("acquire-lock-succeeded"))
			Eventually(session2, 10*time.Second).Should(gbytes.Say("routing-api.started"))

			close(done)
			Eventually(done).Should(BeClosed())
			Eventually(goRoutineFinished).Should(BeClosed())

			_, err := client2.Routes()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
