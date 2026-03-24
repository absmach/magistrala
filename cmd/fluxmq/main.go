// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the FluxMQ auth bridge service entry point.
// This service implements the FluxMQ auth callout server using ConnectRPC,
// bridging authentication requests to SuperMQ's Clients service and
// authorization requests to SuperMQ's Channels service.
package main

import (
	"context"
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
	fluxmqgrpc "github.com/absmach/supermq/fluxmq/api/grpc"
	smqlog "github.com/absmach/supermq/logger"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
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
		fluxmqgrpc.NewServer(clientsClient, channelsClient, parser),
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
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), server.StopWaitTime) //nolint:contextcheck
		defer shutdownCancel()
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
