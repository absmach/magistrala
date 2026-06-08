// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the FluxMQ auth bridge service entry point.
// This service implements the FluxMQ auth callout server using ConnectRPC,
// bridging authentication requests to Magistrala's Clients service and
// authorization requests to Magistrala's Channels service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/absmach/fluxmq/pkg/proto/auth/v1/authv1connect"
	fluxmqgrpc "github.com/absmach/magistrala/fluxmq/api/grpc"
	fluxmqhttp "github.com/absmach/magistrala/fluxmq/api/http"
	"github.com/absmach/magistrala/internal/atom"
	mglog "github.com/absmach/magistrala/logger"
	atomauthn "github.com/absmach/magistrala/pkg/authn/atom"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/messaging"
	fluxmqbroker "github.com/absmach/magistrala/pkg/messaging/fluxmq"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "fluxmq-auth"
	defSvcGRPCPort = "7016"
	envPrefixCache = "MG_FLUXMQ_CACHE_"
	envPrefixGRPC  = "MG_FLUXMQ_GRPC_"
	envPrefixHTTP  = "MG_FLUXMQ_PUBLISH_HTTP_"
)

type config struct {
	LogLevel   string  `env:"MG_FLUXMQ_LOG_LEVEL"    envDefault:"info"`
	BrokerURL  string  `env:"MG_MESSAGE_BROKER_URL"  envDefault:"amqp://guest:guest@localhost:5682/"`
	JaegerURL  url.URL `env:"MG_JAEGER_URL"           envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio float64 `env:"MG_JAEGER_TRACE_RATIO"   envDefault:"1.0"`
	InstanceID string  `env:"MG_FLUXMQ_INSTANCE_ID"   envDefault:""`
}

type fanoutPublisher struct {
	publishers []messaging.Publisher
}

func (fp fanoutPublisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	for _, publisher := range fp.publishers {
		if err := publisher.Publish(ctx, topic, msg); err != nil {
			return err
		}
	}
	return nil
}

func (fp fanoutPublisher) Close() error {
	errs := make([]error, 0, len(fp.publishers))
	for _, publisher := range fp.publishers {
		errs = append(errs, publisher.Close())
	}
	return errors.Join(errs...)
}

type writerBridgeHandler struct {
	ctx       context.Context
	publisher messaging.Publisher
}

func (h writerBridgeHandler) Handle(msg *messaging.Message) error {
	return h.publisher.Publish(h.ctx, messaging.EncodeMessageTopic(msg), msg)
}

func (h writerBridgeHandler) Cancel() error {
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration: %s", svcName, err)
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

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()

	atomCfg := atom.LoadConfig()
	if atomCfg.URL == "" {
		logger.Error("ATOM_URL is required")
		exitCode = 1
		return
	}
	atomAuthz := atom.NewClient(atomCfg)
	authn := atomauthn.NewAuthentication()
	clientsClient := atom.NewClientsCompat(authn, atomAuthz)
	domainsClient := atom.NewDomainsCompat(atomAuthz)
	channelsClient := atom.NewChannelsCompat(atomAuthz)
	logger.Info("FluxMQ authentication, authorization, and route resolution configured to use Atom")

	// Topic parser with cache for route resolution.
	cacheConfig := messaging.CacheConfig{}
	if err := env.ParseWithOptions(&cacheConfig, env.Options{Prefix: envPrefixCache}); err != nil {
		logger.Error(fmt.Sprintf("failed to load cache configuration: %s", err))
		exitCode = 1
		return
	}
	parser, err := messaging.NewTopicParser(cacheConfig, channelsClient, domainsClient)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create topic parser: %s", err))
		exitCode = 1
		return
	}

	// Start FluxMQ auth Connect/gRPC server over h2c.
	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load gRPC server configuration: %s", err))
		exitCode = 1
		return
	}

	mux := http.NewServeMux()
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create OTel interceptor: %s", err))
		exitCode = 1
		return
	}
	path, handler := authv1connect.NewAuthServiceHandler(
		fluxmqgrpc.NewServer(clientsClient, channelsClient, parser, atomAuthz),
		connect.WithInterceptors(otelInterceptor),
	)
	mux.Handle(path, handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck // HTTP response write; client disconnect is non-fatal.
	})

	address := fmt.Sprintf("%s:%s", grpcServerConfig.Host, grpcServerConfig.Port)
	httpServer := &http.Server{
		Addr:              address,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:       grpcServerConfig.ReadTimeout,
		WriteTimeout:      grpcServerConfig.WriteTimeout,
		ReadHeaderTimeout: grpcServerConfig.ReadHeaderTimeout,
		IdleTimeout:       grpcServerConfig.IdleTimeout,
		MaxHeaderBytes:    grpcServerConfig.MaxHeaderBytes,
	}

	messagePublisher, err := fluxmqbroker.NewUndeclaredPublisher(
		ctx,
		cfg.BrokerURL,
		fluxmqbroker.ConnectionName("fluxmq-ui-message-publish-proxy"),
	)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create publish proxy message publisher: %s", err))
		exitCode = 1
		return
	}
	defer messagePublisher.Close()

	writerPublisher, err := fluxmqbroker.NewUndeclaredPublisher(
		ctx,
		cfg.BrokerURL,
		fluxmqbroker.Prefix("writers"),
		fluxmqbroker.ConnectionName("fluxmq-ui-publish-proxy"),
	)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create publish proxy writer publisher: %s", err))
		exitCode = 1
		return
	}
	defer writerPublisher.Close()
	publisher := fanoutPublisher{publishers: []messaging.Publisher{messagePublisher, writerPublisher}}

	writerBridge, err := fluxmqbroker.NewPubSub(
		ctx,
		cfg.BrokerURL,
		logger,
		fluxmqbroker.DirectTopicOnly(),
		fluxmqbroker.ConnectionName("fluxmq-mqtt-writer-bridge"),
	)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create MQTT writer bridge subscriber: %s", err))
		exitCode = 1
		return
	}
	defer writerBridge.Close()
	if err := writerBridge.Subscribe(ctx, messaging.SubscriberConfig{
		ID:             cfg.InstanceID + "-mqtt-writer-bridge",
		Topic:          "m/#",
		Handler:        writerBridgeHandler{ctx: ctx, publisher: writerPublisher},
		DeliveryPolicy: messaging.DeliverNewPolicy,
	}); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe MQTT writer bridge: %s", err))
		exitCode = 1
		return
	}
	logger.Info("FluxMQ MQTT writer bridge subscribed", "topic", "m/#")

	httpServerConfig := server.Config{Port: "9026"}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load publish proxy HTTP server configuration: %s", err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(
		ctx,
		cancel,
		"fluxmq-publish",
		httpServerConfig,
		fluxmqhttp.MakePublishHandler(authn, atomAuthz, publisher),
		logger,
	)

	g.Go(func() error {
		logger.Info(fmt.Sprintf("%s service h2c server listening at %s", svcName, address))
		var err error
		switch {
		case grpcServerConfig.CertFile != "" || grpcServerConfig.KeyFile != "":
			err = httpServer.ListenAndServeTLS(grpcServerConfig.CertFile, grpcServerConfig.KeyFile)
		default:
			err = httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			cancel()
			return err
		}
		return nil
	})

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), server.StopWaitTime) //nolint:contextcheck
		defer shutdownCancel()
		if err := hs.Stop(); err != nil {
			return fmt.Errorf("failed to shutdown publish proxy server: %w", err)
		}
		if err := httpServer.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck
			return fmt.Errorf("failed to shutdown %s server: %w", svcName, err)
		}
		logger.Info(fmt.Sprintf("%s service shutdown at %s", svcName, address))
		return nil
	})

	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		select {
		case sig := <-c:
			cancel()
			logger.Info(fmt.Sprintf("%s service shutdown by signal: %s", svcName, sig))
			return nil
		case <-ctx.Done():
			return nil
		}
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}
