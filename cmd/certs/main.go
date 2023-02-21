// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/certs/api"
	"github.com/mainflux/mainflux/certs/eventhandlers"
	vault "github.com/mainflux/mainflux/certs/pki"
	certsPg "github.com/mainflux/mainflux/certs/postgres"
	"github.com/mainflux/mainflux/internal"
	"github.com/mainflux/mainflux/internal/clients/events/things"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/internal/sqlxt"
	"github.com/mainflux/mainflux/logger"
	"golang.org/x/sync/errgroup"

	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
)

const (
	svcName    = "certs"
	esGroup    = "mainflux.certs"
	esConsumer = "certs"

	envPrefix     = "MF_CERTS_"
	envPrefixHttp = "MF_CERTS_HTTP_"
	envPrefixES   = "MF_CERTS_ES_"

	defDB          = "certs"
	defSvcHttpPort = "8204"
)

var (
	errFailedCertLoading     = errors.New("failed to load certificate")
	errFailedCertDecode      = errors.New("failed to decode certificate")
	errCACertificateNotExist = errors.New("CA certificate does not exist")
	errCAKeyNotExist         = errors.New("CA certificate key does not exist")
)

type config struct {
	LogLevel  string `env:"MF_CERTS_LOG_LEVEL"        envDefault:"info"`
	CertsURL  string `env:"MF_SDK_CERTS_URL"          envDefault:"http://localhost"`
	ThingsURL string `env:"MF_THINGS_URL"             envDefault:"http://things:8182"`
	JaegerURL string `env:"MF_JAEGER_URL"             envDefault:"localhost:6831"`

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
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	tlsCert, caCert, err := loadCertificates(cfg)
	if err != nil {
		logger.Error("Failed to load CA certificates for issuing client certs")
	}
	_ = tlsCert
	_ = caCert

	if cfg.PkiHost == "" {
		log.Fatalf("No host specified for PKI engine")
	}

	pkiClient, err := vault.NewVaultClient(cfg.PkiToken, cfg.PkiHost, cfg.PkiPath, cfg.PkiRole)
	if err != nil {
		log.Fatalf("failed to configure client for PKI engine")
	}

	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *certsPg.Migration(), dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	dbt := sqlxt.NewDatabase(db)

	certsRepo := certsPg.NewRepository(dbt)

	config := mfsdk.Config{
		CertsURL:  cfg.CertsURL,
		ThingsURL: cfg.ThingsURL,
	}
	sdk := mfsdk.NewSDK(config)

	idProvider := uuid.New()

	svc := certs.New(auth, certsRepo, idProvider, pkiClient, sdk)

	svc = api.NewLoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger), logger)

	thingsESClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer thingsESClient.Close()

	certsThingsHandler := eventhandlers.NewThingsEventHandlers(certsRepo, pkiClient)
	te := things.NewEventStore(certsThingsHandler, thingsESClient, esConsumer, logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return te.Subscribe(ctx, esGroup)
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Certs service terminated: %s", err))
	}
}

func loadCertificates(conf config) (tls.Certificate, *x509.Certificate, error) {
	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if conf.SignCAPath == "" || conf.SignCAKeyPath == "" {
		return tlsCert, caCert, nil
	}

	if _, err := os.Stat(conf.SignCAPath); os.IsNotExist(err) {
		return tlsCert, caCert, errCACertificateNotExist
	}

	if _, err := os.Stat(conf.SignCAKeyPath); os.IsNotExist(err) {
		return tlsCert, caCert, errCAKeyNotExist
	}

	tlsCert, err := tls.LoadX509KeyPair(conf.SignCAPath, conf.SignCAKeyPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertLoading, err)
	}

	b, err := os.ReadFile(conf.SignCAPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertLoading, err)
	}

	block, _ := pem.Decode(b)
	if block == nil {
		log.Fatalf("No PEM data found, failed to decode CA")
	}

	caCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(errFailedCertDecode, err)
	}

	return tlsCert, caCert, nil
}
