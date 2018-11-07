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
	"syscall"

	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	grpcapi "github.com/mainflux/mainflux/users/api/grpc"
	httpapi "github.com/mainflux/mainflux/users/api/http"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/postgres"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defLogLevel   = "error"
	defDBHost     = "localhost"
	defDBPort     = "5432"
	defDBUser     = "mainflux"
	defDBPass     = "mainflux"
	defDBName     = "users"
	defDBSSLMode  = "disable"
	defHTTPPort   = "8180"
	defGRPCPort   = "8181"
	defSecret     = "users"
	defServerCert = ""
	defServerKey  = ""
	envLogLevel   = "MF_USERS_LOG_LEVEL"
	envDBHost     = "MF_USERS_DB_HOST"
	envDBPort     = "MF_USERS_DB_PORT"
	envDBUser     = "MF_USERS_DB_USER"
	envDBPass     = "MF_USERS_DB_PASS"
	envDBName     = "MF_USERS_DB"
	envDBSSLMode  = "MF_USERS_DB_SSL_MODE"
	envHTTPPort   = "MF_USERS_HTTP_PORT"
	envGRPCPort   = "MF_USERS_GRPC_PORT"
	envSecret     = "MF_USERS_SECRET"
	envServerCert = "MF_USERS_SERVER_CERT"
	envServerKey  = "MF_USERS_SERVER_KEY"
)

type config struct {
	LogLevel   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	DBSSLMode  string
	HTTPPort   string
	GRPCPort   string
	Secret     string
	ServerCert string
	ServerKey  string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	db := connectToDB(cfg, logger)
	defer db.Close()

	svc := newService(db, cfg.Secret, logger)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg.HTTPPort, cfg.ServerCert, cfg.ServerKey, logger, errs)
	go startGRPCServer(svc, cfg.GRPCPort, cfg.ServerCert, cfg.ServerKey, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Users service terminated: %s", err))
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
		HTTPPort:   mainflux.Env(envHTTPPort, defHTTPPort),
		GRPCPort:   mainflux.Env(envGRPCPort, defGRPCPort),
		Secret:     mainflux.Env(envSecret, defSecret),
		ServerCert: mainflux.Env(envServerCert, defServerCert),
		ServerKey:  mainflux.Env(envServerKey, defServerKey),
	}
}

func connectToDB(cfg config, logger logger.Logger) *sql.DB {
	db, err := postgres.Connect(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPass, cfg.DBSSLMode)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sql.DB, secret string, logger logger.Logger) users.Service {
	repo := postgres.New(db)
	hasher := bcrypt.New()
	idp := jwt.New(secret)

	svc := users.New(repo, hasher, idp)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func startHTTPServer(svc users.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if certFile != "" || keyFile != "" {
		logger.Info(fmt.Sprintf("Things service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		errs <- http.ListenAndServeTLS(p, certFile, keyFile, httpapi.MakeHandler(svc, logger))
	} else {
		logger.Info(fmt.Sprintf("Things service started using http, exposed port %s", port))
		errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc, logger))
	}
}

func startGRPCServer(svc users.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
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
		logger.Info(fmt.Sprintf("Users gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Users gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	mainflux.RegisterUsersServiceServer(server, grpcapi.NewServer(svc))
	logger.Info(fmt.Sprintf("Users gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
