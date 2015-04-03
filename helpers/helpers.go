package helpers

import (
	"time"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/pivotal-golang/lager"
)

func RegisterRoutingAPI(quit chan bool, database db.DB, route db.Route, ticker *time.Ticker, logger lager.Logger) {
	err := database.SaveRoute(route)
	defer close(quit)

	for {
		if err != nil {
			logger.Error("Error registering self", err)
		}

		select {
		case <-ticker.C:
			err = database.SaveRoute(route)
		case <-quit:
			err := database.DeleteRoute(route)
			if err != nil {
				logger.Error("Error deleting route registration", err)
			}
			return
		}
	}
}
