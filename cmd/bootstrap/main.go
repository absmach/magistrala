// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"crypto/aes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	rediscons "github.com/mainflux/mainflux/bootstrap/redis/consumer"
	redisprod "github.com/mainflux/mainflux/bootstrap/redis/producer"
	"github.com/mainflux/mainflux/logger"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/sync/errgroup"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	r "github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	api "github.com/mainflux/mainflux/bootstrap/api"
	"github.com/mainflux/mainflux/bootstrap/postgres"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	stopWaitTime  = 5 * time.Second
	httpProtocol  = "http"
	httpsProtocol = "https"

	defLogLevel       = "error"
	defDBHost         = "localhost"
	defDBPort         = "5432"
	defDBUser         = "mainflux"
	defDBPass         = "mainflux"
	defDB             = "bootstrap"
	defDBSSLMode      = "disable"
	defDBSSLCert      = ""
	defDBSSLKey       = ""
	defDBSSLRootCert  = ""
	defEncryptKey     = "12345678910111213141516171819202"
	defClientTLS      = "false"
	defCACerts        = ""
	defPort           = "8180"
	defServerCert     = ""
	defServerKey      = ""
	defBaseURL        = "http://localhost"
	defThingsPrefix   = ""
	defThingsESURL    = "localhost:6379"
	defThingsESPass   = ""
	defThingsESDB     = "0"
	defESURL          = "localhost:6379"
	defESPass         = ""
	defESDB           = "0"
	defESConsumerName = "bootstrap"
	defJaegerURL      = ""
	defAuthURL        = "localhost:8181"
	defAuthTimeout    = "1s"

	envLogLevel       = "MF_BOOTSTRAP_LOG_LEVEL"
	envDBHost         = "MF_BOOTSTRAP_DB_HOST"
	envDBPort         = "MF_BOOTSTRAP_DB_PORT"
	envDBUser         = "MF_BOOTSTRAP_DB_USER"
	envDBPass         = "MF_BOOTSTRAP_DB_PASS"
	envDB             = "MF_BOOTSTRAP_DB"
	envDBSSLMode      = "MF_BOOTSTRAP_DB_SSL_MODE"
	envDBSSLCert      = "MF_BOOTSTRAP_DB_SSL_CERT"
	envDBSSLKey       = "MF_BOOTSTRAP_DB_SSL_KEY"
	envDBSSLRootCert  = "MF_BOOTSTRAP_DB_SSL_ROOT_CERT"
	envEncryptKey     = "MF_BOOTSTRAP_ENCRYPT_KEY"
	envClientTLS      = "MF_BOOTSTRAP_CLIENT_TLS"
	envCACerts        = "MF_BOOTSTRAP_CA_CERTS"
	envPort           = "MF_BOOTSTRAP_PORT"
	envServerCert     = "MF_BOOTSTRAP_SERVER_CERT"
	envServerKey      = "MF_BOOTSTRAP_SERVER_KEY"
	envBaseURL        = "MF_SDK_BASE_URL"
	envThingsPrefix   = "MF_SDK_THINGS_PREFIX"
	envThingsESURL    = "MF_THINGS_ES_URL"
	envThingsESPass   = "MF_THINGS_ES_PASS"
	envThingsESDB     = "MF_THINGS_ES_DB"
	envESURL          = "MF_BOOTSTRAP_ES_URL"
	envESPass         = "MF_BOOTSTRAP_ES_PASS"
	envESDB           = "MF_BOOTSTRAP_ES_DB"
	envESConsumerName = "MF_BOOTSTRAP_EVENT_CONSUMER"
	envJaegerURL      = "MF_JAEGER_URL"
	envAuthURL        = "MF_AUTH_GRPC_URL"
	envAuthTimeout    = "MF_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel       string
	dbConfig       postgres.Config
	clientTLS      bool
	encKey         []byte
	caCerts        string
	httpPort       string
	serverCert     string
	serverKey      string
	baseURL        string
	thingsPrefix   string
	esThingsURL    string
	esThingsPass   string
	esThingsDB     string
	esURL          string
	esPass         string
	esDB           string
	esConsumerName string
	jaegerURL      string
	authURL        string
	authTimeout    time.Duration
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := mflog.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	thingsESConn := connectToRedis(cfg.esThingsURL, cfg.esThingsPass, cfg.esThingsDB, logger)
	defer thingsESConn.Close()

	esClient := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer esClient.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := connectToAuth(cfg, logger)
	defer authConn.Close()

	auth := authapi.NewClient(authTracer, authConn, cfg.authTimeout)

	svc := newService(auth, db, logger, esClient, cfg)

	g.Go(func() error {
		return startHTTPServer(ctx, svc, cfg, logger)
	})

	go subscribeToThingsES(svc, thingsESConn, cfg.esConsumerName, logger)

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Bootstrap service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		tls = false
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

	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
	}
	encKey, err := hex.DecodeString(mainflux.Env(envEncryptKey, defEncryptKey))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envEncryptKey, err.Error())
	}
	if err := os.Unsetenv(envEncryptKey); err != nil {
		log.Fatalf("Unable to unset %s value: %s", envEncryptKey, err.Error())
	}
	if _, err := aes.NewCipher(encKey); err != nil {
		log.Fatalf("Invalid %s value: %s", envEncryptKey, err.Error())
	}

	return config{
		logLevel:       mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:       dbConfig,
		clientTLS:      tls,
		encKey:         encKey,
		caCerts:        mainflux.Env(envCACerts, defCACerts),
		httpPort:       mainflux.Env(envPort, defPort),
		serverCert:     mainflux.Env(envServerCert, defServerCert),
		serverKey:      mainflux.Env(envServerKey, defServerKey),
		baseURL:        mainflux.Env(envBaseURL, defBaseURL),
		thingsPrefix:   mainflux.Env(envThingsPrefix, defThingsPrefix),
		esThingsURL:    mainflux.Env(envThingsESURL, defThingsESURL),
		esThingsPass:   mainflux.Env(envThingsESPass, defThingsESPass),
		esThingsDB:     mainflux.Env(envThingsESDB, defThingsESDB),
		esURL:          mainflux.Env(envESURL, defESURL),
		esPass:         mainflux.Env(envESPass, defESPass),
		esDB:           mainflux.Env(envESDB, defESDB),
		esConsumerName: mainflux.Env(envESConsumerName, defESConsumerName),
		jaegerURL:      mainflux.Env(envJaegerURL, defJaegerURL),
		authURL:        mainflux.Env(envAuthURL, defAuthURL),
		authTimeout:    authTimeout,
	}
}

