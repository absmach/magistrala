// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains timescale-reader main function to start the timescale-reader service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/atom"
	atomevents "github.com/absmach/magistrala/pkg/atom/events"
	atomauthn "github.com/absmach/magistrala/pkg/authn/atom"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/fluxmq"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/readers"
	readersgrpcapi "github.com/absmach/magistrala/readers/api/grpc"
	httpapi "github.com/absmach/magistrala/readers/api/http"
	middleware "github.com/absmach/magistrala/readers/middleware"
	"github.com/absmach/magistrala/readers/timescale"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName        = "timescaledb-reader"
	envPrefixDB    = "MG_TIMESCALE_"
	envPrefixHTTP  = "MG_TIMESCALE_READER_HTTP_"
	defDB          = "messages"
	defSvcHTTPPort = "9011"
	defSvcGRPCPort = "7011"
	envPrefixGrpc  = "MG_TIMESCALE_READER_GRPC_"
)

type config struct {
	LogLevel      string `env:"MG_TIMESCALE_READER_LOG_LEVEL"    envDefault:"info"`
	SendTelemetry bool   `env:"MG_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID    string `env:"MG_TIMESCALE_READER_INSTANCE_ID"  envDefault:""`
}

// atomEventsConfig points this reader at the same AMQP broker every other
// AMQP-speaking Magistrala service already uses (MG_MESSAGE_BROKER_URL) to
// consume Atom's domain events (MG-14). Exchange/RoutingKey must match
// Atom's own ATOM_EVENTS_AMQP_EXCHANGE/ATOM_EVENTS_AMQP_ROUTING_KEY --
// docker/.env sets both pairs from the same values for exactly that reason.
type atomEventsConfig struct {
	BrokerURL  string `env:"MG_MESSAGE_BROKER_URL"      envDefault:"amqp://guest:guest@localhost:5682/"`
	Exchange   string `env:"MG_ATOM_EVENTS_EXCHANGE"    envDefault:"atom.events"`
	RoutingKey string `env:"MG_ATOM_EVENTS_ROUTING_KEY" envDefault:"atom.events"`
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

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgclient.Connect(dbConfig)
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	repo := newService(db, logger)

	grpcServerConfig := server.Config{
		Port: defSvcGRPCPort,
	}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	registerReadersServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcReadersV1.RegisterReadersServiceServer(srv, readersgrpcapi.NewReadersServer(repo))
	}

	atomCfg := atom.LoadConfig()
	authn := atomauthn.NewAuthentication()
	clientsClient := atom.NewClientsCompat(authn)
	atomClient := atom.NewClient(atomCfg)
	channelsClient := atom.NewChannelsCompat(atomClient)
	policyEvaluator := atom.NewPolicyEvaluator(atomClient)
	policyService := atom.NewPolicyService(atomClient)
	logger.Info("AuthN/AuthZ configured to use Atom")

	invalidation := atomevents.NewRegistry()
	atomEvents := atomEventsConfig{}
	if err := env.Parse(&atomEvents); err != nil {
		logger.Error(fmt.Sprintf("failed to load Atom events configuration : %s", err))
		exitCode = 1
		return
	}
	// Event-driven cache invalidation (MG-14) is an optimization layered on
	// top of readAuthorizer's own TTL, never a substitute for it: a broker
	// that is unreachable at startup, or one that drops out later, must
	// leave the service running on TTL-only invalidation exactly as it did
	// before this existed -- never block or crash the reader.
	if sub, err := fluxmq.NewQueueSubscriber(ctx, atomEvents.BrokerURL, svcName, fluxmq.QueueSubscriberConfig{
		Exchange:   atomEvents.Exchange,
		RoutingKey: atomEvents.RoutingKey,
	}, logger); err != nil {
		logger.Warn(fmt.Sprintf("Atom events broker unavailable, caches will rely on TTL expiry only: %s", err))
	} else {
		defer sub.Close()
		// See cmd/postgres-reader/main.go: the queue is per service and
		// shared across replicas, so an event invalidates one replica's cache
		// and the rest rely on TTL expiry; per-replica queues are a
		// deployment choice for multi-replica setups.
		queue := "atom.events." + svcName
		if err := sub.Subscribe(ctx, events.SubscriberConfig{
			Consumer: svcName,
			Stream:   queue,
			Handler:  atomevents.NewHandler(invalidation, logger),
		}); err != nil {
			logger.Warn(fmt.Sprintf("failed to subscribe to Atom events, caches will rely on TTL expiry only: %s", err))
		}
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(repo, authn, clientsClient, channelsClient, policyEvaluator, policyService, atomClient, invalidation, svcName, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerReadersServiceServer, logger)

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Timescale reader service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, logger *slog.Logger) readers.MessageRepository {
	svc := timescale.New(db)
	svc = middleware.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics("timescale", "message_reader")
	svc = middleware.MetricsMiddleware(svc, counter, latency)

	return svc
}
