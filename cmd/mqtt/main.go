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
	authapi "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt"
	"github.com/mainflux/mainflux/mqtt/events"
	mqtttracing "github.com/mainflux/mainflux/mqtt/tracing"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	brokerstracing "github.com/mainflux/mainflux/pkg/messaging/brokers/tracing"
	mqttpub "github.com/mainflux/mainflux/pkg/messaging/mqtt"
	"github.com/mainflux/mainflux/pkg/uuid"
	mp "github.com/mainflux/mproxy/pkg/mqtt"
	"github.com/mainflux/mproxy/pkg/session"
	"github.com/mainflux/mproxy/pkg/websocket"
	"golang.org/x/sync/errgroup"
)

const svcName = "mqtt"

type config struct {
	LogLevel              string        `env:"MF_MQTT_ADAPTER_LOG_LEVEL"                    envDefault:"info"`
	MQTTPort              string        `env:"MF_MQTT_ADAPTER_MQTT_PORT"                    envDefault:"1883"`
	MQTTTargetHost        string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_HOST"             envDefault:"localhost"`
	MQTTTargetPort        string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_PORT"             envDefault:"1883"`
	MQTTForwarderTimeout  time.Duration `env:"MF_MQTT_ADAPTER_FORWARDER_TIMEOUT"            envDefault:"30s"`
	MQTTTargetHealthCheck string        `env:"MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"     envDefault:""`
	HTTPPort              string        `env:"MF_MQTT_ADAPTER_WS_PORT"                      envDefault:"8080"`
	HTTPTargetHost        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_HOST"               envDefault:"localhost"`
	HTTPTargetPort        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_PORT"               envDefault:"8080"`
	HTTPTargetPath        string        `env:"MF_MQTT_ADAPTER_WS_TARGET_PATH"               envDefault:"/mqtt"`
	Instance              string        `env:"MF_MQTT_ADAPTER_INSTANCE"                     envDefault:""`
	JaegerURL             string        `env:"MF_JAEGER_URL"                                envDefault:"http://jaeger:14268/api/traces"`
	BrokerURL             string        `env:"MF_BROKER_URL"                                envDefault:"nats://localhost:4222"`
	SendTelemetry         bool          `env:"MF_SEND_TELEMETRY"                            envDefault:"true"`
	InstanceID            string        `env:"MF_MQTT_ADAPTER_INSTANCE_ID"                  envDefault:""`
	ESURL                 string        `env:"MF_MQTT_ADAPTER_ES_URL"                       envDefault:"redis://localhost:6379/0"`
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

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

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

	nps, err := brokers.NewPubSub(cfg.BrokerURL, "mqtt", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(serverConfig, tracer, nps)

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("%s:%s", cfg.MQTTTargetHost, cfg.MQTTTargetPort), cfg.MQTTForwarderTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create MQTT publisher: %s", err))
		exitCode = 1
		return
	}
	defer mpub.Close()

	fwd := mqtt.NewForwarder(brokers.SubjectAllChannels, logger)
	fwd = mqtttracing.New(serverConfig, tracer, fwd, brokers.SubjectAllChannels)
	if err := fwd.Forward(ctx, svcName, nps, mpub); err != nil {
		logger.Error(fmt.Sprintf("failed to forward message broker messages: %s", err))
		exitCode = 1
		return
	}

	np, err := brokers.NewPublisher(cfg.BrokerURL)
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

	auth, aHandler, err := authapi.SetupAuthz("authz")
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer aHandler.Close()

	logger.Info("Successfully connected to things grpc server " + aHandler.Secure())

	h := mqtt.NewHandler([]messaging.Publisher{np}, es, logger, auth)
	h = mqtttracing.NewHandler(tracer, h)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
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
	address := fmt.Sprintf(":%s", cfg.MQTTPort)
	target := fmt.Sprintf("%s:%s", cfg.MQTTTargetHost, cfg.MQTTTargetPort)
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
	target := fmt.Sprintf("%s:%s", cfg.HTTPTargetHost, cfg.HTTPTargetPort)
	wp := websocket.New(target, cfg.HTTPTargetPath, "ws", handler, logger)
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
