// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains lora main function to start the lora service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal"
	"github.com/absmach/magistrala/internal/clients/jaeger"
	redisclient "github.com/absmach/magistrala/internal/clients/redis"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/lora"
	"github.com/absmach/magistrala/lora/api"
	"github.com/absmach/magistrala/lora/events"
	"github.com/absmach/magistrala/lora/mqtt"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	mqttpaho "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "lora-adapter"
	envPrefixHTTP  = "MG_LORA_ADAPTER_HTTP_"
	defSvcHTTPPort = "9017"

	thingsRMPrefix   = "thing"
	channelsRMPrefix = "channel"
	connsRMPrefix    = "connection"
	thingsStream     = "magistrala.things"
)

type config struct {
	LogLevel       string        `env:"MG_LORA_ADAPTER_LOG_LEVEL"           envDefault:"info"`
	LoraMsgURL     string        `env:"MG_LORA_ADAPTER_MESSAGES_URL"        envDefault:"tcp://localhost:1883"`
	LoraMsgUser    string        `env:"MG_LORA_ADAPTER_MESSAGES_USER"       envDefault:""`
	LoraMsgPass    string        `env:"MG_LORA_ADAPTER_MESSAGES_PASS"       envDefault:""`
	LoraMsgTopic   string        `env:"MG_LORA_ADAPTER_MESSAGES_TOPIC"      envDefault:"application/+/device/+/event/up"`
	LoraMsgTimeout time.Duration `env:"MG_LORA_ADAPTER_MESSAGES_TIMEOUT"    envDefault:"30s"`
	ESConsumerName string        `env:"MG_LORA_ADAPTER_EVENT_CONSUMER"      envDefault:"lora-adapter"`
	BrokerURL      string        `env:"MG_MESSAGE_BROKER_URL"               envDefault:"nats://localhost:4222"`
	JaegerURL      url.URL       `env:"MG_JAEGER_URL"                       envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry  bool          `env:"MG_SEND_TELEMETRY"                   envDefault:"true"`
	InstanceID     string        `env:"MG_LORA_ADAPTER_INSTANCE_ID"         envDefault:""`
	ESURL          string        `env:"MG_ES_URL"                           envDefault:"nats://localhost:4222"`
	RouteMapURL    string        `env:"MG_LORA_ADAPTER_ROUTE_MAP_URL"       envDefault:"redis://localhost:6379/0"`
	TraceRatio     float64       `env:"MG_JAEGER_TRACE_RATIO"               envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	rmConn, err := redisclient.Connect(cfg.RouteMapURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup route map redis client : %s", err))
		exitCode = 1
		return
	}
	defer rmConn.Close()

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	pub, err := brokers.NewPublisher(ctx, cfg.BrokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pub.Close()
	pub = brokerstracing.NewPublisher(httpServerConfig, tracer, pub)

	svc := newService(pub, rmConn, thingsRMPrefix, channelsRMPrefix, connsRMPrefix, logger)

	mqttConn, err := connectToMQTTBroker(cfg.LoraMsgURL, cfg.LoraMsgUser, cfg.LoraMsgPass, cfg.LoraMsgTimeout, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}

	if err = subscribeToLoRaBroker(svc, mqttConn, cfg.LoraMsgTimeout, cfg.LoraMsgTopic, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to Lora MQTT broker: %s", err))
		exitCode = 1
		return
	}

	if err = subscribeToThingsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to things event store: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("LoRa adapter terminated: %s", err))
	}
}

func connectToMQTTBroker(burl, user, password string, timeout time.Duration, logger *slog.Logger) (mqttpaho.Client, error) {
	opts := mqttpaho.NewClientOptions()
	opts.AddBroker(burl)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetOnConnectHandler(func(_ mqttpaho.Client) {
		logger.Info("Connected to Lora MQTT broker")
	})
	opts.SetConnectionLostHandler(func(_ mqttpaho.Client, err error) {
		logger.Error(fmt.Sprintf("MQTT connection lost: %s", err))
	})

	client := mqttpaho.NewClient(opts)

	if token := client.Connect(); token.WaitTimeout(timeout) && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to Lora MQTT broker: %s", token.Error())
	}

	return client, nil
}

func subscribeToLoRaBroker(svc lora.Service, mc mqttpaho.Client, timeout time.Duration, topic string, logger *slog.Logger) error {
	mqttBroker := mqtt.NewBroker(svc, mc, timeout, logger)
	logger.Info("Subscribed to Lora MQTT broker")
	if err := mqttBroker.Subscribe(topic); err != nil {
		return fmt.Errorf("failed to subscribe to Lora MQTT broker: %s", err)
	}
	return nil
}

func subscribeToThingsES(ctx context.Context, svc lora.Service, cfg config, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, cfg.ESURL, thingsStream, cfg.ESConsumerName, logger)
	if err != nil {
		return err
	}

	handler := events.NewEventHandler(svc)

	logger.Info("Subscribed to Redis Event Store")

	return subscriber.Subscribe(ctx, handler)
}

func newRouteMapRepository(client *redis.Client, prefix string, logger *slog.Logger) lora.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return events.NewRouteMapRepository(client, prefix)
}

func newService(pub messaging.Publisher, rmConn *redis.Client, thingsRMPrefix, channelsRMPrefix, connsRMPrefix string, logger *slog.Logger) lora.Service {
	thingsRM := newRouteMapRepository(rmConn, thingsRMPrefix, logger)
	chansRM := newRouteMapRepository(rmConn, channelsRMPrefix, logger)
	connsRM := newRouteMapRepository(rmConn, connsRMPrefix, logger)

	svc := lora.New(pub, thingsRM, chansRM, connsRM)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("lora_adapter", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
