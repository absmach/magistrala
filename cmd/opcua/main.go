// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	r "github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/internal"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/opcua/api"
	"github.com/mainflux/mainflux/opcua/db"
	"github.com/mainflux/mainflux/opcua/gopcua"
	"github.com/mainflux/mainflux/opcua/redis"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "opc-ua-adapter"
	envPrefix         = "MF_OPCUA_ADAPTER_"
	envPrefixES       = "MF_OPCUA_ADAPTER_ES_"
	envPrefixHttp     = "MF_OPCUA_ADAPTER_HTTP_"
	envPrefixRouteMap = "MF_OPCUA_ADAPTER_ROUTE_MAP_"
	defSvcHttpPort    = "8180"

	thingsRMPrefix     = "thing"
	channelsRMPrefix   = "channel"
	connectionRMPrefix = "connection"
)

type config struct {
	LogLevel       string `env:"MF_OPCUA_ADAPTER_LOG_LEVEL"          envDefault:"info"`
	ESConsumerName string `env:"MF_OPCUA_ADAPTER_EVENT_CONSUMER"     envDefault:""`
	BrokerURL      string `env:"MF_BROKER_URL"                       envDefault:"nats://localhost:4222"`
}

func main() {
	httpCtx, httpCancel := context.WithCancel(context.Background())
	g, httpCtx := errgroup.WithContext(httpCtx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	opcConfig := opcua.Config{}
	if err := env.Parse(&opcConfig); err != nil {
		log.Fatalf("failed to load %s opcua client configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	rmConn, err := redisClient.Setup(envPrefixRouteMap)
	if err != nil {
		log.Fatalf("failed to setup %s bootstrap event store redis client : %s", svcName, err.Error())
	}
	defer rmConn.Close()

	thingRM := newRouteMapRepositoy(rmConn, thingsRMPrefix, logger)
	chanRM := newRouteMapRepositoy(rmConn, channelsRMPrefix, logger)
	connRM := newRouteMapRepositoy(rmConn, connectionRMPrefix, logger)

	esConn, err := redisClient.Setup(envPrefixES)
	if err != nil {
		log.Fatalf("failed to setup %s bootstrap event store redis client : %s", svcName, err.Error())
	}
	defer esConn.Close()

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		log.Fatalf("failed to connect to message broker: %s", err.Error())
	}
	defer pubSub.Close()

	ctx := context.Background()
	sub := gopcua.NewSubscriber(ctx, pubSub, thingRM, chanRM, connRM, logger)
	browser := gopcua.NewBrowser(ctx, logger)

	svc := newService(sub, browser, thingRM, chanRM, connRM, opcConfig, logger)

	go subscribeToStoredSubs(sub, opcConfig, logger)
	go subscribeToThingsES(svc, esConn, cfg.ESConsumerName, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(httpCtx, httpCancel, svcName, httpServerConfig, api.MakeHandler(svc, logger), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(httpCtx, httpCancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("OPC-UA adapter service terminated: %s", err))
	}
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

func newService(sub opcua.Subscriber, browser opcua.Browser, thingRM, chanRM, connRM opcua.RouteMapRepository, opcuaConfig opcua.Config, logger logger.Logger) opcua.Service {
	svc := opcua.New(sub, browser, thingRM, chanRM, connRM, opcuaConfig, logger)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
