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
	defLogLevel  = "error"
	defDBHost    = "localhost"
	defDBPort    = "5432"
	defDBUser    = "mainflux"
	defDBPass    = "mainflux"
	defDBName    = "things"
	defCacheURL  = "localhost:6379"
	defCachePass = ""
	defCacheDB   = "0"
	defHTTPPort  = "8180"
	defGRPCPort  = "8181"
	defUsersURL  = "localhost:8181"
	envLogLevel  = "MF_THINGS_LOG_LEVEL"
	envDBHost    = "MF_THINGS_DB_HOST"
	envDBPort    = "MF_THINGS_DB_PORT"
	envDBUser    = "MF_THINGS_DB_USER"
	envDBPass    = "MF_THINGS_DB_PASS"
	envDBName    = "MF_THINGS_DB"
	envCacheURL  = "MF_THINGS_CACHE_URL"
	envCachePass = "MF_THINGS_CACHE_PASS"
	envCacheDB   = "MF_THINGS_CACHE_DB"
	envHTTPPort  = "MF_THINGS_HTTP_PORT"
	envGRPCPort  = "MF_THINGS_GRPC_PORT"
	envUsersURL  = "MF_USERS_URL"
)

type config struct {
	LogLevel  string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	CacheURL  string
	CachePass string
	CacheDB   string
	HTTPPort  string
	GRPCPort  string
	UsersURL  string
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

	conn := connectToUsersService(cfg.UsersURL, logger)
	defer conn.Close()

	svc := newService(conn, db, cache, logger)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg.HTTPPort, logger, errs)
	go startGRPCServer(svc, cfg.GRPCPort, logger, errs)

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
		LogLevel:  mainflux.Env(envLogLevel, defLogLevel),
		DBHost:    mainflux.Env(envDBHost, defDBHost),
		DBPort:    mainflux.Env(envDBPort, defDBPort),
		DBUser:    mainflux.Env(envDBUser, defDBUser),
		DBPass:    mainflux.Env(envDBPass, defDBPass),
		DBName:    mainflux.Env(envDBName, defDBName),
		CacheURL:  mainflux.Env(envCacheURL, defCacheURL),
		CachePass: mainflux.Env(envCachePass, defCachePass),
		CacheDB:   mainflux.Env(envCacheDB, defCacheDB),
		HTTPPort:  mainflux.Env(envHTTPPort, defHTTPPort),
		GRPCPort:  mainflux.Env(envGRPCPort, defGRPCPort),
		UsersURL:  mainflux.Env(envUsersURL, defUsersURL),
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
	db, err := postgres.Connect(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPass)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToUsersService(usersAddr string, logger logger.Logger) *grpc.ClientConn {
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
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

func startHTTPServer(svc things.Service, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Things service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc))
}

func startGRPCServer(svc things.Service, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}
	server := grpc.NewServer()
	mainflux.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
	logger.Info(fmt.Sprintf("Things gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
