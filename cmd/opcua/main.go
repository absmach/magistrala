// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains opcua-adapter main function to start the opcua-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	redisclient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
	"github.com/mainflux/mainflux/opcua/api"
	"github.com/mainflux/mainflux/opcua/db"
	"github.com/mainflux/mainflux/opcua/events"
	"github.com/mainflux/mainflux/opcua/gopcua"
	mfredis "github.com/mainflux/mainflux/pkg/events/redis"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	brokerstracing "github.com/mainflux/mainflux/pkg/messaging/brokers/tracing"
	"github.com/mainflux/mainflux/pkg/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "opc-ua-adapter"
	envPrefixHTTP     = "MF_OPCUA_ADAPTER_HTTP_"
	envPrefixRouteMap = "MF_OPCUA_ADAPTER_ROUTE_MAP_"
	defSvcHTTPPort    = "8180"

	thingsRMPrefix     = "thing"
	channelsRMPrefix   = "channel"
	connectionRMPrefix = "connection"

	thingsStream = "mainflux.things"
)

type config struct {
	LogLevel       string `env:"MF_OPCUA_ADAPTER_LOG_LEVEL"          envDefault:"info"`
	ESConsumerName string `env:"MF_OPCUA_ADAPTER_EVENT_CONSUMER"     envDefault:"opcua-adapter"`
	BrokerURL      string `env:"MF_BROKER_URL"                       envDefault:"nats://localhost:4222"`
	JaegerURL      string `env:"MF_JAEGER_URL"                       envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry  bool   `env:"MF_SEND_TELEMETRY"                   envDefault:"true"`
	InstanceID     string `env:"MF_OPCUA_ADAPTER_INSTANCE_ID"        envDefault:""`
	ESURL          string `env:"MF_OPCUA_ADAPTER_ES_URL"             envDefault:"redis://localhost:6379/0"`
}

func main() {
	ctx, httpCancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	opcConfig := opcua.Config{}
	if err := env.Parse(&opcConfig); err != nil {
		log.Fatalf("failed to load %s opcua client configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	rmConn, err := redisclient.Setup(envPrefixRouteMap)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup %s bootstrap event store redis client : %s", svcName, err))
		exitCode = 1
		return
	}
	defer rmConn.Close()

	thingRM := newRouteMapRepositoy(rmConn, thingsRMPrefix, logger)
	chanRM := newRouteMapRepositoy(rmConn, channelsRMPrefix, logger)
	connRM := newRouteMapRepositoy(rmConn, connectionRMPrefix, logger)

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	sub := gopcua.NewSubscriber(ctx, pubSub, thingRM, chanRM, connRM, logger)
	browser := gopcua.NewBrowser(ctx, logger)

	svc := newService(sub, browser, thingRM, chanRM, connRM, opcConfig, logger)

	go subscribeToStoredSubs(ctx, sub, opcConfig, logger)

	if err = subscribeToThingsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to things event store: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, httpCancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, httpCancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, httpCancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("OPC-UA adapter service terminated: %s", err))
	}
}

func subscribeToStoredSubs(ctx context.Context, sub opcua.Subscriber, cfg opcua.Config, logger mflog.Logger) {
	// Get all stored subscriptions
	nodes, err := db.ReadAll()
	if err != nil {
		logger.Warn(fmt.Sprintf("Read stored subscriptions failed: %s", err))
	}

	for _, n := range nodes {
		cfg.ServerURI = n.ServerURI
		cfg.NodeID = n.NodeID
		go func() {
			if err := sub.Subscribe(ctx, cfg); err != nil {
				logger.Warn(fmt.Sprintf("Subscription failed: %s", err))
			}
		}()
	}
}

func subscribeToThingsES(ctx context.Context, svc opcua.Service, cfg config, logger mflog.Logger) error {
	subscriber, err := mfredis.NewSubscriber(cfg.ESURL, thingsStream, cfg.ESConsumerName, logger)
	if err != nil {
		return err
	}

	handler := events.NewEventHandler(svc)

	logger.Info("Subscribed to Redis Event Store")

	return subscriber.Subscribe(ctx, handler)
}

func newRouteMapRepositoy(client *redis.Client, prefix string, logger mflog.Logger) opcua.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return events.NewRouteMapRepository(client, prefix)
}

func newService(sub opcua.Subscriber, browser opcua.Browser, thingRM, chanRM, connRM opcua.RouteMapRepository, opcuaConfig opcua.Config, logger mflog.Logger) opcua.Service {
	svc := opcua.New(sub, browser, thingRM, chanRM, connRM, opcuaConfig, logger)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("opc_ua_adapter", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
