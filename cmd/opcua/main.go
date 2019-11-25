// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	r "github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/opcua/api"
	"github.com/mainflux/mainflux/opcua/gopcua"
	pub "github.com/mainflux/mainflux/opcua/nats"
	"github.com/mainflux/mainflux/opcua/redis"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	nats "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	defHTTPPort       = "8188"
	defOPCPolicy      = ""
	defOPCMode        = ""
	defOPCCertFile    = ""
	defOPCKeyFile     = ""
	defNatsURL        = nats.DefaultURL
	defLogLevel       = "debug"
	defESURL          = "localhost:6379"
	defESPass         = ""
	defESDB           = "0"
	defESConsumerName = "opcua"
	defRouteMapURL    = "localhost:6379"
	defRouteMapPass   = ""
	defRouteMapDB     = "0"
	defNodesConfig    = "/nodes.csv"

	envHTTPPort       = "MF_OPCUA_ADAPTER_HTTP_PORT"
	envLogLevel       = "MF_OPCUA_ADAPTER_LOG_LEVEL"
	envOPCPolicy      = "MF_OPCUA_ADAPTER_POLICY"
	envOPCMode        = "MF_OPCUA_ADAPTER_MODE"
	envOPCCertFile    = "MF_OPCUA_ADAPTER_CERT_FILE"
	envOPCKeyFile     = "MF_OPCUA_ADAPTER_KEY_FILE"
	envNatsURL        = "MF_NATS_URL"
	envESURL          = "MF_THINGS_ES_URL"
	envESPass         = "MF_THINGS_ES_PASS"
	envESDB           = "MF_THINGS_ES_DB"
	envESConsumerName = "MF_OPCUA_ADAPTER_EVENT_CONSUMER"
	envRouteMapURL    = "MF_OPCUA_ADAPTER_ROUTE_MAP_URL"
	envRouteMapPass   = "MF_OPCUA_ADAPTER_ROUTE_MAP_PASS"
	envRouteMapDB     = "MF_OPCUA_ADAPTER_ROUTE_MAP_DB"
	envNodesConfig    = "/nodes.csv"

	thingsRMPrefix   = "thing"
	channelsRMPrefix = "channel"
)

type config struct {
	httpPort       string
	opcConfig      opcua.Config
	natsURL        string
	logLevel       string
	esURL          string
	esPass         string
	esDB           string
	esConsumerName string
	routeMapURL    string
	routeMapPass   string
	routeMapDB     string
	nodesConfig    string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	natsConn := connectToNATS(cfg.natsURL, logger)
	defer natsConn.Close()

	rmConn := connectToRedis(cfg.routeMapURL, cfg.routeMapPass, cfg.routeMapDB, logger)
	defer rmConn.Close()

	esConn := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer esConn.Close()

	publisher := pub.NewMessagePublisher(natsConn)

	thingRM := newRouteMapRepositoy(rmConn, thingsRMPrefix, logger)
	chanRM := newRouteMapRepositoy(rmConn, channelsRMPrefix, logger)

	svc := opcua.New(publisher, thingRM, chanRM)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "opc_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "opc_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	go subscribeToOpcuaServers(svc, cfg.nodesConfig, cfg.opcConfig, logger)
	go subscribeToThingsES(svc, esConn, cfg.esConsumerName, logger)

	errs := make(chan error, 2)

	go startHTTPServer(cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("OPC-UA adapter terminated: %s", err))
}

func loadConfig() config {
	oc := opcua.Config{
		Policy:   mainflux.Env(envOPCPolicy, defOPCPolicy),
		Mode:     mainflux.Env(envOPCMode, defOPCMode),
		CertFile: mainflux.Env(envOPCCertFile, defOPCCertFile),
		KeyFile:  mainflux.Env(envOPCKeyFile, defOPCKeyFile),
	}
	return config{
		httpPort:       mainflux.Env(envHTTPPort, defHTTPPort),
		opcConfig:      oc,
		natsURL:        mainflux.Env(envNatsURL, defNatsURL),
		logLevel:       mainflux.Env(envLogLevel, defLogLevel),
		esURL:          mainflux.Env(envESURL, defESURL),
		esPass:         mainflux.Env(envESPass, defESPass),
		esDB:           mainflux.Env(envESDB, defESDB),
		esConsumerName: mainflux.Env(envESConsumerName, defESConsumerName),
		routeMapURL:    mainflux.Env(envRouteMapURL, defRouteMapURL),
		routeMapPass:   mainflux.Env(envRouteMapPass, defRouteMapPass),
		routeMapDB:     mainflux.Env(envRouteMapDB, defRouteMapDB),
		nodesConfig:    mainflux.Env(envNodesConfig, defNodesConfig),
	}
}

func connectToNATS(url string, logger logger.Logger) *nats.Conn {
	conn, err := nats.Connect(url)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}

	logger.Info("Connected to NATS")
	return conn
}

func connectToRedis(redisURL, redisPass, redisDB string, logger logger.Logger) *r.Client {
	db, err := strconv.Atoi(redisDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return r.NewClient(&r.Options{
		Addr:     redisURL,
		Password: redisPass,
		DB:       db,
	})
}

func readFromOpcuaServer(svc opcua.Service, cfg opcua.Config, logger logger.Logger) {
	ctx := context.Background()
	gr := gopcua.NewReader(ctx, svc, logger)
	if err := gr.Read(cfg); err != nil {
		logger.Warn(fmt.Sprintf("OPC-UA Read failed: %s", err))
	}
}

func subscribeToOpcuaServers(svc opcua.Service, nodes string, cfg opcua.Config, logger logger.Logger) {
	if _, err := os.Stat(nodes); os.IsNotExist(err) {
		logger.Warn(fmt.Sprintf("Config file not found: %s", err))
		return
	}

	file, err := os.OpenFile(nodes, os.O_RDONLY, os.ModePerm)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to open config file: %s", err))
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	ctx := context.Background()
	gc := gopcua.NewClient(ctx, svc, logger)

	for {
		l, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to read config file: %s", err))
			return
		}

		if len(l) < 3 {
			logger.Warn(fmt.Sprintf("Empty or incomplete line found in file"))
			return
		}

		cfg.ServerURI = l[0]
		cfg.NodeNamespace = l[1]
		cfg.NodeIdintifier = l[2]

		go subscribeToOpcuaServer(gc, cfg, logger)
	}
}

func subscribeToOpcuaServer(gc opcua.Subscriber, cfg opcua.Config, logger logger.Logger) {
	if err := gc.Subscribe(cfg); err != nil {
		logger.Warn(fmt.Sprintf("OPC-UA Subscription failed: %s", err))
	}
}

func subscribeToThingsES(svc opcua.Service, client *r.Client, prefix string, logger logger.Logger) {
	eventStore := redis.NewEventStore(svc, client, prefix, logger)
	if err := eventStore.Subscribe("mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Failed to subscribe to Redis event source: %s", err))
	}
}

func newRouteMapRepositoy(client *r.Client, prefix string, logger logger.Logger) opcua.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return redis.NewRouteMapRepository(client, prefix)
}

func startHTTPServer(cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	logger.Info(fmt.Sprintf("opcua-adapter service started, exposed port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, api.MakeHandler())
}
