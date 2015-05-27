package consuladapter_test

import (
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/hashicorp/consul/api"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Watching", func() {
	var client *api.Client
	var logger *lagertest.TestLogger

	var disappearChan <-chan []string

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		startClusterAndSession()
	})

	AfterEach(stopClusterAndSession)

	Context("when the watch starts first", func() {
		BeforeEach(func() {
			var err error
			client = clusterRunner.NewClient()

			disappearChan = session.WatchForDisappearancesUnder(logger, "under")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there are keys", func() {
			var bsession *consuladapter.Session

			BeforeEach(func() {
				bsession = clusterRunner.NewSession("bsession")
			})

			AfterEach(func() {
				bsession.Destroy()
			})

			It("detects removals of keys", func() {
				_, err := bsession.SetPresence("under/here", []byte("value"))
				Expect(err).NotTo(HaveOccurred())

				bsession.Destroy()

				Eventually(disappearChan).Should(Receive(Equal([]string{"under/here"})))
			})

			Context("with other prefixes", func() {
				BeforeEach(func() {
					_, err := bsession.SetPresence("other", []byte("value"))
					Expect(err).NotTo(HaveOccurred())
				})

				It("does not detect removal of keys under other prefixes", func() {
					bsession.Destroy()

					Consistently(disappearChan).ShouldNot(Receive())
				})
			})

			Context("when destroying the session", func() {
				It("closes the disappearance channel", func() {
					session.Destroy()

					Eventually(disappearChan, 15).Should(BeClosed())
				})
			})

			Context("when an error occurs", func() {
				It("retries", func() {
					stopCluster()

					Consistently(disappearChan).ShouldNot(Receive())

					startCluster()

					time.Sleep(1 * time.Second) // allow the watch to retry

					bsession = clusterRunner.NewSession("bession")
					_, err := bsession.SetPresence("under/here", []byte("value"))
					Expect(err).NotTo(HaveOccurred())

					bsession.Destroy()

					Eventually(disappearChan).Should(Receive(Equal([]string{"under/here"})))
				})
			})
		})
	})

	Context("when the watch starts later", func() {
		var bsession *consuladapter.Session

		BeforeEach(func() {
			bsession = clusterRunner.NewSession("bsession")
			_, err := bsession.SetPresence("under/here", []byte("value"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("detects removals of keys", func() {
			disappearChan := session.WatchForDisappearancesUnder(logger, "under")

			time.Sleep(1 * time.Second) // allow the watch to retry
			bsession.Destroy()

			Eventually(disappearChan).Should(Receive(Equal([]string{"under/here"})))
		})
	})
})
