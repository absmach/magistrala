// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains coap-adapter main function to start the coap-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/mgate"
	mgatecoap "github.com/absmach/mgate/pkg/coap"
	"github.com/absmach/mgate/pkg/session"
	mgtls "github.com/absmach/mgate/pkg/tls"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/coap"
	httpapi "github.com/absmach/supermq/coap/api"
	"github.com/absmach/supermq/coap/middleware"
	smqlog "github.com/absmach/supermq/logger"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	msgevents "github.com/absmach/supermq/pkg/messaging/events"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	coapserver "github.com/absmach/supermq/pkg/server/coap"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/pion/dtls/v3"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "coap_adapter"
	envPrefix         = "MG_COAP_ADAPTER_"
	envPrefixHTTP     = "MG_COAP_ADAPTER_HTTP_"
	envPrefixDTLS     = "MG_COAP_ADAPTER_SERVER_"
	envPrefixCache    = "MG_COAP_CACHE_"
	envPrefixClients  = "MG_CLIENTS_GRPC_"
	envPrefixChannels = "MG_CHANNELS_GRPC_"
	envPrefixDomains  = "MG_DOMAINS_GRPC_"
	defSvcHTTPPort    = "5683"
	defSvcCoAPPort    = "5683"
	targetProtocol    = "coap"
	targetCoapPort    = "5682"
)

type config struct {
	LogLevel      string  `env:"MG_COAP_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string  `env:"MG_COAP_ADAPTER_INSTANCE_ID" envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
	ESURL         string  `env:"MG_ES_URL"                   envDefault:"nats://localhost:4222"`
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	coapServerConfig := server.Config{Port: defSvcCoAPPort}
	if err := env.ParseWithOptions(&coapServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s CoAP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	dtlsCfg, err := mgtls.NewConfig(env.Options{Prefix: envPrefixDTLS})
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load %s DTLS configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	cacheConfig := messaging.CacheConfig{}
	if err := env.ParseWithOptions(&cacheConfig, env.Options{Prefix: envPrefixCache}); err != nil {
		logger.Error(fmt.Sprintf("failed to load cache configuration : %s", err))
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

	logger.Info("Domains service gRPC client successfully connected to domains gRPC server " + domainsHandler.Secure())

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

	nps, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(coapServerConfig, tracer, nps)

	nps, err = msgevents.NewPubSubMiddleware(ctx, nps, cfg.ESURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create event store middleware: %s", err))
		exitCode = 1
		return
	}

	svc := coap.New(clientsClient, channelsClient, nps)

	svc = middleware.NewTracing(tracer, svc)

	svc = middleware.NewLogging(svc, logger)

	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetrics(svc, counter, latency)

	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(cfg.InstanceID), logger)

	parser, err := messaging.NewTopicParser(cacheConfig, channelsClient, domainsClient)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create topic parsers: %s", err))
		exitCode = 1
		return
	}
	cs := coapserver.NewServer(ctx, cancel, svcName, server.Config{Host: coapServerConfig.Host, Port: targetCoapPort}, httpapi.MakeCoAPHandler(svc, channelsClient, parser, logger), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		g.Go(func() error {
			return cs.Start()
		})
		handler := coap.NewHandler(logger, clientsClient, channelsClient, parser)
		return proxyCoAP(ctx, coapServerConfig, dtlsCfg, handler, logger)
	})
	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs, cs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("CoAP adapter service terminated: %s", err))
	}
}

func proxyCoAP(ctx context.Context, cfg server.Config, dtlsCfg mgtls.Config, handler session.Handler, logger *slog.Logger) error {
	var err error
	config := mgate.Config{
		Host:           "",
		Port:           cfg.Port,
		TargetProtocol: targetProtocol,
		TargetHost:     cfg.Host,
		TargetPort:     targetCoapPort,
	}

	mg := mgatecoap.NewProxy(config, handler, logger)

	errCh := make(chan error)

	config.DTLSConfig, err = mgtls.LoadTLSConfig(&dtlsCfg, &dtls.Config{})
	if err != nil {
		return err
	}

	switch {
	case config.DTLSConfig != nil:
		dltsCfg := config
		mgDtls := mgatecoap.NewProxy(dltsCfg, handler, logger)
		logger.Info(fmt.Sprintf("Starting COAP with DTLS proxy on port %s", cfg.Port))
		go func() {
			errCh <- mgDtls.Listen(ctx)
		}()
	default:
		logger.Info(fmt.Sprintf("Starting COAP without DTLS proxy on port %s", cfg.Port))
		go func() {
			errCh <- mg.Listen(ctx)
		}()
	}
	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy COAP shutdown at %s:%s", config.Host, config.Port))
		return nil
	case err := <-errCh:
		return err
	}
}
