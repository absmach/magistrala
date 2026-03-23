// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains http-adapter main function to start the http-adapter service.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/mgate"
	mgatehttp "github.com/absmach/mgate/pkg/http"
	"github.com/absmach/mgate/pkg/session"
	"github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcDomainsV1 "github.com/absmach/supermq/api/grpc/domains/v1"
	"github.com/absmach/supermq/auth"
	adapter "github.com/absmach/supermq/http"
	httpapi "github.com/absmach/supermq/http/api"
	"github.com/absmach/supermq/http/middleware"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authsvcAuthn "github.com/absmach/supermq/pkg/authn/authsvc"
	jwksAuthn "github.com/absmach/supermq/pkg/authn/jwks"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	msgevents "github.com/absmach/supermq/pkg/messaging/events"
	"github.com/absmach/supermq/pkg/messaging/handler"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName            = "http_adapter"
	envPrefix          = "MG_HTTP_ADAPTER_"
	envPrefixCache     = "MG_HTTP_ADAPTER_CACHE_"
	envPrefixClients   = "MG_CLIENTS_GRPC_"
	envPrefixChannels  = "MG_CHANNELS_GRPC_"
	envPrefixAuth      = "MG_AUTH_GRPC_"
	envPrefixDomains   = "MG_DOMAINS_GRPC_"
	defSvcHTTPPort     = "80"
	targetHTTPProtocol = "http"
	targetHTTPHost     = "localhost"
	targetHTTPPort     = "81"
	targetHTTPPath     = ""
)

type config struct {
	LogLevel         string  `env:"MG_HTTP_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL        string  `env:"MG_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL        url.URL `env:"MG_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry    bool    `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID       string  `env:"MG_HTTP_ADAPTER_INSTANCE_ID" envDefault:""`
	TraceRatio       float64 `env:"MG_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
	ESURL            string  `env:"MG_ES_URL"                   envDefault:"nats://localhost:4222"`
	AuthKeyAlgorithm string  `env:"MG_AUTH_KEYS_ALGORITHM"      envDefault:"RS256"`
	JWKSURL          string  `env:"MG_AUTH_JWKS_URL"            envDefault:"http://auth:9001/keys/.well-known/jwks.json"`
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
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
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
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration : %s", err))
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

	authnCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&authnCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	isSymmetric, err := auth.IsSymmetricAlgorithm(cfg.AuthKeyAlgorithm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse auth key algorithm : %s", err))
		exitCode = 1
		return
	}
	var authn smqauthn.Authentication
	var authnClient grpcclient.Handler
	switch {
	case !isSymmetric:
		authn, authnClient, err = jwksAuthn.NewAuthentication(ctx, cfg.JWKSURL, authnCfg)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully set up jwks authentication on " + cfg.JWKSURL)
	default:
		authn, authnClient, err = authsvcAuthn.NewAuthentication(ctx, authnCfg)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())
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

	nps, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(httpServerConfig, tracer, nps)

	nps, err = msgevents.NewPubSubMiddleware(ctx, nps, cfg.ESURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create event store middleware: %s", err))
		exitCode = 1
		return
	}

	resolver := messaging.NewTopicResolver(channelsClient, domainsClient)
	handler, err := newHandler(nps, authn, cacheConfig, clientsClient, channelsClient, domainsClient, logger, tracer)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))
		exitCode = 1
		return
	}
	svc := newService(clientsClient, channelsClient, authn, nps, logger, tracer)

	targetServerCfg := server.Config{Port: targetHTTPPort}

	hs := httpserver.NewServer(ctx, cancel, svcName, targetServerCfg, httpapi.MakeHandler(ctx, svc, resolver, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return proxyHTTP(ctx, httpServerConfig, logger, handler)
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("HTTP adapter service terminated: %s", err))
	}
}

func newHandler(pubsub messaging.PubSub, authn smqauthn.Authentication, cacheCfg messaging.CacheConfig, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, domains grpcDomainsV1.DomainsServiceClient, logger *slog.Logger, tracer trace.Tracer) (session.Handler, error) {
	parser, err := messaging.NewTopicParser(cacheCfg, channels, domains)
	if err != nil {
		return nil, err
	}
	h := adapter.NewHandler(pubsub, logger, authn, clients, channels, parser)
	h = handler.NewTracing(tracer, h)
	h = handler.NewLogging(h, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "handler")
	h = handler.NewMetrics(h, counter, latency)

	return h, nil
}

func newService(clientsClient grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, authn smqauthn.Authentication, nps messaging.PubSub, logger *slog.Logger, tracer trace.Tracer) adapter.Service {
	svc := adapter.NewService(clientsClient, channels, authn, nps)
	svc = middleware.NewTracing(tracer, svc)
	svc = middleware.NewLogging(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetrics(svc, counter, latency)
	return svc
}

func proxyHTTP(ctx context.Context, cfg server.Config, logger *slog.Logger, sessionHandler session.Handler) error {
	config := mgate.Config{
		Port:           cfg.Port,
		TargetProtocol: targetHTTPProtocol,
		TargetHost:     targetHTTPHost,
		TargetPort:     targetHTTPPort,
		TargetPath:     targetHTTPPath,
	}
	if cfg.CertFile != "" || cfg.KeyFile != "" {
		tlsCert, err := server.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return err
		}
		config.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}
	}
	mp, err := mgatehttp.NewProxy(config, sessionHandler, logger, []string{}, []string{"/health", "/metrics"})
	if err != nil {
		return err
	}
	http.HandleFunc("/", mp.ServeHTTP)

	errCh := make(chan error)
	switch {
	case cfg.CertFile != "" || cfg.KeyFile != "":
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service HTTPS server listening at %s:%s with TLS", svcName, cfg.Host, cfg.Port))
	default:
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service HTTP server listening at %s:%s without TLS", svcName, cfg.Host, cfg.Port))
	}

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy HTTP shutdown at %s:%s", config.Host, config.Port))
		return nil
	case err := <-errCh:
		return err
	}
}