func connectToDB(cfg postgres.Config, logger mflog.Logger) *sqlx.DB {
	db, err := postgres.Connect(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToRedis(redisURL, redisPass, redisDB string, logger mflog.Logger) *r.Client {
	db, err := strconv.Atoi(redisDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return r.NewClient(&r.Options{
		Addr:     redisURL,
		Password: redisPass,
		DB:       db,
	})
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func newService(auth mainflux.AuthServiceClient, db *sqlx.DB, logger mflog.Logger, esClient *r.Client, cfg config) bootstrap.Service {
	thingsRepo := postgres.NewConfigRepository(db, logger)

	config := mfsdk.Config{
		ThingsURL: cfg.baseURL,
	}

	sdk := mfsdk.NewSDK(config)

	svc := bootstrap.New(auth, thingsRepo, sdk, cfg.encKey)
	svc = redisprod.NewEventStoreMiddleware(svc, esClient)
	svc = api.NewLoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "bootstrap",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "bootstrap",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func connectToAuth(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
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

	return conn
}

func startHTTPServer(ctx context.Context, svc bootstrap.Service, cfg config, logger mflog.Logger) error {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svc, bootstrap.NewConfigReader(cfg.encKey), logger)}
	errCh := make(chan error)
	protocol := httpProtocol
	switch {
	case cfg.serverCert != "" || cfg.serverKey != "":
		logger.Info(fmt.Sprintf("Bootstrap service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		go func() {
			errCh <- server.ListenAndServeTLS(cfg.serverCert, cfg.serverKey)
		}()
		protocol = httpsProtocol

	default:
		logger.Info(fmt.Sprintf("Bootstrap service started using http on port %s", cfg.httpPort))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Bootstrap %s service error occurred during shutdown at %s: %s", protocol, p, err))
			return fmt.Errorf("bootstrap %s service error occurred during shutdown at %s: %w", protocol, p, err)
		}
		logger.Info(fmt.Sprintf("Bootstrap %s service shutdown of http at %s", protocol, p))
		return nil
	case err := <-errCh:
		return err
	}
}

func subscribeToThingsES(svc bootstrap.Service, client *r.Client, consumer string, logger mflog.Logger) {
	eventStore := rediscons.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	if err := eventStore.Subscribe(context.Background(), "mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Bootstrap service failed to subscribe to event sourcing: %s", err))
	}
}
