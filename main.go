package main

import (
	"flag"
	"net/http"
	"strconv"

	"github.com/pivotal-cf-experimental/routing-api/handlers"
	"github.com/pivotal-golang/lager"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/tedsuo/rata"
)

var Routes = rata.Routes{
	{Path: "/v1/routes", Method: "POST", Name: "Routes"},
}

var serverAddress = flag.String(
	"address",
	"",
	"The host:port that the server is bound to.",
)

var maxTTL = flag.Int("maxTTL", 120, "Maximum TTL on the route")
var port = flag.Int("port", 8080, "Port to run rounting-api server on")

func route(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(f)
}

func main() {
	flag.Parse()

	logger := cf_lager.New("routing-api")
	routesHandler := handlers.NewRoutesHandler(*maxTTL, logger)

	actions := rata.Handlers{
		"Routes": route(routesHandler.Routes),
	}

	handler, err := rata.NewRouter(Routes, actions)
	if err != nil {
		panic("unable to create router: " + err.Error())
	}

	handler = handlers.LogWrap(handler, logger)

	logger.Info("starting", lager.Data{"port": *port})
	err = http.ListenAndServe(":"+strconv.Itoa(*port), handler)
	if err != nil {
		panic(err)
	}
}
