// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/emailer"
	"github.com/mainflux/mainflux/users/token"
	"github.com/mainflux/mainflux/users/tracing"

	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users/api"
	grpcapi "github.com/mainflux/mainflux/users/api/grpc"
	httpapi "github.com/mainflux/mainflux/users/api/http"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/postgres"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
)

const (
	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDBName        = "users"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8180"
	defGRPCPort      = "8181"
	defSecret        = "users"
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""

	defEmailLogLevel    = "debug"
	defEmailDriver      = "smtp"
	defEmailHost        = "localhost"
	defEmailPort        = "25"
	defEmailUsername    = "root"
	defEmailPassword    = ""
	defEmailFromAddress = ""
	defEmailFromName    = ""
	defEmailTemplate    = "../configs/resetPasswEmail.tmpl"

	defTokenSecret        = "mainflux-secret"
	defTokenDuration      = "5"
	defTokenResetEndpoint = "/reset-request" // URL where user lands after click on the reset link from email

	envLogLevel      = "MF_USERS_LOG_LEVEL"
	envDBHost        = "MF_USERS_DB_HOST"
	envDBPort        = "MF_USERS_DB_PORT"
	envDBUser        = "MF_USERS_DB_USER"
	envDBPass        = "MF_USERS_DB_PASS"
	envDBName        = "MF_USERS_DB"
	envDBSSLMode     = "MF_USERS_DB_SSL_MODE"
	envDBSSLCert     = "MF_USERS_DB_SSL_CERT"
	envDBSSLKey      = "MF_USERS_DB_SSL_KEY"
	envDBSSLRootCert = "MF_USERS_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_USERS_HTTP_PORT"
	envGRPCPort      = "MF_USERS_GRPC_PORT"
	envSecret        = "MF_USERS_SECRET"
	envServerCert    = "MF_USERS_SERVER_CERT"
	envServerKey     = "MF_USERS_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"

	envEmailDriver      = "MF_EMAIL_DRIVER"
	envEmailHost        = "MF_EMAIL_HOST"
	envEmailPort        = "MF_EMAIL_PORT"
	envEmailUsername    = "MF_EMAIL_USERNAME"
	envEmailPassword    = "MF_EMAIL_PASSWORD"
	envEmailFromAddress = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName    = "MF_EMAIL_FROM_NAME"
	envEmailLogLevel    = "MF_EMAIL_LOG_LEVEL"
	envEmailTemplate    = "MF_EMAIL_TEMPLATE"

	envTokenSecret        = "MF_TOKEN_SECRET"
	envTokenDuration      = "MF_TOKEN_DURATION"
	envTokenResetEndpoint = "MF_TOKEN_RESET_ENDPOINT"
)

type config struct {
	logLevel   string
	dbConfig   postgres.Config
	emailConf  email.Config
	tokenConf  tokenConfig
	httpPort   string
	grpcPort   string
	secret     string
	serverCert string
	serverKey  string
	jaegerURL  string
	resetURL   string
}

type tokenConfig struct {
	hmacSampleSecret []byte // secret for signing token
	tokenDuration    string // token in duration in min
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	tracer, closer := initJaeger("users", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("users_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, cfg, logger)
	errs := make(chan error, 2)

	go startHTTPServer(tracer, svc, cfg.httpPort, cfg.serverCert, cfg.serverKey, logger, errs)
	go startGRPCServer(tracer, svc, cfg.grpcPort, cfg.serverCert, cfg.serverKey, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Users service terminated: %s", err))
}

func loadConfig() config {
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

	emailConf := email.Config{
		Driver:      mainflux.Env(envEmailDriver, defEmailDriver),
		FromAddress: mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:    mainflux.Env(envEmailFromName, defEmailFromName),
		Host:        mainflux.Env(envEmailHost, defEmailHost),
		Port:        mainflux.Env(envEmailPort, defEmailPort),
		Username:    mainflux.Env(envEmailUsername, defEmailUsername),
		Password:    mainflux.Env(envEmailPassword, defEmailPassword),
		Template:    mainflux.Env(envEmailTemplate, defEmailTemplate),
	}

	tokenConf := tokenConfig{
		hmacSampleSecret: []byte(mainflux.Env(envTokenSecret, defTokenSecret)),
		tokenDuration:    mainflux.Env(envTokenDuration, defTokenDuration),
	}

	return config{
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:   dbConfig,
		emailConf:  emailConf,
		tokenConf:  tokenConf,
		httpPort:   mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:   mainflux.Env(envGRPCPort, defGRPCPort),
		secret:     mainflux.Env(envSecret, defSecret),
		serverCert: mainflux.Env(envServerCert, defServerCert),
		serverKey:  mainflux.Env(envServerKey, defServerKey),
		jaegerURL:  mainflux.Env(envJaegerURL, defJaegerURL),
		resetURL:   mainflux.Env(envTokenResetEndpoint, defTokenResetEndpoint),
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

func newService(db *sqlx.DB, tracer opentracing.Tracer, c config, logger logger.Logger) users.Service {
	database := postgres.NewDatabase(db)
	repo := tracing.UserRepositoryMiddleware(postgres.New(database), tracer)
	hasher := bcrypt.New()
	idp := jwt.New(c.secret)
	emailer, err := emailer.New(c.resetURL, &c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}
	tDur, err := strconv.Atoi(mainflux.Env(envTokenDuration, defTokenDuration))
	if err != nil {
		logger.Error(err.Error())
	}
	tokenizer := token.New(c.tokenConf.hmacSampleSecret, tDur)

	svc := users.New(repo, hasher, idp, emailer, tokenizer)
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

func startHTTPServer(tracer opentracing.Tracer, svc users.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if certFile != "" || keyFile != "" {
		logger.Info(fmt.Sprintf("Users service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		errs <- http.ListenAndServeTLS(p, certFile, keyFile, httpapi.MakeHandler(svc, tracer, logger))
	} else {
		logger.Info(fmt.Sprintf("Users service started using http, exposed port %s", port))
		errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc, tracer, logger))
	}
}

func startGRPCServer(tracer opentracing.Tracer, svc users.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}

	var server *grpc.Server
	if certFile != "" || keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load users certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Users gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Users gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	mainflux.RegisterUsersServiceServer(server, grpcapi.NewServer(tracer, svc))
	logger.Info(fmt.Sprintf("Users gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
