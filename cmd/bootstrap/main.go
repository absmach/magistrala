//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	r "github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	api "github.com/mainflux/mainflux/bootstrap/api"
	"github.com/mainflux/mainflux/bootstrap/postgres"
	"github.com/mainflux/mainflux/bootstrap/redis"
	"github.com/mainflux/mainflux/logger"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
	usersapi "github.com/mainflux/mainflux/users/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBName        = "bootstrap"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defClientTLS     = "false"
	defCACerts       = ""
	defPort          = "8180"
	defServerCert    = ""
	defServerKey     = ""
	defBaseURL       = "http://localhost"
	defThingsPrefix  = ""
	defUsersURL      = "localhost:8181"

	defESURL        = "localhost:6379"
	defESPass       = ""
	defESDB         = "0"
	defInstanceName = "bootstrap"

	envLogLevel      = "MF_BOOTSTRAP_LOG_LEVEL"
	envDBHost        = "MF_BOOTSTRAP_DB_HOST"
	envDBPort        = "MF_BOOTSTRAP_DB_PORT"
	envDBUser        = "MF_BOOTSTRAP_DB_USER"
	envDBPass        = "MF_BOOTSTRAP_DB_PASS"
	envDBName        = "MF_BOOTSTRAP_DB"
	envDBSSLMode     = "MF_BOOTSTRAP_DB_SSL_MODE"
	envDBSSLCert     = "MF_BOOTSTRAP_DB_SSL_CERT"
	envDBSSLKey      = "MF_BOOTSTRAP_DB_SSL_KEY"
	envDBSSLRootCert = "MF_BOOTSTRAP_DB_SSL_ROOT_CERT"
	envClientTLS     = "MF_BOOTSTRAP_CLIENT_TLS"
	envCACerts       = "MF_BOOTSTRAP_CA_CERTS"
	envPort          = "MF_BOOTSTRAP_PORT"
	envServerCert    = "MF_BOOTSTRAP_SERVER_CERT"
	envServerKey     = "MF_BOOTSTRAP_SERVER_KEY"
	envBaseURL       = "MF_SDK_BASE_URL"
	envThingsPrefix  = "MF_SDK_THINGS_PREFIX"
	envUsersURL      = "MF_USERS_URL"

	envESURL        = "MF_THINGS_ES_URL"
	envESPass       = "MF_THINGS_ES_PASS"
	envESDB         = "MF_THINGS_ES_DB"
	envInstanceName = "MF_BOOTSTRAP_INSTANCE_NAME"
)

type config struct {
	logLevel     string
	dbConfig     postgres.Config
	clientTLS    bool
	caCerts      string
	httpPort     string
	serverCert   string
	serverKey    string
	baseURL      string
	thingsPrefix string
	usersURL     string

	esURL        string
	esPass       string
	esDB         string
	instanceName string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	conn := connectToUsers(cfg, logger)
	defer conn.Close()

	esConn := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer esConn.Close()

	svc := newService(conn, db, logger, cfg)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg, logger, errs)
	go subscribeToThingsES(svc, esConn, cfg.instanceName, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
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
		Name:        mainflux.Env(envDBName, defDBName),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	return config{
		logLevel:     mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:     dbConfig,
		clientTLS:    tls,
		caCerts:      mainflux.Env(envCACerts, defCACerts),
		httpPort:     mainflux.Env(envPort, defPort),
		serverCert:   mainflux.Env(envServerCert, defServerCert),
		serverKey:    mainflux.Env(envServerKey, defServerKey),
		baseURL:      mainflux.Env(envBaseURL, defBaseURL),
		thingsPrefix: mainflux.Env(envThingsPrefix, defThingsPrefix),
		usersURL:     mainflux.Env(envUsersURL, defUsersURL),

		esURL:        mainflux.Env(envESURL, defESURL),
		esPass:       mainflux.Env(envESPass, defESPass),
		esDB:         mainflux.Env(envESDB, defESDB),
		instanceName: mainflux.Env(envInstanceName, defInstanceName),
	}
}

func connectToDB(cfg postgres.Config, logger logger.Logger) *sql.DB {
	db, err := postgres.Connect(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToRedis(redisURL, redisPass, redisDB string, logger logger.Logger) *r.Client {
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

func newService(conn *grpc.ClientConn, db *sql.DB, logger logger.Logger, cfg config) bootstrap.Service {
	thingsRepo := postgres.NewConfigRepository(db, logger)

	config := mfsdk.Config{
		BaseURL:      cfg.baseURL,
		ThingsPrefix: cfg.thingsPrefix,
	}

	sdk := mfsdk.NewSDK(config)
	users := usersapi.NewClient(conn)

	svc := bootstrap.New(users, thingsRepo, sdk)
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

func connectToUsers(cfg config, logger logger.Logger) *grpc.ClientConn {
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

	conn, err := grpc.Dial(cfg.usersURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to users service: %s", err))
		os.Exit(1)
	}

	return conn
}

func startHTTPServer(svc bootstrap.Service, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("Bootstrap service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, api.MakeHandler(svc, bootstrap.NewConfigReader()))
		return
	}
	logger.Info(fmt.Sprintf("Bootstrap service started using http on port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc, bootstrap.NewConfigReader()))
}

func subscribeToThingsES(svc bootstrap.Service, client *r.Client, consumer string, logger logger.Logger) {
	eventStore := redis.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	eventStore.Subscribe("mainflux.things")
}
