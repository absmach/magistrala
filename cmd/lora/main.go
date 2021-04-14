// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	r "github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"
	"github.com/mainflux/mainflux/lora/api"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging/nats"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux/lora/redis"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	defLogLevel       = "error"
	defHTTPPort       = "8180"
	defLoraMsgURL     = "tcp://localhost:1883"
	defSubTimeout     = "30s" // 30 seconds
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
	envSubTimeout     = "MF_LORA_ADAPTER_SUBSCRIBER_TIMEOUT"
	envNatsURL        = "MF_NATS_URL"
	envLogLevel       = "MF_LORA_ADAPTER_LOG_LEVEL"
	envESURL          = "MF_THINGS_ES_URL"
	envESPass         = "MF_THINGS_ES_PASS"
	envESDB           = "MF_THINGS_ES_DB"
	envESConsumerName = "MF_LORA_ADAPTER_EVENT_CONSUMER"
	envRouteMapURL    = "MF_LORA_ADAPTER_ROUTE_MAP_URL"
	envRouteMapPass   = "MF_LORA_ADAPTER_ROUTE_MAP_PASS"
	envRouteMapDB     = "MF_LORA_ADAPTER_ROUTE_MAP_DB"

	loraServerTopic = "application/+/device/+/rx"

	thingsRMPrefix   = "thing"
	channelsRMPrefix = "channel"
	connsRMPrefix    = "connection"
)

type config struct {
	httpPort       string
	loraMsgURL     string
	natsURL        string
	subTimeout     time.Duration
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

	msub, err := mqtt.NewSubscriber(cfg.loraMsgURL, cfg.subTimeout, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create MQTT subscriber: %s", err))
		os.Exit(1)
	}

	go subscribeToLoRaBroker(svc, msub, logger)

	go subscribeToThingsES(svc, esConn, cfg.esConsumerName, logger)

	errs := make(chan error, 2)

	go startHTTPServer(cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("LoRa adapter terminated: %s", err))
}

func loadConfig() config {
	mqttTimeout, err := time.ParseDuration(mainflux.Env(envSubTimeout, defSubTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envSubTimeout, err.Error())
	}
	return config{
		httpPort:       mainflux.Env(envHTTPPort, defHTTPPort),
		loraMsgURL:     mainflux.Env(envLoraMsgURL, defLoraMsgURL),
		subTimeout:     mqttTimeout,
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

func subscribeToLoRaBroker(svc lora.Service, msub messaging.Subscriber, logger logger.Logger) {
	err := msub.Subscribe(loraServerTopic, func(msg messaging.Message) error {
		var m lora.Message
		if err := json.Unmarshal(msg.Payload, &m); err != nil {
			logger.Warn(fmt.Sprintf("Failed to Unmarshal message: %s", err.Error()))
			return err
		}
		if err := svc.Publish(m); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to LoRa MQTT broker: %s", err))
		os.Exit(1)
	}
	logger.Info("Subscribed to LoRa MQTT broker")
}

func subscribeToThingsES(svc lora.Service, client *r.Client, consumer string, logger logger.Logger) {
	eventStore := redis.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	if err := eventStore.Subscribe("mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("LoRa-adapter service failed to subscribe to Redis event source: %s", err))
	}
}

func newRouteMapRepository(client *r.Client, prefix string, logger logger.Logger) lora.RouteMapRepository {
	logger.Info(fmt.Sprintf("Connected to %s Redis Route-map", prefix))
	return redis.NewRouteMapRepository(client, prefix)
}

func startHTTPServer(cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	logger.Info(fmt.Sprintf("LoRa-adapter service started, exposed port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, api.MakeHandler())
}
