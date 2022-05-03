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

	mqttPaho "github.com/eclipse/paho.mqtt.golang"
	r "github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"
	"github.com/mainflux/mainflux/lora/api"
	"github.com/mainflux/mainflux/lora/mqtt"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"golang.org/x/sync/errgroup"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux/lora/redis"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	stopWaitTime = 5 * time.Second

	defLogLevel       = "error"
	defHTTPPort       = "8180"
	defLoraMsgURL     = "tcp://localhost:1883"
	defLoraMsgTopic   = "application/+/device/+/event/up"
	defLoraMsgUser    = ""
	defLoraMsgPass    = ""
	defLoraMsgTimeout = "30s"
	defNatsURL        = "nats://localhost:4222"
	defESURL          = "localhost:6379"
	defESPass         = ""
	defESDB           = "0"
	defESConsumerName = "lora"
	defRouteMapURL    = "localhost:6379"
	defRouteMapPass   = ""
	defRouteMapDB     = "0"

	envHTTPPort       = "MF_LORA_ADAPTER_HTTP_PORT"
	envLoraMsgURL     = "MF_LORA_ADAPTER_MESSAGES_URL"
	envLoraMsgTopic   = "MF_LORA_ADAPTER_MESSAGES_TOPIC"
	envLoraMsgUser    = "MF_LORA_ADAPTER_MESSAGES_USER"
	envLoraMsgPass    = "MF_LORA_ADAPTER_MESSAGES_PASS"
	envLoraMsgTimeout = "MF_LORA_ADAPTER_MESSAGES_TIMEOUT"
	envNatsURL        = "MF_NATS_URL"
	envLogLevel       = "MF_LORA_ADAPTER_LOG_LEVEL"
	envESURL          = "MF_THINGS_ES_URL"
	envESPass         = "MF_THINGS_ES_PASS"
	envESDB           = "MF_THINGS_ES_DB"
	envESConsumerName = "MF_LORA_ADAPTER_EVENT_CONSUMER"
	envRouteMapURL    = "MF_LORA_ADAPTER_ROUTE_MAP_URL"
	envRouteMapPass   = "MF_LORA_ADAPTER_ROUTE_MAP_PASS"
	envRouteMapDB     = "MF_LORA_ADAPTER_ROUTE_MAP_DB"

	thingsRMPrefix   = "thing"
	channelsRMPrefix = "channel"
	connsRMPrefix    = "connection"
)

type config struct {
	httpPort       string
	loraMsgURL     string
	loraMsgUser    string
	loraMsgPass    string
	loraMsgTopic   string
	loraMsgTimeout time.Duration
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
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	rmConn := connectToRedis(cfg.routeMapURL, cfg.routeMapPass, cfg.routeMapDB, logger)
	defer rmConn.Close()

	esConn := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer esConn.Close()

	pub, err := nats.NewPublisher(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer pub.Close()

	thingsRM := newRouteMapRepository(rmConn, thingsRMPrefix, logger)
	chansRM := newRouteMapRepository(rmConn, channelsRMPrefix, logger)
	connsRM := newRouteMapRepository(rmConn, connsRMPrefix, logger)

	svc := lora.New(pub, thingsRM, chansRM, connsRM)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "lora_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "lora_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	mqttConn := connectToMQTTBroker(cfg.loraMsgURL, cfg.loraMsgUser, cfg.loraMsgPass, cfg.loraMsgTimeout, logger)

	go subscribeToLoRaBroker(svc, mqttConn, cfg.loraMsgTimeout, cfg.loraMsgTopic, logger)
	go subscribeToThingsES(svc, esConn, cfg.esConsumerName, logger)

	g.Go(func() error {
		return startHTTPServer(ctx, cfg, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("LoRa adapter shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("LoRa adapter terminated: %s", err))
	}

}

func loadConfig() config {
	mqttTimeout, err := time.ParseDuration(mainflux.Env(envLoraMsgTimeout, defLoraMsgTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envLoraMsgTimeout, err.Error())
	}

	return config{
		httpPort:       mainflux.Env(envHTTPPort, defHTTPPort),
		loraMsgURL:     mainflux.Env(envLoraMsgURL, defLoraMsgURL),
		loraMsgTopic:   mainflux.Env(envLoraMsgTopic, defLoraMsgTopic),
		loraMsgUser:    mainflux.Env(envLoraMsgUser, defLoraMsgUser),
		loraMsgPass:    mainflux.Env(envLoraMsgPass, defLoraMsgPass),
		loraMsgTimeout: mqttTimeout,
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

func connectToMQTTBroker(url, user, password string, timeout time.Duration, logger logger.Logger) mqttPaho.Client {
	opts := mqttPaho.NewClientOptions()
	opts.AddBroker(url)
	opts.SetUsername(user)
	opts.SetPassword(password)
	opts.SetOnConnectHandler(func(c mqttPaho.Client) {
		logger.Info("Connected to Lora MQTT broker")
	})
	opts.SetConnectionLostHandler(func(c mqttPaho.Client, err error) {
		logger.Error(fmt.Sprintf("MQTT connection lost: %s", err.Error()))
		os.Exit(1)
	})

	client := mqttPaho.NewClient(opts)

	if token := client.Connect(); token.WaitTimeout(timeout) && token.Error() != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Lora MQTT broker: %s", token.Error()))
		os.Exit(1)
	}

	return client
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

func subscribeToLoRaBroker(svc lora.Service, mc mqttPaho.Client, timeout time.Duration, topic string, logger logger.Logger) {
	mqtt := mqtt.NewBroker(svc, mc, timeout, logger)
	logger.Info("Subscribed to Lora MQTT broker")
	if err := mqtt.Subscribe(topic); err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to Lora MQTT broker: %s", err))
		os.Exit(1)
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

func startHTTPServer(ctx context.Context, cfg config, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler()}

	logger.Info(fmt.Sprintf("LoRa-adapter service started, exposed port %s", cfg.httpPort))

	go func() {
		errCh <- http.ListenAndServe(p, api.MakeHandler())
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("LoRa-adapter service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("LoRa-adapter service error occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("LoRa-adapter service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}

}
