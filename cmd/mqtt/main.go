// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains mqtt-adapter main function to start the mqtt-adapter service.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/absmach/magistrala"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	"github.com/absmach/magistrala/internal/server"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/mqtt"
	"github.com/absmach/magistrala/mqtt/events"
	mqtttracing "github.com/absmach/magistrala/mqtt/tracing"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/messaging/handler"
	mqttpub "github.com/absmach/magistrala/pkg/messaging/mqtt"
	"github.com/absmach/magistrala/pkg/uuid"
	mp "github.com/absmach/mproxy/pkg/mqtt"
	"github.com/absmach/mproxy/pkg/mqtt/websocket"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/caarlos0/env/v10"
	"github.com/cenkalti/backoff/v4"
	chclient "github.com/mainflux/callhome/pkg/client"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "mqtt"
	envPrefixAuthz = "MG_THINGS_AUTH_GRPC_"
)

type config struct {
	LogLevel              string        `env:"MG_MQTT_ADAPTER_LOG_LEVEL"                    envDefault:"info"`
	MQTTPort              string        `env:"MG_MQTT_ADAPTER_MQTT_PORT"                    envDefault:"1883"`
	MQTTTargetHost        string        `env:"MG_MQTT_ADAPTER_MQTT_TARGET_HOST"             envDefault:"localhost"`
	MQTTTargetPort        string        `env:"MG_MQTT_ADAPTER_MQTT_TARGET_PORT"             envDefault:"1883"`
	MQTTForwarderTimeout  time.Duration `env:"MG_MQTT_ADAPTER_FORWARDER_TIMEOUT"            envDefault:"30s"`
	MQTTTargetHealthCheck string        `env:"MG_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"     envDefault:""`
	MQTTQoS               uint8         `env:"MG_MQTT_ADAPTER_MQTT_QOS"                     envDefault:"1"`
	HTTPPort              string        `env:"MG_MQTT_ADAPTER_WS_PORT"                      envDefault:"8080"`
	HTTPTargetHost        string        `env:"MG_MQTT_ADAPTER_WS_TARGET_HOST"               envDefault:"localhost"`
	HTTPTargetPort        string        `env:"MG_MQTT_ADAPTER_WS_TARGET_PORT"               envDefault:"8080"`
	HTTPTargetPath        string        `env:"MG_MQTT_ADAPTER_WS_TARGET_PATH"               envDefault:"/mqtt"`
	Instance              string        `env:"MG_MQTT_ADAPTER_INSTANCE"                     envDefault:""`
	JaegerURL             url.URL       `env:"MG_JAEGER_URL"                                envDefault:"http://localhost:14268/api/traces"`
	BrokerURL             string        `env:"MG_MESSAGE_BROKER_URL"                        envDefault:"nats://localhost:4222"`
	SendTelemetry         bool          `env:"MG_SEND_TELEMETRY"                            envDefault:"true"`
	InstanceID            string        `env:"MG_MQTT_ADAPTER_INSTANCE_ID"                  envDefault:""`
	ESURL                 string        `env:"MG_ES_URL"                                    envDefault:"nats://localhost:4222"`
	TraceRatio            float64       `env:"MG_JAEGER_TRACE_RATIO"                        envDefault:"1.0"`
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
		log.Fatalf("failed to init logger: %s", err)
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

	if cfg.MQTTTargetHealthCheck != "" {
		notify := func(e error, next time.Duration) {
			logger.Info(fmt.Sprintf("Broker not ready: %s, next try in %s", e.Error(), next))
		}

		err := backoff.RetryNotify(healthcheck(cfg), backoff.NewExponentialBackOff(), notify)
		if err != nil {
			logger.Error(fmt.Sprintf("MQTT healthcheck limit exceeded, exiting. %s ", err))
			exitCode = 1
			return
		}
	}

	serverConfig := server.Config{
		Host: cfg.HTTPTargetHost,
		Port: cfg.HTTPTargetPort,
	}

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	bsub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer bsub.Close()
	bsub = brokerstracing.NewPubSub(serverConfig, tracer, bsub)

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("mqtt://%s:%s", cfg.MQTTTargetHost, cfg.MQTTTargetPort), cfg.MQTTQoS, cfg.MQTTForwarderTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create MQTT publisher: %s", err))
		exitCode = 1
		return
	}
	defer mpub.Close()

	fwd := mqtt.NewForwarder(brokers.SubjectAllChannels, logger)
	fwd = mqtttracing.New(serverConfig, tracer, fwd, brokers.SubjectAllChannels)
	if err := fwd.Forward(ctx, svcName, bsub, mpub); err != nil {
		logger.Error(fmt.Sprintf("failed to forward message broker messages: %s", err))
		exitCode = 1
		return
	}

	np, err := brokers.NewPublisher(ctx, cfg.BrokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer np.Close()
	np = brokerstracing.NewPublisher(serverConfig, tracer, np)

	es, err := events.NewEventStore(ctx, cfg.ESURL, cfg.Instance)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s event store : %s", svcName, err))
		exitCode = 1
		return
	}

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuthz}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.SetupAuthz(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Successfully connected to things grpc server " + authHandler.Secure())

	h := mqtt.NewHandler(np, es, logger, authClient)
	h = handler.NewTracing(tracer, h)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.MQTTPort))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, logger, h)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.HTTPPort))
	g.Go(func() error {
		return proxyWS(ctx, cfg, logger, h)
	})

	g.Go(func() error {
		return stopSignalHandler(ctx, cancel, logger)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
	}
}

func proxyMQTT(ctx context.Context, cfg config, logger mglog.Logger, sessionHandler session.Handler) error {
	address := fmt.Sprintf(":%s", cfg.MQTTPort)
	target := fmt.Sprintf("%s:%s", cfg.MQTTTargetHost, cfg.MQTTTargetPort)
	mproxy := mp.New(address, target, sessionHandler, logger)

	errCh := make(chan error)
	go func() {
		errCh <- mproxy.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}

func proxyWS(ctx context.Context, cfg config, logger mglog.Logger, sessionHandler session.Handler) error {
	target := fmt.Sprintf("%s:%s", cfg.HTTPTargetHost, cfg.HTTPTargetPort)
	wp := websocket.New(target, cfg.HTTPTargetPath, "ws", sessionHandler, logger)
	http.Handle("/mqtt", wp.Handler())

	errCh := make(chan error)

	go func() {
		errCh <- wp.Listen(cfg.HTTPPort)
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
		res, err := http.Get(cfg.MQTTTargetHealthCheck)
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

func stopSignalHandler(ctx context.Context, cancel context.CancelFunc, logger mglog.Logger) error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGABRT)
	select {
	case sig := <-c:
		defer cancel()
		logger.Info(fmt.Sprintf("%s service shutdown by signal: %s", svcName, sig))
		return nil
	case <-ctx.Done():
		return nil
	}
}
