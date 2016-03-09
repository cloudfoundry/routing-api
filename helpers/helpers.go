package helpers

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/models"
	"github.com/pivotal-golang/lager"
)

const (
	DefaultRouterGroupGuid = "bad25cff-9332-48a6-8603-b619858e7992"
	DefaultRouterGroupName = "default-tcp"
	DefaultRouterGroupType = "tcp"
)

type RouteRegister struct {
	database db.DB
	route    models.Route
	ticker   *time.Ticker
	logger   lager.Logger
}

func NewRouteRegister(database db.DB, route models.Route, ticker *time.Ticker, logger lager.Logger) *RouteRegister {
	return &RouteRegister{
		database: database,
		route:    route,
		ticker:   ticker,
		logger:   logger,
	}
}

func (r *RouteRegister) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	err := r.database.SaveRoute(r.route)
	if err != nil {
		r.logger.Error("Error registering self", err)
	}
	close(ready)

	for {

		select {
		case <-r.ticker.C:
			err = r.database.SaveRoute(r.route)
		case <-signals:
			err := r.database.DeleteRoute(r.route)
			if err != nil {
				r.logger.Error("Error deleting route registration", err)
				return err
			}
			return nil
		}
	}
}

func GetDefaultRouterGroup() models.RouterGroup {
	return models.RouterGroup{
		Guid: DefaultRouterGroupGuid,
		Name: DefaultRouterGroupName,
		Type: DefaultRouterGroupType,
	}
}
