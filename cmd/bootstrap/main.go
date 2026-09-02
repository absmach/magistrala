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
	"strings"

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
	LogLevel string `env:"MG_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	// Deliberately has no default: it seals every enrollment's bootstrap root
	// key at rest, so a built-in value would make every deployment that did
	// not override it decryptable from this source file.
	DBEncryptionKey   string `env:"MG_BOOTSTRAP_DB_ENCRYPTION_KEY,required"`
	DBEncryptionKeyID string `env:"MG_BOOTSTRAP_DB_ENCRYPTION_KEY_ID" envDefault:"primary"`
	// Retired keys, as "<keyID>:<key>" pairs, kept readable across a
	// rotation. Format: MG_BOOTSTRAP_DB_ENCRYPTION_PREVIOUS_KEYS=old:<32b>,older:<32b>
	DBEncryptionPreviousKeys []string `env:"MG_BOOTSTRAP_DB_ENCRYPTION_PREVIOUS_KEYS" envSeparator:","`
	JaegerURL                url.URL  `env:"MG_JAEGER_URL"                envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry            bool     `env:"MG_SEND_TELEMETRY"            envDefault:"true"`
	InstanceID               string   `env:"MG_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
	ESURL                    string   `env:"MG_ES_URL"                    envDefault:"amqp://guest:guest@localhost:5682/"`
	TraceRatio               float64  `env:"MG_JAEGER_TRACE_RATIO"        envDefault:"1.0"`
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

	idp := uuid.New()
	resolver := bootstrap.NewAtomResolver(atomClient)
	renderer := bootstrap.NewRenderer()

	previousKeys, err := parsePreviousKeys(cfg.DBEncryptionPreviousKeys)
	if err != nil {
		return nil, err
	}

	svc, err := bootstrap.New(
		repoConfig,
		repoProfile,
		repoBindings,
		repoChallenges,
		resolver,
		renderer,
		[]byte(cfg.DBEncryptionKey),
		cfg.DBEncryptionKeyID,
		idp,
		previousKeys...,
	)
	if err != nil {
		return nil, fmt.Errorf("init Bootstrap service: %w", err)
	}
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

// parsePreviousKeys turns "<keyID>:<key>" entries into retired encryption
// keys. They let the service open secrets sealed before a key rotation while
// new writes use the active key.
func parsePreviousKeys(entries []string) ([]bootstrap.PreviousKey, error) {
	var keys []bootstrap.PreviousKey
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		id, key, found := strings.Cut(entry, ":")
		if !found {
			return nil, fmt.Errorf("invalid previous encryption key %q: expected <keyID>:<key>", entry)
		}
		keys = append(keys, bootstrap.PreviousKey{ID: strings.TrimSpace(id), Key: []byte(key)})
	}
	return keys, nil
}
