package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/routing-api/authentication"
	"github.com/cloudfoundry-incubator/routing-api/config"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/handlers"
	"github.com/cloudfoundry/dropsonde"
	"github.com/pivotal-golang/lager"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/tedsuo/rata"
)

var Routes = rata.Routes{
	{Path: "/v1/routes", Method: "POST", Name: "Upsert"},
	{Path: "/v1/routes", Method: "DELETE", Name: "Delete"},
	{Path: "/v1/routes", Method: "GET", Name: "List"},
	{Path: "/v1/events", Method: "GET", Name: "EventStream"},
}

var maxTTL = flag.Int("maxTTL", 120, "Maximum TTL on the route")
var port = flag.Int("port", 8080, "Port to run rounting-api server on")
var cfg_flag = flag.String("config", "", "Configuration for routing-api")

func route(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(f)
}

func main() {
	logger := cf_lager.New("routing-api")

	flag.Parse()
	if *cfg_flag == "" {
		logger.Error("starting", errors.New("No configuration file provided"))
		os.Exit(1)
	}

	cfg, err := config.NewConfigFromFile(*cfg_flag)
	if err != nil {
		logger.Error("starting", err)
		os.Exit(1)
	}

	err = dropsonde.Initialize(cfg.MetronConfig.Address+":"+cfg.MetronConfig.Port, cfg.LogGuid)
	if err != nil {
		logger.Error("Dropsonde failed to initialize:", err)
		os.Exit(1)
	}

	logger.Info("database", lager.Data{"etcd-addresses": flag.Args()})
	database := db.NewETCD(flag.Args())
	err = database.Connect()
	if err != nil {
		logger.Error("failed to connect to etcd", err)
		os.Exit(1)
	}
	defer database.Disconnect()

	token := authentication.NewAccessToken(cfg.UAAPublicKey)
	err = token.CheckPublicToken()
	if err != nil {
		logger.Error("starting", err)
		os.Exit(1)
	}

	validator := handlers.NewValidator()

	routesHandler := handlers.NewRoutesHandler(token, *maxTTL, validator, database, logger)
	eventStreamHandler := handlers.NewEventStreamHandler(token, database, logger)

	actions := rata.Handlers{
		"Upsert":      route(routesHandler.Upsert),
		"Delete":      route(routesHandler.Delete),
		"List":        route(routesHandler.List),
		"EventStream": route(eventStreamHandler.EventStream),
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
