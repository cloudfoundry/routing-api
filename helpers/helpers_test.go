package helpers_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/routing-api/db"
	fake_db "github.com/cloudfoundry-incubator/routing-api/db/fakes"
	"github.com/cloudfoundry-incubator/routing-api/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Helpers", func() {
	Describe("RegisterRoutingAPI", func() {
		var (
			database fake_db.FakeDB
			route    db.Route
			logger   *lagertest.TestLogger

			timeChan chan time.Time
			ticker   *time.Ticker

			quitChan chan bool
		)

		BeforeEach(func() {
			route = db.Route{
				Route:   "i dont care",
				Port:    3000,
				IP:      "i dont care even more",
				TTL:     120,
				LogGuid: "i care a little bit more now",
			}
			database = fake_db.FakeDB{}
			logger = lagertest.NewTestLogger("event-handler-test")

			timeChan = make(chan time.Time)
			ticker = &time.Ticker{C: timeChan}

			quitChan = make(chan bool)
		})

		Context("registration", func() {
			Context("with no errors", func() {
				BeforeEach(func() {
					database.SaveRouteStub = func(route db.Route) error {
						return nil
					}
				})

				It("registers the route for a routing api on init", func() {
					go helpers.RegisterRoutingAPI(quitChan, &database, route, ticker, logger)

					Eventually(func() int { return database.SaveRouteCallCount() }).Should(Equal(1))
					Eventually(func() db.Route { return database.SaveRouteArgsForCall(0) }).Should(Equal(route))
				})

				It("registers on an interval", func() {
					go helpers.RegisterRoutingAPI(quitChan, &database, route, ticker, logger)
					timeChan <- time.Now()

					Eventually(func() int { return database.SaveRouteCallCount() }).Should(Equal(2))
					Eventually(func() db.Route { return database.SaveRouteArgsForCall(1) }).Should(Equal(route))
					Eventually(func() int { return len(logger.Logs()) }).Should(Equal(0))
				})
			})

			Context("when there are errors", func() {
				It("only logs the error once for each attempt", func() {
					database.SaveRouteStub = func(route db.Route) error {
						return errors.New("beep boop, self destruct mode engaged")
					}

					go helpers.RegisterRoutingAPI(quitChan, &database, route, ticker, logger)

					Consistently(func() int { return len(logger.Logs()) }).Should(BeNumerically("<=", 1))
					Eventually(func() string {
						if len(logger.Logs()) > 0 {
							return logger.Logs()[0].Data["error"].(string)
						} else {
							return ""
						}
					}).Should(ContainSubstring("beep boop, self destruct mode engaged"))
				})
			})
		})

		Context("unregistration", func() {
			It("unregisters the routing api when a quit message is received", func() {
				go func() {
					quitChan <- true
				}()

				helpers.RegisterRoutingAPI(quitChan, &database, route, ticker, logger)

				Expect(database.DeleteRouteCallCount()).To(Equal(1))
				Expect(database.DeleteRouteArgsForCall(0)).To(Equal(route))
				Expect(quitChan).To(BeClosed())
			})
		})
	})
})
