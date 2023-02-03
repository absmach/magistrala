// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	mqttPaho "github.com/eclipse/paho.mqtt.golang"
	r "github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/internal"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"
	"github.com/mainflux/mainflux/lora/api"
	"github.com/mainflux/mainflux/lora/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"golang.org/x/sync/errgroup"

	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/lora/redis"
)

const (
	svcName           = "lora-adapter"
	envPrefix         = "MF_LORA_ADAPTER_"
	envPrefixHttp     = "MF_LORA_ADAPTER_HTTP_"
	envPrefixRouteMap = "MF_LORA_ADAPTER_ROUTE_MAP_"
	envPrefixThingsES = "MF_THINGS_ES_"
	defSvcHttpPort    = "8180"

	thingsRMPrefix   = "thing"
	channelsRMPrefix = "channel"
	connsRMPrefix    = "connection"
)

type config struct {
	LogLevel       string        `env:"MF_BOOTSTRAP_LOG_LEVEL"              envDefault:"info"`
	LoraMsgURL     string        `env:"MF_LORA_ADAPTER_MESSAGES_URL"        envDefault:"tcp://localhost:1883"`
	LoraMsgUser    string        `env:"MF_LORA_ADAPTER_MESSAGES_USER"       envDefault:""`
	LoraMsgPass    string        `env:"MF_LORA_ADAPTER_MESSAGES_PASS"       envDefault:""`
	LoraMsgTopic   string        `env:"MF_LORA_ADAPTER_MESSAGES_TOPIC"      envDefault:"application/+/device/+/event/up"`
	LoraMsgTimeout time.Duration `env:"MF_LORA_ADAPTER_MESSAGES_TIMEOUT"    envDefault:"30s"`
	ESConsumerName string        `env:"MF_LORA_ADAPTER_EVENT_CONSUMER"      envDefault:"lora"`
	BrokerURL      string        `env:"MF_BROKER_URL"                       envDefault:"nats://localhost:4222"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	rmConn, err := redisClient.Setup(envPrefixRouteMap)
	if err != nil {
		log.Fatalf("failed to setup route map redis client : %s", err.Error())
	}
	defer rmConn.Close()

	pub, err := brokers.NewPublisher(cfg.BrokerURL)
	if err != nil {
		log.Fatalf("failed to connect to message broker: %s", err.Error())
	}
	defer pub.Close()

	svc := newService(pub, rmConn, thingsRMPrefix, channelsRMPrefix, connsRMPrefix, logger)

	esConn, err := redisClient.Setup(envPrefixThingsES)
	if err != nil {
		log.Fatalf("failed to setup things event store redis client : %s", err.Error())
	}
	defer esConn.Close()

	mqttConn := connectToMQTTBroker(cfg.LoraMsgURL, cfg.LoraMsgUser, cfg.LoraMsgPass, cfg.LoraMsgTimeout, logger)

	go subscribeToLoRaBroker(svc, mqttConn, cfg.LoraMsgTimeout, cfg.LoraMsgTopic, logger)
	go subscribeToThingsES(svc, esConn, cfg.ESConsumerName, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(), logger)

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

func connectToMQTTBroker(url, user, password string, timeout time.Duration, logger logger.Logger) mqttPaho.Client {
	opts := mqttPaho.NewClientOptions()
	opts.AddBroker(url)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetOnConnectHandler(func(c mqttPaho.Client) {
		logger.Info("Connected to Lora MQTT broker")
	})
	opts.SetConnectionLostHandler(func(c mqttPaho.Client, err error) {
		log.Fatalf("MQTT connection lost: %s", err.Error())
	})

	client := mqttPaho.NewClient(opts)

	if token := client.Connect(); token.WaitTimeout(timeout) && token.Error() != nil {
		log.Fatalf("failed to connect to Lora MQTT broker: %s", token.Error())
	}

	return client
}

func subscribeToLoRaBroker(svc lora.Service, mc mqttPaho.Client, timeout time.Duration, topic string, logger logger.Logger) {
	mqtt := mqtt.NewBroker(svc, mc, timeout, logger)
	logger.Info("Subscribed to Lora MQTT broker")
	if err := mqtt.Subscribe(topic); err != nil {
		log.Fatalf("failed to subscribe to Lora MQTT broker: %s", err.Error())
	}
}

func subscribeToThingsES(svc lora.Service, client *r.Client, consumer string, logger logger.Logger) {
	eventStore := redis.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	if err := eventStore.Subscribe(context.Background(), "mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Lora-adapter service failed to subscribe to Redis event source: %s", err))
	}
}

func newRouteMapRepository(client *r.Client, prefix string, logger logger.Logger) lora.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return redis.NewRouteMapRepository(client, prefix)
}

func newService(pub messaging.Publisher, rmConn *r.Client, thingsRMPrefix, channelsRMPrefix, connsRMPrefix string, logger logger.Logger) lora.Service {
	thingsRM := newRouteMapRepository(rmConn, thingsRMPrefix, logger)
	chansRM := newRouteMapRepository(rmConn, channelsRMPrefix, logger)
	connsRM := newRouteMapRepository(rmConn, connsRMPrefix, logger)

	svc := lora.New(pub, thingsRM, chansRM, connsRM)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
