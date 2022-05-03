// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	r "github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/opcua/api"
	"github.com/mainflux/mainflux/opcua/db"
	"github.com/mainflux/mainflux/opcua/gopcua"
	"github.com/mainflux/mainflux/opcua/redis"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"golang.org/x/sync/errgroup"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	stopWaitTime = 5 * time.Second

	defLogLevel       = "error"
	defHTTPPort       = "8180"
	defOPCIntervalMs  = "1000"
	defOPCPolicy      = ""
	defOPCMode        = ""
	defOPCCertFile    = ""
	defOPCKeyFile     = ""
	defNatsURL        = "nats://localhost:4222"
	defESURL          = "localhost:6379"
	defESPass         = ""
	defESDB           = "0"
	defESConsumerName = "opcua"
	defRouteMapURL    = "localhost:6379"
	defRouteMapPass   = ""
	defRouteMapDB     = "0"

	envLogLevel       = "MF_OPCUA_ADAPTER_LOG_LEVEL"
	envHTTPPort       = "MF_OPCUA_ADAPTER_HTTP_PORT"
	envOPCIntervalMs  = "MF_OPCUA_ADAPTER_INTERVAL_MS"
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

	thingsRMPrefix     = "thing"
	channelsRMPrefix   = "channel"
	connectionRMPrefix = "connection"
)

type config struct {
	httpPort       string
	opcuaConfig    opcua.Config
	natsURL        string
	logLevel       string
	esURL          string
	esPass         string
	esDB           string
	esConsumerName string
	routeMapURL    string
	routeMapPass   string
	routeMapDB     string
}

func main() {
	cfg := loadConfig()
	httpCtx, httpCancel := context.WithCancel(context.Background())
	g, httpCtx := errgroup.WithContext(httpCtx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	rmConn := connectToRedis(cfg.routeMapURL, cfg.routeMapPass, cfg.routeMapDB, logger)
	defer rmConn.Close()

	thingRM := newRouteMapRepositoy(rmConn, thingsRMPrefix, logger)
	chanRM := newRouteMapRepositoy(rmConn, channelsRMPrefix, logger)
	connRM := newRouteMapRepositoy(rmConn, connectionRMPrefix, logger)

	esConn := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer esConn.Close()

	pubSub, err := nats.NewPubSub(cfg.natsURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	ctx := context.Background()
	sub := gopcua.NewSubscriber(ctx, pubSub, thingRM, chanRM, connRM, logger)
	browser := gopcua.NewBrowser(ctx, logger)

	svc := opcua.New(sub, browser, thingRM, chanRM, connRM, cfg.opcuaConfig, logger)
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

	go subscribeToStoredSubs(sub, cfg.opcuaConfig, logger)
	go subscribeToThingsES(svc, esConn, cfg.esConsumerName, logger)

	g.Go(func() error {
		return startHTTPServer(httpCtx, svc, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(httpCtx); sig != nil {
			httpCancel()
			logger.Info(fmt.Sprintf("OPC-UA adapter service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("OPC-UA adapter service terminated: %s", err))
	}
}

func loadConfig() config {
	oc := opcua.Config{
		Interval: mainflux.Env(envOPCIntervalMs, defOPCIntervalMs),
		Policy:   mainflux.Env(envOPCPolicy, defOPCPolicy),
		Mode:     mainflux.Env(envOPCMode, defOPCMode),
		CertFile: mainflux.Env(envOPCCertFile, defOPCCertFile),
		KeyFile:  mainflux.Env(envOPCKeyFile, defOPCKeyFile),
	}
	return config{
		httpPort:       mainflux.Env(envHTTPPort, defHTTPPort),
		opcuaConfig:    oc,
		natsURL:        mainflux.Env(envNatsURL, defNatsURL),
		logLevel:       mainflux.Env(envLogLevel, defLogLevel),
		esURL:          mainflux.Env(envESURL, defESURL),
		esPass:         mainflux.Env(envESPass, defESPass),
		esDB:           mainflux.Env(envESDB, defESDB),
		esConsumerName: mainflux.Env(envESConsumerName, defESConsumerName),
		routeMapURL:    mainflux.Env(envRouteMapURL, defRouteMapURL),
		routeMapPass:   mainflux.Env(envRouteMapPass, defRouteMapPass),
		routeMapDB:     mainflux.Env(envRouteMapDB, defRouteMapDB),
	}
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

func subscribeToStoredSubs(sub opcua.Subscriber, cfg opcua.Config, logger logger.Logger) {
	// Get all stored subscriptions
	nodes, err := db.ReadAll()
	if err != nil {
		logger.Warn(fmt.Sprintf("Read stored subscriptions failed: %s", err))
	}

	for _, n := range nodes {
		cfg.ServerURI = n.ServerURI
		cfg.NodeID = n.NodeID
		go func() {
			if err := sub.Subscribe(context.Background(), cfg); err != nil {
				logger.Warn(fmt.Sprintf("Subscription failed: %s", err))
			}
		}()
	}
}

func subscribeToOpcuaServer(gc opcua.Subscriber, cfg opcua.Config, logger logger.Logger) {
	if err := gc.Subscribe(context.Background(), cfg); err != nil {
		logger.Warn(fmt.Sprintf("OPC-UA Subscription failed: %s", err))
	}
}

func subscribeToThingsES(svc opcua.Service, client *r.Client, prefix string, logger logger.Logger) {
	eventStore := redis.NewEventStore(svc, client, prefix, logger)
	if err := eventStore.Subscribe(context.Background(), "mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Failed to subscribe to Redis event source: %s", err))
	}
}

func newRouteMapRepositoy(client *r.Client, prefix string, logger logger.Logger) opcua.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return redis.NewRouteMapRepository(client, prefix)
}

func startHTTPServer(ctx context.Context, svc opcua.Service, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.httpPort)

	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svc, logger)}
	logger.Info(fmt.Sprintf("OPC-UA adapter service started, exposed port %s", cfg.httpPort))

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("OPC-UA adapter service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("OPC-UA adapter service error occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("OPC-UA adapter service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
