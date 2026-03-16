// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the FluxMQ auth bridge service entry point.
// This service implements the FluxMQ auth callout gRPC server, bridging
// authentication requests to SuperMQ's Clients service and authorization
// requests to SuperMQ's Channels service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	grpcFluxmqV1 "github.com/absmach/supermq/api/grpc/fluxmq/v1"
	fluxmqgrpc "github.com/absmach/supermq/fluxmq/api/grpc"
	smqlog "github.com/absmach/supermq/logger"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/server"
	grpcserver "github.com/absmach/supermq/pkg/server/grpc"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName           = "fluxmq-auth"
	defSvcGRPCPort    = "7016"
	envPrefixClients  = "SMQ_CLIENTS_GRPC_"
	envPrefixChannels = "SMQ_CHANNELS_GRPC_"
	envPrefixDomains  = "SMQ_DOMAINS_GRPC_"
	envPrefixCache    = "SMQ_FLUXMQ_CACHE_"
	envPrefixGRPC     = "SMQ_FLUXMQ_GRPC_"
)

type config struct {
	LogLevel   string  `env:"SMQ_FLUXMQ_LOG_LEVEL"    envDefault:"info"`
	JaegerURL  url.URL `env:"SMQ_JAEGER_URL"           envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio float64 `env:"SMQ_JAEGER_TRACE_RATIO"   envDefault:"1.0"`
	InstanceID string  `env:"SMQ_FLUXMQ_INSTANCE_ID"   envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration: %s", svcName, err)
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

	// Connect to Domains gRPC service (needed for topic route resolution).
	domsGrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&domsGrpcCfg, env.Options{Prefix: envPrefixDomains}); err != nil {
		logger.Error(fmt.Sprintf("failed to load domains gRPC client configuration: %s", err))
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
	logger.Info("Domains gRPC client connected " + domainsHandler.Secure())

	// Connect to Clients gRPC service (authentication).
	clientsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientsClientCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration: %s", err))
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
	logger.Info("Clients gRPC client connected " + clientsHandler.Secure())

	// Connect to Channels gRPC service (authorization + route resolution).
	channelsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&channelsClientCfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration: %s", err))
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
	logger.Info("Channels gRPC client connected " + channelsHandler.Secure())

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

	// Start FluxMQ auth gRPC server.
	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load gRPC server configuration: %s", err))
		exitCode = 1
		return
	}

	registerFluxMQServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcFluxmqV1.RegisterAuthServiceServer(srv, fluxmqgrpc.NewServer(clientsClient, channelsClient, parser))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerFluxMQServer, logger)

	g.Go(func() error {
		return gs.Start()
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
