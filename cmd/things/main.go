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
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/api"
	grpcapi "github.com/mainflux/mainflux/things/api/grpc"
	httpapi "github.com/mainflux/mainflux/things/api/http"
	"github.com/mainflux/mainflux/things/postgres"
	rediscache "github.com/mainflux/mainflux/things/redis"
	"github.com/mainflux/mainflux/things/uuid"
	usersapi "github.com/mainflux/mainflux/users/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBName        = "things"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defClientTLS     = "false"
	defCACerts       = ""
	defCacheURL      = "localhost:6379"
	defCachePass     = ""
	defCacheDB       = "0"
	defHTTPPort      = "8180"
	defGRPCPort      = "8181"
	defServerCert    = ""
	defServerKey     = ""
	defUsersURL      = "localhost:8181"
	envLogLevel      = "MF_THINGS_LOG_LEVEL"
	envDBHost        = "MF_THINGS_DB_HOST"
	envDBPort        = "MF_THINGS_DB_PORT"
	envDBUser        = "MF_THINGS_DB_USER"
	envDBPass        = "MF_THINGS_DB_PASS"
	envDBName        = "MF_THINGS_DB"
	envDBSSLMode     = "MF_THINGS_DB_SSL_MODE"
	envDBSSLCert     = "MF_THINGS_DB_SSL_CERT"
	envDBSSLKey      = "MF_THINGS_DB_SSL_KEY"
	envDBSSLRootCert = "MF_THINGS_DB_SSL_ROOT_CERT"
	envClientTLS     = "MF_THINGS_CLIENT_TLS"
	envCACerts       = "MF_THINGS_CA_CERTS"
	envCacheURL      = "MF_THINGS_CACHE_URL"
	envCachePass     = "MF_THINGS_CACHE_PASS"
	envCacheDB       = "MF_THINGS_CACHE_DB"
	envHTTPPort      = "MF_THINGS_HTTP_PORT"
	envGRPCPort      = "MF_THINGS_GRPC_PORT"
	envUsersURL      = "MF_USERS_URL"
	envServerCert    = "MF_THINGS_SERVER_CERT"
	envServerKey     = "MF_THINGS_SERVER_KEY"
)

type config struct {
	logLevel   string
	dbConfig   postgres.Config
	clientTLS  bool
	caCerts    string
	cacheURL   string
	cachePass  string
	cacheDB    string
	httpPort   string
	grpcPort   string
	usersURL   string
	serverCert string
	serverKey  string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	cache := connectToCache(cfg.cacheURL, cfg.cachePass, cfg.cacheDB, logger)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	conn := connectToUsers(cfg, logger)
	defer conn.Close()

	svc := newService(conn, db, cache, logger)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg, logger, errs)
	go startGRPCServer(svc, cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Things service terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
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
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:   dbConfig,
		clientTLS:  tls,
		caCerts:    mainflux.Env(envCACerts, defCACerts),
		cacheURL:   mainflux.Env(envCacheURL, defCacheURL),
		cachePass:  mainflux.Env(envCachePass, defCachePass),
		cacheDB:    mainflux.Env(envCacheDB, defCacheDB),
		httpPort:   mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:   mainflux.Env(envGRPCPort, defGRPCPort),
		usersURL:   mainflux.Env(envUsersURL, defUsersURL),
		serverCert: mainflux.Env(envServerCert, defServerCert),
		serverKey:  mainflux.Env(envServerKey, defServerKey),
	}
}

func connectToCache(cacheURL, cachePass string, cacheDB string, logger logger.Logger) *redis.Client {

	db, err := strconv.Atoi(cacheDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to cache: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(&redis.Options{
		Addr:     cacheURL,
		Password: cachePass,
		DB:       db,
	})
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sql.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
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

func newService(conn *grpc.ClientConn, db *sql.DB, client *redis.Client, logger logger.Logger) things.Service {
	users := usersapi.NewClient(conn)
	thingsRepo := postgres.NewThingRepository(db, logger)
	channelsRepo := postgres.NewChannelRepository(db, logger)
	chanCache := rediscache.NewChannelCache(client)
	thingCache := rediscache.NewThingCache(client)
	idp := uuid.New()

	svc := things.New(users, thingsRepo, channelsRepo, chanCache, thingCache, idp)
	svc = rediscache.NewEventStoreMiddleware(svc, client)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "things",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "things",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func startHTTPServer(svc things.Service, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("Things service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, httpapi.MakeHandler(svc))
	} else {
		logger.Info(fmt.Sprintf("Things service started using http on port %s", cfg.httpPort))
		errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc))
	}
}

func startGRPCServer(svc things.Service, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.grpcPort)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", cfg.grpcPort, err))
	}

	var server *grpc.Server
	if cfg.serverCert != "" || cfg.serverKey != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.serverCert, cfg.serverKey)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load things certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Things gRPC service started using https on port %s with cert %s key %s",
			cfg.grpcPort, cfg.serverCert, cfg.serverKey))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Things gRPC service started using http on port %s", cfg.grpcPort))
		server = grpc.NewServer()
	}

	mainflux.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
	errs <- server.Serve(listener)
}
