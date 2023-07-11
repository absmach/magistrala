// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains certs main function to start the certs service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"

	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/certs/api"
	vault "github.com/mainflux/mainflux/certs/pki"
	certsPg "github.com/mainflux/mainflux/certs/postgres"
	"github.com/mainflux/mainflux/internal"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users/policies"
	"golang.org/x/sync/errgroup"

	"github.com/jmoiron/sqlx"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
)

const (
	svcName        = "certs"
	envPrefix      = "MF_CERTS_"
	envPrefixHttp  = "MF_CERTS_HTTP_"
	defDB          = "certs"
	defSvcHttpPort = "9019"
)

type config struct {
	LogLevel      string `env:"MF_CERTS_LOG_LEVEL"        envDefault:"info"`
	CertsURL      string `env:"MF_SDK_CERTS_URL"          envDefault:"http://localhost"`
	ThingsURL     string `env:"MF_THINGS_URL"             envDefault:"http://things:9000"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"         envDefault:"true"`
	InstanceID    string `env:"MF_CERTS_INSTANCE_ID"      envDefault:""`
	JaegerURL     string `env:"MF_JAEGER_URL"             envDefault:"http://jaeger:14268/api/traces"`

	// Sign and issue certificates without 3rd party PKI
	SignCAPath    string `env:"MF_CERTS_SIGN_CA_PATH"        envDefault:"ca.crt"`
	SignCAKeyPath string `env:"MF_CERTS_SIGN_CA_KEY_PATH"    envDefault:"ca.key"`
	// used in pki mock , need to clean up certs in separate PR
	SignRSABits    int    `env:"MF_CERTS_SIGN_RSA_BITS,"     envDefault:""`
	SignHoursValid string `env:"MF_CERTS_SIGN_HOURS_VALID"   envDefault:"2048h"`

	// 3rd party PKI API access settings
	PkiHost  string `env:"MF_CERTS_VAULT_HOST"    envDefault:""`
	PkiPath  string `env:"MF_VAULT_PKI_INT_PATH"  envDefault:"pki_int"`
	PkiRole  string `env:"MF_VAULT_CA_ROLE_NAME"  envDefault:"mainflux"`
	PkiToken string `env:"MF_VAULT_TOKEN"         envDefault:""`
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

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID, err = uuid.New().ID()
		if err != nil {
			log.Fatalf("Failed to generate instanceID: %s", err)
		}
	}

	if cfg.PkiHost == "" {
		logger.Fatal("No host specified for PKI engine")
	}

	pkiClient, err := vault.NewVaultClient(cfg.PkiToken, cfg.PkiHost, cfg.PkiPath, cfg.PkiRole)
	if err != nil {
		logger.Fatal("failed to configure client for PKI engine")
	}

	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *certsPg.Migration(), dbConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer db.Close()

	auth, authHandler, err := authClient.Setup(envPrefix, svcName)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	svc := newService(auth, db, logger, cfg, pkiClient)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
	}
	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL, instanceID)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, instanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Certs service terminated: %s", err))
	}
}

func newService(auth policies.AuthServiceClient, db *sqlx.DB, logger mflog.Logger, cfg config, pkiAgent vault.Agent) certs.Service {
	certsRepo := certsPg.NewRepository(db, logger)
	config := mfsdk.Config{
		CertsURL:  cfg.CertsURL,
		ThingsURL: cfg.ThingsURL,
	}
	sdk := mfsdk.NewSDK(config)
	svc := certs.New(auth, certsRepo, sdk, pkiAgent)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	return svc
}
