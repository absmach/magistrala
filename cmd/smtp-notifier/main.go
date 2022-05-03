// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/consumers/notifiers/api"
	"github.com/mainflux/mainflux/consumers/notifiers/postgres"
	"github.com/mainflux/mainflux/consumers/notifiers/smtp"
	"github.com/mainflux/mainflux/consumers/notifiers/tracing"
	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/mainflux/pkg/ulid"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime = 5 * time.Second

	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDB            = "subscriptions"
	defConfigPath    = "/config.toml"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8906"
	defServerCert    = ""
	defServerKey     = ""
	defFrom          = ""
	defJaegerURL     = ""
	defNatsURL       = "nats://localhost:4222"

	defEmailHost        = "localhost"
	defEmailPort        = "25"
	defEmailUsername    = "root"
	defEmailPassword    = ""
	defEmailFromAddress = ""
	defEmailFromName    = ""
	defEmailTemplate    = "email.tmpl"

	defAuthTLS     = "false"
	defAuthCACerts = ""
	defAuthURL     = "localhost:8181"
	defAuthTimeout = "1s"

	envLogLevel      = "MF_SMTP_NOTIFIER_LOG_LEVEL"
	envDBHost        = "MF_SMTP_NOTIFIER_DB_HOST"
	envDBPort        = "MF_SMTP_NOTIFIER_DB_PORT"
	envDBUser        = "MF_SMTP_NOTIFIER_DB_USER"
	envDBPass        = "MF_SMTP_NOTIFIER_DB_PASS"
	envDB            = "MF_SMTP_NOTIFIER_DB"
	envConfigPath    = "MF_SMTP_NOTIFIER_CONFIG_PATH"
	envDBSSLMode     = "MF_SMTP_NOTIFIER_DB_SSL_MODE"
	envDBSSLCert     = "MF_SMTP_NOTIFIER_DB_SSL_CERT"
	envDBSSLKey      = "MF_SMTP_NOTIFIER_DB_SSL_KEY"
	envDBSSLRootCert = "MF_SMTP_NOTIFIER_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_SMTP_NOTIFIER_PORT"
	envServerCert    = "MF_SMTP_NOTIFIER_SERVER_CERT"
	envServerKey     = "MF_SMTP_NOTIFIER_SERVER_KEY"
	envFrom          = "MF_SMTP_NOTIFIER_FROM_ADDR"
	envJaegerURL     = "MF_JAEGER_URL"
	envNatsURL       = "MF_NATS_URL"

	envEmailHost        = "MF_EMAIL_HOST"
	envEmailPort        = "MF_EMAIL_PORT"
	envEmailUsername    = "MF_EMAIL_USERNAME"
	envEmailPassword    = "MF_EMAIL_PASSWORD"
	envEmailFromAddress = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName    = "MF_EMAIL_FROM_NAME"
	envEmailTemplate    = "MF_SMTP_NOTIFIER_TEMPLATE"

	envAuthTLS     = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts = "MF_AUTH_CA_CERTS"
	envAuthURL     = "MF_AUTH_GRPC_URL"
	envAuthTimeout = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	natsURL     string
	configPath  string
	logLevel    string
	dbConfig    postgres.Config
	emailConf   email.Config
	from        string
	httpPort    string
	serverCert  string
	serverKey   string
	jaegerURL   string
	authTLS     bool
	authCACerts string
	authURL     string
	authTimeout time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	pubSub, err := nats.NewPubSub(cfg.natsURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	authTracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	auth, close := connectToAuth(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	tracer, closer := initJaeger("smtp-notifier", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("smtp-notifier_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, cfg, logger)

	if err = consumers.Start(pubSub, svc, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Postgres writer: %s", err))
	}

	g.Go(func() error {
		return startHTTPServer(ctx, tracer, svc, cfg.httpPort, cfg.serverCert, cfg.serverKey, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("SMTP notifier service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("SMTP notifier service terminated: %s", err))
	}

}

func loadConfig() config {
	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
	}

	tls, err := strconv.ParseBool(mainflux.Env(envAuthTLS, defAuthTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthTLS)
	}

	dbConfig := postgres.Config{
		Host:        mainflux.Env(envDBHost, defDBHost),
		Port:        mainflux.Env(envDBPort, defDBPort),
		User:        mainflux.Env(envDBUser, defDBUser),
		Pass:        mainflux.Env(envDBPass, defDBPass),
		Name:        mainflux.Env(envDB, defDB),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	emailConf := email.Config{
		FromAddress: mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:    mainflux.Env(envEmailFromName, defEmailFromName),
		Host:        mainflux.Env(envEmailHost, defEmailHost),
		Port:        mainflux.Env(envEmailPort, defEmailPort),
		Username:    mainflux.Env(envEmailUsername, defEmailUsername),
		Password:    mainflux.Env(envEmailPassword, defEmailPassword),
		Template:    mainflux.Env(envEmailTemplate, defEmailTemplate),
	}

	return config{
		logLevel:    mainflux.Env(envLogLevel, defLogLevel),
		natsURL:     mainflux.Env(envNatsURL, defNatsURL),
		configPath:  mainflux.Env(envConfigPath, defConfigPath),
		dbConfig:    dbConfig,
		emailConf:   emailConf,
		from:        mainflux.Env(envFrom, defFrom),
		httpPort:    mainflux.Env(envHTTPPort, defHTTPPort),
		serverCert:  mainflux.Env(envServerCert, defServerCert),
		serverKey:   mainflux.Env(envServerKey, defServerKey),
		jaegerURL:   mainflux.Env(envJaegerURL, defJaegerURL),
		authTLS:     tls,
		authCACerts: mainflux.Env(envAuthCACerts, defAuthCACerts),
		authURL:     mainflux.Env(envAuthURL, defAuthURL),
		authTimeout: authTimeout,
	}

}

func initJaeger(svcName, url string, logger logger.Logger) (opentracing.Tracer, io.Closer) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil)
	}

	tracer, closer, err := jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToAuth(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthServiceClient, func() error) {
	var opts []grpc.DialOption
	if cfg.authTLS {
		if cfg.authCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.authCACerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create tls credentials: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		opts = append(opts, grpc.WithInsecure())
		logger.Info("gRPC communication is not encrypted")
	}

	conn, err := grpc.Dial(cfg.authURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return authapi.NewClient(tracer, conn, cfg.authTimeout), conn.Close
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, auth mainflux.AuthServiceClient, c config, logger logger.Logger) notifiers.Service {
	database := postgres.NewDatabase(db)
	repo := tracing.New(postgres.New(database), tracer)
	idp := ulid.New()

	agent, err := email.New(&c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create email agent: %s", err))
		os.Exit(1)
	}

	notifier := smtp.New(agent)
	svc := notifiers.New(auth, repo, idp, notifier, c.from)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "notifier",
			Subsystem: "smtp",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "notifier",
			Subsystem: "smtp",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func startHTTPServer(ctx context.Context, tracer opentracing.Tracer, svc notifiers.Service, port string, certFile string, keyFile string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svc, tracer, logger)}

	switch {
	case certFile != "" || keyFile != "":
		logger.Info(fmt.Sprintf("SMTP notifier service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		go func() {
			errCh <- server.ListenAndServeTLS(certFile, keyFile)
		}()
	default:
		logger.Info(fmt.Sprintf("SMTP notifier service started using http, exposed port %s", port))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("SMTP notifier service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("smtp notifier service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("SMTP notifier service  shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
