package admin

import (
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/routing-api/db"
	"code.cloudfoundry.org/routing-api/handlers"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/rata"
)

const (
	LockRouterGroupReadsRoute    = "LockRouterGroupReads"
	UnlockRouterGroupReadsRoute  = "UnlockRouterGroupReads"
	LockRouterGroupWritesRoute   = "LockRouterGroupWrites"
	UnlockRouterGroupWritesRoute = "UnlockRouterGroupWrites"
)

var AdminRoutesMap = map[string]rata.Route{
	LockRouterGroupReadsRoute:    {Path: "/lock_router_group_reads", Method: "PUT", Name: LockRouterGroupReadsRoute},
	UnlockRouterGroupReadsRoute:  {Path: "/unlock_router_group_reads", Method: "PUT", Name: UnlockRouterGroupReadsRoute},
	LockRouterGroupWritesRoute:   {Path: "/lock_router_group_writes", Method: "PUT", Name: LockRouterGroupWritesRoute},
	UnlockRouterGroupWritesRoute: {Path: "/unlock_router_group_writes", Method: "PUT", Name: UnlockRouterGroupWritesRoute},
}

func AdminRoutes() rata.Routes {
	var routes rata.Routes
	for _, r := range AdminRoutesMap {
		routes = append(routes, r)
	}

	return routes
}
func NewServer(socket string, db db.DB, logger lager.Logger) ifrit.Runner {
	rglHandler := NewRouterGroupLockHandler(db, logger)
	actions := rata.Handlers{
		LockRouterGroupReadsRoute:    http.HandlerFunc(rglHandler.LockReads),
		UnlockRouterGroupReadsRoute:  http.HandlerFunc(rglHandler.UnlockReads),
		LockRouterGroupWritesRoute:   http.HandlerFunc(rglHandler.LockWrites),
		UnlockRouterGroupWritesRoute: http.HandlerFunc(rglHandler.UnlockWrites),
	}
	handler, err := rata.NewRouter(AdminRoutes(), actions)
	if err != nil {
		logger.Error("failed to create router", err)
		os.Exit(1)
	}

	handler = handlers.LogWrap(handler, logger)
	return http_server.NewUnixServer(socket, handler)
}
