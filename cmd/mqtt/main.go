// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains mqtt-adapter main function to start the mqtt-adapter service.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	thingsClient "github.com/mainflux/mainflux/internal/clients/grpc/things"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt"
	mqttredis "github.com/mainflux/mainflux/mqtt/redis"
	mqtttracing "github.com/mainflux/mainflux/mqtt/tracing"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	mqttpub "github.com/mainflux/mainflux/pkg/messaging/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging/tracing"
	mp "github.com/mainflux/mproxy/pkg/mqtt"
	"github.com/mainflux/mproxy/pkg/session"
	ws "github.com/mainflux/mproxy/pkg/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	svcName            = "mqtt"
	envPrefix          = "MF_MQTT_ADAPTER_"
	envPrefixES        = "MF_MQTT_ADAPTER_ES_"
	envPrefixAuthCache = "MF_AUTH_CACHE_"
)

type config struct {
	LogLevel              string        `env:"MF_MQTT_ADAPTER_LOG_LEVEL"                    envDefault:"info"`
	MqttPort              string        `env:"MF_MQTT_ADAPTER_MQTT_PORT"                    envDefault:"1883"`
	MqttTargetHost        string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_HOST"             envDefault:"localhost"`
	MqttTargetPort        string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_PORT"             envDefault:"1883"`
	MqttForwarderTimeout  time.Duration `env:"MF_MQTT_ADAPTER_FORWARDER_TIMEOUT"            envDefault:"30s"`
	MqttTargetHealthCheck string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"     envDefault:""`
	HttpPort              string        `env:"MF_MQTT_ADAPTER_WS_PORT"                      envDefault:"8080"`
	HttpTargetHost        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_HOST"               envDefault:"localhost"`
	HttpTargetPort        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_PORT"               envDefault:"8080"`
	HttpTargetPath        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_PATH"               envDefault:"/mqtt"`
	Instance              string        `env:"MF_MQTT_ADAPTER_INSTANCE"                     envDefault:""`
	JaegerURL             string        `env:"MF_JAEGER_URL"                                envDefault:"http://jaeger:14268/api/traces"`
	BrokerURL             string        `env:"MF_BROKER_URL"                                envDefault:"nats://localhost:4222"`
	SendTelemetry         bool          `env:"MF_SEND_TELEMETRY"                            envDefault:"true"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	if cfg.MqttTargetHealthCheck != "" {
		notify := func(e error, next time.Duration) {
			logger.Info(fmt.Sprintf("Broker not ready: %s, next try in %s", e.Error(), next))
		}

		err := backoff.RetryNotify(healthcheck(cfg), backoff.NewExponentialBackOff(), notify)
		if err != nil {
			logger.Fatal(fmt.Sprintf("MQTT healthcheck limit exceeded, exiting. %s ", err))
		}
	}

	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	nps, err := brokers.NewPubSub(cfg.BrokerURL, "mqtt", logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to message broker: %s", err))
	}
	nps = tracing.NewPubSub(tracer, nps)
	defer nps.Close()

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("%s:%s", cfg.MqttTargetHost, cfg.MqttTargetPort), cfg.MqttForwarderTimeout)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to create MQTT publisher: %s", err))
	}
	mpub = tracing.New(tracer, mpub)

	fwd := mqtt.NewForwarder(brokers.SubjectAllChannels, logger)
	fwd = mqtttracing.New(tracer, fwd, brokers.SubjectAllChannels)
	if err := fwd.Forward(ctx, svcName, nps, mpub); err != nil {
		logger.Fatal(fmt.Sprintf("failed to forward message broker messages: %s", err))
	}

	np, err := brokers.NewPublisher(cfg.BrokerURL)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to message broker: %s", err))
	}
	np = tracing.New(tracer, np)
	defer np.Close()

	ec, err := redisClient.Setup(envPrefixES)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to setup %s event store redis client : %s", svcName, err))
	}
	defer ec.Close()

	es := mqttredis.NewEventStore(ec, cfg.Instance)

	ac, err := redisClient.Setup(envPrefixAuthCache)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to setup %s event store redis client : %s", svcName, err))
	}
	defer ac.Close()

	tc, tcHandler, err := thingsClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer tcHandler.Close()
	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	h := mqtt.NewHandler([]messaging.Publisher{np}, es, logger, tc)
	h = mqtttracing.NewHandler(tracer, h)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.MqttPort))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, logger, h)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.HttpPort))
	g.Go(func() error {
		return proxyWS(ctx, cfg, logger, h)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("mProxy shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
	}
}

func proxyMQTT(ctx context.Context, cfg config, logger mflog.Logger, handler session.Handler) error {
	address := fmt.Sprintf(":%s", cfg.MqttPort)
	target := fmt.Sprintf("%s:%s", cfg.MqttTargetHost, cfg.MqttTargetPort)
	mp := mp.New(address, target, handler, logger)

	errCh := make(chan error)
	go func() {
		errCh <- mp.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}

func proxyWS(ctx context.Context, cfg config, logger mflog.Logger, handler session.Handler) error {
	target := fmt.Sprintf("%s:%s", cfg.HttpTargetHost, cfg.HttpTargetPort)
	wp := ws.New(target, cfg.HttpTargetPath, "ws", handler, logger)
	http.Handle("/mqtt", wp.Handler())

	errCh := make(chan error)

	go func() {
		errCh <- wp.Listen(cfg.HttpPort)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}

func healthcheck(cfg config) func() error {
	return func() error {
		res, err := http.Get(cfg.MqttTargetHealthCheck)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errors.New(string(body))
		}
		return nil
	}
}
