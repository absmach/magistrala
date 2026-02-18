// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains mqtt-adapter main function to start the mqtt-adapter service.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	mgate "github.com/absmach/mgate"
	mgatemqtt "github.com/absmach/mgate/pkg/mqtt"
	"github.com/absmach/mgate/pkg/mqtt/websocket"
	"github.com/absmach/mgate/pkg/session"
	mgtls "github.com/absmach/mgate/pkg/tls"
	"github.com/absmach/supermq"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/mqtt"
	"github.com/absmach/supermq/mqtt/events"
	mqtttracing "github.com/absmach/supermq/mqtt/tracing"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	msgevents "github.com/absmach/supermq/pkg/messaging/events"
	"github.com/absmach/supermq/pkg/messaging/handler"
	mqttpub "github.com/absmach/supermq/pkg/messaging/mqtt"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/cenkalti/backoff/v4"
	"github.com/eclipse/paho.mqtt.golang/packets"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "mqtt"
	envPrefixCache    = "SMQ_MQTT_ADAPTER_CACHE_"
	envPrefixClients  = "SMQ_CLIENTS_GRPC_"
	envPrefixChannels = "SMQ_CHANNELS_GRPC_"
	envPrefixDomains  = "SMQ_DOMAINS_GRPC_"
	envPrefixMQTT     = "SMQ_MQTT_ADAPTER_"
	wsPathPrefix      = "/mqtt"
)

