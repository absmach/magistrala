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
	defLogLevel   = "error"
	defDBHost     = "localhost"
	defDBPort     = "5432"
	defDBUser     = "mainflux"
	defDBPass     = "mainflux"
	defDBName     = "things"
	defDBSSLMode  = "disable"
	defCACerts    = ""
	defCacheURL   = "localhost:6379"
	defCachePass  = ""
	defCacheDB    = "0"
	defHTTPPort   = "8180"
	defGRPCPort   = "8181"
	defServerCert = ""
	defServerKey  = ""
	defUsersURL   = "localhost:8181"
	envLogLevel   = "MF_THINGS_LOG_LEVEL"
	envDBHost     = "MF_THINGS_DB_HOST"
	envDBPort     = "MF_THINGS_DB_PORT"
	envDBUser     = "MF_THINGS_DB_USER"
	envDBPass     = "MF_THINGS_DB_PASS"
	envDBName     = "MF_THINGS_DB"
	envDBSSLMode  = "MF_THINGS_DB_SSL_MODE"
	envCACerts    = "MF_THINGS_CA_CERTS"
	envCacheURL   = "MF_THINGS_CACHE_URL"
	envCachePass  = "MF_THINGS_CACHE_PASS"
	envCacheDB    = "MF_THINGS_CACHE_DB"
	envHTTPPort   = "MF_THINGS_HTTP_PORT"
	envGRPCPort   = "MF_THINGS_GRPC_PORT"
	envUsersURL   = "MF_USERS_URL"
	envServerCert = "MF_THINGS_SERVER_CERT"
	envServerKey  = "MF_THINGS_SERVER_KEY"
)

type config struct {
	LogLevel   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	DBSSLMode  string
	CACerts    string
	CacheURL   string
	CachePass  string
	CacheDB    string
	HTTPPort   string
	GRPCPort   string
	UsersURL   string
	ServerCert string
	ServerKey  string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	cache := connectToCache(cfg.CacheURL, cfg.CachePass, cfg.CacheDB, logger)

	db := connectToDB(cfg, logger)
	defer db.Close()

	conn := connectToUsersService(cfg, logger)
	defer conn.Close()

	svc := newService(conn, db, cache, logger)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg.HTTPPort, cfg.ServerCert, cfg.ServerKey, logger, errs)
	go startGRPCServer(svc, cfg.GRPCPort, cfg.ServerCert, cfg.ServerKey, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Things service terminated: %s", err))
}

func loadConfig() config {
	return config{
		LogLevel:   mainflux.Env(envLogLevel, defLogLevel),
		DBHost:     mainflux.Env(envDBHost, defDBHost),
		DBPort:     mainflux.Env(envDBPort, defDBPort),
		DBUser:     mainflux.Env(envDBUser, defDBUser),
		DBPass:     mainflux.Env(envDBPass, defDBPass),
		DBName:     mainflux.Env(envDBName, defDBName),
		DBSSLMode:  mainflux.Env(envDBSSLMode, defDBSSLMode),
		CACerts:    mainflux.Env(envCACerts, defCACerts),
		CacheURL:   mainflux.Env(envCacheURL, defCacheURL),
		CachePass:  mainflux.Env(envCachePass, defCachePass),
		CacheDB:    mainflux.Env(envCacheDB, defCacheDB),
		HTTPPort:   mainflux.Env(envHTTPPort, defHTTPPort),
		GRPCPort:   mainflux.Env(envGRPCPort, defGRPCPort),
		UsersURL:   mainflux.Env(envUsersURL, defUsersURL),
		ServerCert: mainflux.Env(envServerCert, defServerCert),
		ServerKey:  mainflux.Env(envServerKey, defServerKey),
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

func connectToDB(cfg config, logger logger.Logger) *sql.DB {
	db, err := postgres.Connect(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPass, cfg.DBSSLMode)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToUsersService(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.CACerts != "" {
		tpc, err := credentials.NewClientTLSFromFile(cfg.CACerts, "")
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create tls credentials: %s", err))
			os.Exit(1)
		}
		opts = append(opts, grpc.WithTransportCredentials(tpc))
	} else {
		opts = append(opts, grpc.WithInsecure())
		logger.Info("gRPC communication is not encrypted")
	}

	conn, err := grpc.Dial(cfg.UsersURL, opts...)
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

func startHTTPServer(svc things.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if certFile != "" || keyFile != "" {
		logger.Info(fmt.Sprintf("Things service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		errs <- http.ListenAndServeTLS(p, certFile, keyFile, httpapi.MakeHandler(svc))
	} else {
		logger.Info(fmt.Sprintf("Things service started using http on port %s", port))
		errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc))
	}
}

func startGRPCServer(svc things.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}

	var server *grpc.Server
	if certFile != "" || keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load things certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Things gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Things gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	mainflux.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
	errs <- server.Serve(listener)
}
