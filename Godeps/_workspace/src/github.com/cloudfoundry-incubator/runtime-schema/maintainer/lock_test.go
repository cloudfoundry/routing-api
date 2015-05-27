package maintainer_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"
	"github.com/hashicorp/consul/api"

	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Lock", func() {
	var (
		lockKey   string
		lockValue []byte

		consulSession *consuladapter.Session

		lockRunner    ifrit.Runner
		lockProcess   ifrit.Process
		retryInterval time.Duration
		logger        lager.Logger
	)

	getLockValue := func() ([]byte, error) {
		return consulSession.GetAcquiredValue(lockKey)
	}

	BeforeEach(func() {
		consulSession = consulRunner.NewSession("a-session")

		lockKey = "some-key"
		lockValue = []byte("some-value")

		retryInterval = 500 * time.Millisecond
		logger = lagertest.NewTestLogger("maintainer")
	})

	JustBeforeEach(func() {
		clock := clock.NewClock()
		lockRunner = maintainer.NewLock(consulSession, lockKey, lockValue, clock, retryInterval, logger)
	})

	AfterEach(func() {
		ginkgomon.Kill(lockProcess)
		consulSession.Destroy()
	})

	Context("When consul is running", func() {
		Context("an error occurs while acquiring the lock", func() {
			BeforeEach(func() {
				lockKey = ""
			})

			It("continues to retry", func() {
				lockProcess = ifrit.Background(lockRunner)
				Eventually(consulSession.ID).ShouldNot(Equal(""))

				Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
				Consistently(lockProcess.Wait()).ShouldNot(Receive())

				Eventually(logger).Should(Say("acquire-lock-failed"))
				Eventually(logger).Should(Say("retrying-acquiring-lock"))
			})
		})

		Context("and the lock is available", func() {
			It("acquires the lock", func() {
				lockProcess = ifrit.Background(lockRunner)
				Eventually(lockProcess.Ready()).Should(BeClosed())
				Expect(getLockValue()).To(Equal(lockValue))
			})

			Context("and we have acquired the lock", func() {
				JustBeforeEach(func() {
					lockProcess = ifrit.Background(lockRunner)
					Eventually(lockProcess.Ready()).Should(BeClosed())
				})

				Context("when consul shuts down", func() {
					JustBeforeEach(func() {
						consulRunner.Stop()
					})

					AfterEach(func() {
						consulRunner.Start()
						consulRunner.WaitUntilReady()
					})

					It("loses the lock and exits", func() {
						var err error
						Eventually(lockProcess.Wait()).Should(Receive(&err))
						Expect(err).To(Equal(maintainer.ErrLockLost))
					})
				})

				Context("and the process is shutting down", func() {
					It("releases the lock and exits", func() {
						ginkgomon.Interrupt(lockProcess)
						Eventually(lockProcess.Wait()).Should(Receive(BeNil()))
						_, err := getLockValue()
						Expect(err).To(Equal(consuladapter.NewKeyNotFoundError(lockKey)))
					})
				})
			})
		})

		Context("and the lock is unavailable", func() {
			var (
				otherProcess ifrit.Process
				otherValue   []byte
			)

			BeforeEach(func() {
				otherValue = []byte("doppel-value")
				otherSession := consulRunner.NewSession("other-session")
				clock := clock.NewClock()

				otherRunner := maintainer.NewLock(otherSession, lockKey, otherValue, clock, retryInterval, logger)
				otherProcess = ifrit.Background(otherRunner)

				Eventually(otherProcess.Ready()).Should(BeClosed())
				Expect(getLockValue()).To(Equal(otherValue))
			})

			AfterEach(func() {
				ginkgomon.Interrupt(otherProcess)
			})

			It("waits for the lock to become available", func() {
				lockProcess = ifrit.Background(lockRunner)
				Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
				Expect(getLockValue()).To(Equal(otherValue))
			})

			Context("when consul shuts down", func() {
				JustBeforeEach(func() {
					lockProcess = ifrit.Background(lockRunner)
					Eventually(consulSession.ID).ShouldNot(Equal(""))

					consulRunner.Stop()
				})

				AfterEach(func() {
					consulRunner.Start()
					consulRunner.WaitUntilReady()
				})

				It("continues to wait for the lock", func() {
					Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
					Consistently(lockProcess.Wait()).ShouldNot(Receive())

					Eventually(logger).Should(Say("acquire-lock-failed"))
					Eventually(logger).Should(Say("retrying-acquiring-lock"))
				})
			})

			Context("and the session is destroyed", func() {
				It("should recreate the session and continue to retry", func() {
					lockProcess = ifrit.Background(lockRunner)
					Eventually(consulSession.ID).ShouldNot(Equal(""))

					sessionID := consulSession.ID()

					consulSession.Destroy()
					Eventually(logger).Should(Say("consul-error"))
					Eventually(logger).Should(Say("retrying-acquiring-lock"))

					client := consulRunner.NewClient()

					var entry *api.SessionEntry
					Eventually(func() *api.SessionEntry {
						entries, _, err := client.Session().List(nil)
						Expect(err).NotTo(HaveOccurred())
						for _, e := range entries {
							if e.Name == "a-session" {
								entry = e
								return e
							}
						}
						return nil
					}).ShouldNot(BeNil())

					Expect(entry.ID).NotTo(Equal(sessionID))
				})
			})

			Context("and the process is shutting down", func() {
				It("exits", func() {
					lockProcess = ifrit.Background(lockRunner)
					Eventually(consulSession.ID).ShouldNot(Equal(""))

					ginkgomon.Interrupt(lockProcess)
					Eventually(lockProcess.Wait()).Should(Receive(BeNil()))
				})
			})

			Context("and the lock is released", func() {
				It("acquires the lock", func() {
					lockProcess = ifrit.Background(lockRunner)
					Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
					Expect(getLockValue()).To(Equal(otherValue))

					ginkgomon.Interrupt(otherProcess)

					Eventually(lockProcess.Ready()).Should(BeClosed())
					Expect(getLockValue()).To(Equal(lockValue))
				})
			})
		})
	})

	Context("When consul is down", func() {
		BeforeEach(func() {
			consulRunner.Stop()
		})

		AfterEach(func() {
			consulRunner.Start()
			consulRunner.WaitUntilReady()
		})

		It("continues to retry acquiring the lock", func() {
			lockProcess = ifrit.Background(lockRunner)

			Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
			Consistently(lockProcess.Wait()).ShouldNot(Receive())

			Eventually(logger).Should(Say("acquire-lock-failed"))
			Eventually(logger).Should(Say("retrying-acquiring-lock"))
			Eventually(logger).Should(Say("retrying-acquiring-lock"))
		})

		Context("when consul starts up", func() {
			It("acquires the lock", func() {
				lockProcess = ifrit.Background(lockRunner)

				Eventually(logger).Should(Say("acquire-lock-failed"))
				Eventually(logger).Should(Say("retrying-acquiring-lock"))
				Consistently(lockProcess.Ready()).ShouldNot(BeClosed())
				Consistently(lockProcess.Wait()).ShouldNot(Receive())

				consulRunner.Start()
				consulRunner.WaitUntilReady()

				Eventually(lockProcess.Ready()).Should(BeClosed())
				Expect(getLockValue()).To(Equal(lockValue))
			})
		})
	})
})