type config struct {
	LogLevel              string        `env:"SMQ_MQTT_ADAPTER_LOG_LEVEL"                    envDefault:"info"`
	MQTTPort              string        `env:"SMQ_MQTT_ADAPTER_MQTT_PORT"                    envDefault:"1883"`
	MQTTTargetProtocol    string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_PROTOCOL"         envDefault:"mqtt"`
	MQTTTargetHost        string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_HOST"             envDefault:"localhost"`
	MQTTTargetPort        string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_PORT"             envDefault:"1883"`
	MQTTTargetUsername    string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_USERNAME"         envDefault:""`
	MQTTTargetPassword    string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_PASSWORD"         envDefault:""`
	MQTTForwarderTimeout  time.Duration `env:"SMQ_MQTT_ADAPTER_FORWARDER_TIMEOUT"            envDefault:"30s"`
	MQTTTargetHealthCheck string        `env:"SMQ_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"     envDefault:""`
	MQTTQoS               uint8         `env:"SMQ_MQTT_ADAPTER_MQTT_QOS"                     envDefault:"1"`
	HTTPPort              string        `env:"SMQ_MQTT_ADAPTER_WS_PORT"                      envDefault:"8080"`
	HTTPTargetProtocol    string        `env:"SMQ_MQTT_ADAPTER_WS_TARGET_PROTOCOL"           envDefault:"http"`
	HTTPTargetHost        string        `env:"SMQ_MQTT_ADAPTER_WS_TARGET_HOST"               envDefault:"localhost"`
	HTTPTargetPort        string        `env:"SMQ_MQTT_ADAPTER_WS_TARGET_PORT"               envDefault:"8080"`
	HTTPTargetPath        string        `env:"SMQ_MQTT_ADAPTER_WS_TARGET_PATH"               envDefault:"/mqtt"`
	Instance              string        `env:"SMQ_MQTT_ADAPTER_INSTANCE"                     envDefault:""`
	JaegerURL             url.URL       `env:"SMQ_JAEGER_URL"                                envDefault:"http://localhost:4318/v1/traces"`
	BrokerURL             string        `env:"SMQ_MESSAGE_BROKER_URL"                        envDefault:"nats://localhost:4222"`
	SendTelemetry         bool          `env:"SMQ_SEND_TELEMETRY"                            envDefault:"true"`
	InstanceID            string        `env:"SMQ_MQTT_ADAPTER_INSTANCE_ID"                  envDefault:""`
	ESURL                 string        `env:"SMQ_ES_URL"                                    envDefault:"nats://localhost:4222"`
	TraceRatio            float64       `env:"SMQ_JAEGER_TRACE_RATIO"                        envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := smqlog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer smqlog.ExitWithError(&exitCode)

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

	tlsCfg, err := mgtls.NewConfig(env.Options{Prefix: envPrefixMQTT})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load TLS config: %s", err))
		exitCode = 1
		return
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

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("mqtt://%s:%s", cfg.MQTTTargetHost, cfg.MQTTTargetPort), cfg.MQTTTargetUsername, cfg.MQTTTargetPassword, cfg.MQTTQoS, cfg.MQTTForwarderTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create MQTT publisher: %s", err))
		exitCode = 1
		return
	}
	defer mpub.Close()

	fwd := mqtt.NewForwarder(brokers.SubjectAllMessages, logger)
	fwd = mqtttracing.New(serverConfig, tracer, fwd, brokers.SubjectAllMessages)
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

	np, err = msgevents.NewPublisherMiddleware(ctx, np, cfg.ESURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create event store middleware: %s", err))
		exitCode = 1
		return
	}

	domsGrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&domsGrpcCfg, env.Options{Prefix: envPrefixDomains}); err != nil {
		logger.Error(fmt.Sprintf("failed to load domains gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	_, domainsClient, domainsHandler, err := domainsAuthz.NewAuthorization(ctx, domsGrpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer domainsHandler.Close()

	clientsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientsClientCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	clientsClient, clientsHandler, err := grpcclient.SetupClientsClient(ctx, clientsClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer clientsHandler.Close()
	logger.Info("Clients service gRPC client successfully connected to clients gRPC server " + clientsHandler.Secure())

	channelsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&channelsClientCfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	channelsClient, channelsHandler, err := grpcclient.SetupChannelsClient(ctx, channelsClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer channelsHandler.Close()
	logger.Info("Channels service gRPC client successfully connected to channels gRPC server " + channelsHandler.Secure())

	cacheConfig := messaging.CacheConfig{}
	if err := env.ParseWithOptions(&cacheConfig, env.Options{Prefix: envPrefixCache}); err != nil {
		logger.Error(fmt.Sprintf("failed to load cache configuration : %s", err))
		exitCode = 1
		return
	}
	parser, err := messaging.NewTopicParser(cacheConfig, channelsClient, domainsClient)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create topic parsers: %s", err))
		exitCode = 1
		return
	}

	h := mqtt.NewHandler(np, logger, clientsClient, channelsClient, parser)

	h, err = events.NewEventStoreMiddleware(ctx, h, cfg.ESURL, cfg.Instance)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create event store middleware: %s", err))
		exitCode = 1
		return
	}

	h = handler.NewTracing(tracer, h)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	beforeHandler := beforeHandler{
		resolver: messaging.NewTopicResolver(channelsClient, domainsClient),
	}

	afterHandler := afterHandler{
		username: cfg.MQTTTargetUsername,
		password: cfg.MQTTTargetPassword,
	}
	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.MQTTPort))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, tlsCfg, logger, h, beforeHandler, afterHandler)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.HTTPPort))
	g.Go(func() error {
		return proxyWS(ctx, cfg, tlsCfg, logger, h, afterHandler)
	})

	g.Go(func() error {
		return stopSignalHandler(ctx, cancel, logger)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
	}
}

func proxyMQTT(ctx context.Context, cfg config, tlsCfg mgtls.Config, logger *slog.Logger, sessionHandler session.Handler, beforeHandler, afterHandler session.Interceptor) error {
	var err error
	config := mgate.Config{
		Port:       cfg.MQTTPort,
		TargetHost: cfg.MQTTTargetHost,
		TargetPort: cfg.MQTTTargetPort,
	}
	errCh := make(chan error)

	config.TLSConfig, err = mgtls.LoadTLSConfig(&tlsCfg, &tls.Config{})
	if err != nil {
		return err
	}

	mgate := mgatemqtt.New(config, sessionHandler, beforeHandler, afterHandler, logger)

	go func() {
		errCh <- mgate.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT shutdown at %s:%s", config.Host, config.Port))
		return nil
	case err := <-errCh:
		return err
	}
}

func proxyWS(ctx context.Context, cfg config, tlsCfg mgtls.Config, logger *slog.Logger, sessionHandler session.Handler, interceptor session.Interceptor) error {
	var err error
	config := mgate.Config{
		Port:           cfg.HTTPPort,
		TargetProtocol: "ws",
		TargetHost:     cfg.HTTPTargetHost,
		TargetPort:     cfg.HTTPTargetPort,
		TargetPath:     cfg.HTTPTargetPath,
		PathPrefix:     wsPathPrefix,
	}
	config.TLSConfig, err = mgtls.LoadTLSConfig(&tlsCfg, &tls.Config{})
	if err != nil {
		return err
	}

	wp := websocket.New(config, sessionHandler, nil, interceptor, logger)
	http.HandleFunc(wsPathPrefix, wp.ServeHTTP)

	errCh := make(chan error)

	go func() {
		errCh <- wp.Listen(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s:%s", config.Host, config.Port))
		return nil
	case err := <-errCh:
		return err
	}
}

func healthcheck(cfg config) func() error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return func() error {
		res, err := client.Get(cfg.MQTTTargetHealthCheck)
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

func stopSignalHandler(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) error {
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

type afterHandler struct {
	username string
	password string
}

// This interceptor adds the correct credentials to upstream MQTT broker since the downstream clients
// are authenticated to the MQTT adapter but not upstream MQTT broker.
func (ah afterHandler) Intercept(ctx context.Context, pkt packets.ControlPacket, dir session.Direction) (packets.ControlPacket, error) {
	if connectPkt, ok := pkt.(*packets.ConnectPacket); ok {
		if ah.username != "" {
			connectPkt.Username = ah.username
			connectPkt.UsernameFlag = true
		}
		if ah.password != "" {
			connectPkt.Password = []byte(ah.password)
			connectPkt.PasswordFlag = true
		}

		return connectPkt, nil
	}

	return pkt, nil
}

type beforeHandler struct {
	resolver messaging.TopicResolver
}

// This interceptor is used to replace domain and channel routes with relevant domain and channel IDs in the message topic.
func (bh beforeHandler) Intercept(ctx context.Context, pkt packets.ControlPacket, dir session.Direction) (packets.ControlPacket, error) {
	switch pt := pkt.(type) {
	case *packets.SubscribePacket:
		for i, topic := range pt.Topics {
			ft, err := bh.resolver.ResolveTopic(ctx, topic)
			if err != nil {
				return nil, err
			}
			pt.Topics[i] = ft
		}

		return pt, nil
	case *packets.UnsubscribePacket:
		for i, topic := range pt.Topics {
			ft, err := bh.resolver.ResolveTopic(ctx, topic)
			if err != nil {
				return nil, err
			}
			pt.Topics[i] = ft
		}
		return pt, nil
	case *packets.PublishPacket:
		ft, err := bh.resolver.ResolveTopic(ctx, pt.TopicName)
		if err != nil {
			return nil, err
		}
		pt.TopicName = ft

		return pt, nil
	}

	return pkt, nil
}
