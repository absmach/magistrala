// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains bootstrap main function to start the bootstrap service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	httpapi "github.com/absmach/magistrala/bootstrap/api"
	"github.com/absmach/magistrala/bootstrap/events/producer"
	"github.com/absmach/magistrala/bootstrap/middleware"
	bootstrappg "github.com/absmach/magistrala/bootstrap/postgres"
	"github.com/absmach/magistrala/bootstrap/tracing"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/atom"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	atomauthn "github.com/absmach/magistrala/pkg/authn/atom"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/jaeger"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "bootstrap"
	envPrefixDB    = "MG_BOOTSTRAP_DB_"
	envPrefixHTTP  = "MG_BOOTSTRAP_HTTP_"
	defDB          = "bootstrap"
	defSvcHTTPPort = "9013"
)

type config struct {
	LogLevel          string  `env:"MG_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	DBEncryptionKey   string  `env:"MG_BOOTSTRAP_DB_ENCRYPTION_KEY"    envDefault:"12345678910111213141516171819202"`
	DBEncryptionKeyID string  `env:"MG_BOOTSTRAP_DB_ENCRYPTION_KEY_ID" envDefault:"primary"`
	ESConsumerName    string  `env:"MG_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	JaegerURL         url.URL `env:"MG_JAEGER_URL"                envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry     bool    `env:"MG_SEND_TELEMETRY"            envDefault:"true"`
	InstanceID        string  `env:"MG_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
	ESURL             string  `env:"MG_ES_URL"                    envDefault:"nats://localhost:4222"`
	TraceRatio        float64 `env:"MG_JAEGER_TRACE_RATIO"        envDefault:"1.0"`
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

	// Create new postgres client
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}
	migration := bootstrappg.Migration()

	db, err := pgclient.Setup(dbConfig, *migration)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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
	tracer := tp.Tracer(svcName)

	atomCfg := atom.LoadConfig()
	if atomCfg.URL == "" {
		logger.Error("ATOM_URL is required")
		exitCode = 1
		return
	}
	am := smqauthn.NewAuthNMiddleware(atomauthn.NewAuthentication())
	logger.Info("AuthN and AuthZ configured to use Atom")

	database := pgclient.NewDatabase(db, dbConfig, tracer)

	// Create new service
	svc, err := newService(ctx, atom.NewClient(atomCfg), database, tracer, logger, cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, am, bootstrap.NewConfigReader(), logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start servers
	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func newService(ctx context.Context, atomClient *atom.Client, database pgclient.Database, tracer trace.Tracer, logger *slog.Logger, cfg config) (bootstrap.Service, error) {
	repoConfig := bootstrappg.NewConfigRepository(database, logger)
	repoProfile := bootstrappg.NewProfileRepository(database, logger)
	repoBindings := bootstrappg.NewBindingRepository(database, logger)
	repoChallenges := bootstrappg.NewBootstrapChallengeRepository(database)
	if _, err := bootstrap.NewSecretCipher([]byte(cfg.DBEncryptionKey), cfg.DBEncryptionKeyID); err != nil {
		return nil, err
	}

	sdk := mgsdk.NewSDK(mgsdk.Config{})
	idp := uuid.New()
	resolver := bootstrap.NewAtomResolver(atomClient)
	renderer := bootstrap.NewRenderer()

	svc := bootstrap.NewWithChallenges(
		repoConfig,
		repoProfile,
		repoBindings,
		repoChallenges,
		resolver,
		renderer,
		sdk,
		[]byte(cfg.DBEncryptionKey),
		cfg.DBEncryptionKeyID,
		idp,
	)
	if err := bootstrap.ReconcileAtom(ctx, svc, atomClient); err != nil {
		return nil, fmt.Errorf("reconcile Bootstrap Atom projections: %w", err)
	}

	publisher, err := store.NewPublisher(ctx, cfg.ESURL, "bootstrap-es-pub")
	if err != nil {
		return nil, err
	}

	svc = bootstrap.WithAtom(svc, atomClient)
	svc = middleware.AtomAuthorizationMiddleware(svc, atomClient)
	svc = producer.NewEventStoreMiddleware(svc, publisher)
	svc = middleware.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc, nil
}
