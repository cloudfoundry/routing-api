package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/authentication"
	"github.com/cloudfoundry-incubator/routing-api/config"
	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/handlers"
	"github.com/cloudfoundry-incubator/routing-api/helpers"
	"github.com/cloudfoundry-incubator/routing-api/metrics"
	"github.com/cloudfoundry-incubator/runtime-schema/maintainer"
	"github.com/cloudfoundry/dropsonde"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/quipo/statsd"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

var maxTTL = flag.Int("maxTTL", 120, "Maximum TTL on the route")
var port = flag.Int("port", 8080, "Port to run rounting-api server on")
var configPath = flag.String("config", "", "Configuration for routing-api")
var devMode = flag.Bool("devMode", false, "Disable authentication for easier development iteration")
var ip = flag.String("ip", "", "The public ip of the routing api")
var systemDomain = flag.String("systemDomain", "", "System domain that the routing api should register on")
var consulCluster = flag.String("consulCluster", "", "comma-separated list of consul server URLs (scheme://ip:port)")

func route(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(f)
}

func main() {
	logger := cf_lager.New("routing-api")

	err := checkFlags()
	if err != nil {
		logger.Error("failed to start", err)
		os.Exit(1)
	}

	cfg, err := config.NewConfigFromFile(*configPath)
	if err != nil {
		logger.Error("failed to start", err)
		os.Exit(1)
	}

	err = dropsonde.Initialize(cfg.MetronConfig.Address+":"+cfg.MetronConfig.Port, cfg.LogGuid)
	if err != nil {
		logger.Error("failed to initialize Dropsonde", err)
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

	var token authentication.Token

	if *devMode {
		token = authentication.NullToken{}
	} else {
		token = authentication.NewAccessToken(cfg.UAAPublicKey)
		err = token.CheckPublicToken()
		if err != nil {
			logger.Error("failed to check public token", err)
			os.Exit(1)
		}
	}

	validator := handlers.NewValidator()

	prefix := "routing_api."
	statsdclient := statsd.NewStatsdClient(cfg.StatsdEndpoint, prefix) // make sure you config this yo
	statsdclient.CreateSocket()
	interval := cfg.MetricsReportingInterval
	stats := statsd.NewStatsdBuffer(interval, statsdclient)
	defer stats.Close()

	routesHandler := handlers.NewRoutesHandler(token, *maxTTL, validator, database, logger)
	eventStreamHandler := handlers.NewEventStreamHandler(token, database, logger, stats)

	actions := rata.Handlers{
		"Upsert":      route(routesHandler.Upsert),
		"Delete":      route(routesHandler.Delete),
		"List":        route(routesHandler.List),
		"EventStream": route(eventStreamHandler.EventStream),
	}

	handler, err := rata.NewRouter(routing_api.Routes, actions)
	if err != nil {
		logger.Error("failed to create router", err)
		os.Exit(1)
	}

	handler = handlers.LogWrap(handler, logger)

	registerInterval := *maxTTL / 2

	host := fmt.Sprintf("routing-api.%s", *systemDomain)
	route := db.Route{
		Route:   host,
		Port:    *port,
		IP:      *ip,
		TTL:     *maxTTL,
		LogGuid: cfg.LogGuid,
	}

	ticker := time.NewTicker(time.Duration(registerInterval) * time.Second)
	quitChan := make(chan bool)
	go helpers.RegisterRoutingAPI(quitChan, database, route, ticker, logger)

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1)
	go func() {
		<-c
		logger.Info("Unregistering from etcd")

		quitChan <- true
		<-quitChan

		os.Exit(0)
	}()

	metricsTicker := time.NewTicker(cfg.MetricsReportingInterval)
	consulSession := initializeConsulSession(cfg.ConsulConfig.TTL, logger)
	lock := maintainer.NewLock(
		consulSession,
		"v1/locks/routing-api",
		[]byte("something-else"),
		clock.NewClock(),
		cfg.ConsulConfig.LockRetryInterval,
		logger,
	)

	metricsReporter := metrics.NewMetricsReporter(database, stats, metricsTicker)

	members := grouper.Members{
		{"lock", lock},
		{"metrics", metricsReporter},
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	ifrit.Background(sigmon.New(group))

	logger.Info("starting", lager.Data{"port": *port})
	err = http.ListenAndServe(":"+strconv.Itoa(*port), handler)
	if err != nil {
		panic(err)
	}
}

func initializeConsulSession(lockTTL time.Duration, logger lager.Logger) *consuladapter.Session {
	client, err := consuladapter.NewClient(*consulCluster)
	if err != nil {
		logger.Fatal("new-client-failed", err)
	}

	sessionMgr := consuladapter.NewSessionManager(client)
	consulSession, err := consuladapter.NewSession("routing-api", lockTTL, client, sessionMgr)
	if err != nil {
		logger.Fatal("consul-session-failed", err)
	}

	return consulSession
}

func checkFlags() error {
	flag.Parse()
	if *configPath == "" {
		return errors.New("No configuration file provided")
	}

	if *ip == "" {
		return errors.New("No ip address provided")
	}

	if *systemDomain == "" {
		return errors.New("No system domain provided")
	}

	if *consulCluster == "" {
		return errors.New("No consul cluster provided")
	}

	return nil
}
