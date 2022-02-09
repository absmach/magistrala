// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/timescale"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	svcName = "timescaledb-reader"

	defLogLevel          = "error"
	defPort              = "8911"
	defClientTLS         = "false"
	defCACerts           = ""
	defDBHost            = "localhost"
	defDBPort            = "5432"
	defDBUser            = "mainflux"
	defDBPass            = "mainflux"
	defDB                = "mainflux"
	defDBSSLMode         = "disable"
	defDBSSLCert         = ""
	defDBSSLKey          = ""
	defDBSSLRootCert     = ""
	defJaegerURL         = ""
	defThingsAuthURL     = "localhost:8183"
	defThingsAuthTimeout = "1s"

	envLogLevel          = "MF_TIMESCALE_READER_LOG_LEVEL"
	envPort              = "MF_TIMESCALE_READER_PORT"
	envClientTLS         = "MF_TIMESCALE_READER_CLIENT_TLS"
	envCACerts           = "MF_TIMESCALE_READER_CA_CERTS"
	envDBHost            = "MF_TIMESCALE_READER_DB_HOST"
	envDBPort            = "MF_TIMESCALE_READER_DB_PORT"
	envDBUser            = "MF_TIMESCALE_READER_DB_USER"
	envDBPass            = "MF_TIMESCALE_READER_DB_PASS"
	envDB                = "MF_TIMESCALE_READER_DB"
	envDBSSLMode         = "MF_TIMESCALE_READER_DB_SSL_MODE"
	envDBSSLCert         = "MF_TIMESCALE_READER_DB_SSL_CERT"
	envDBSSLKey          = "MF_TIMESCALE_READER_DB_SSL_KEY"
	envDBSSLRootCert     = "MF_TIMESCALE_READER_DB_SSL_ROOT_CERT"
	envJaegerURL         = "MF_JAEGER_URL"
	envThingsAuthURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsAuthTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
)

type config struct {
	logLevel          string
	port              string
	clientTLS         bool
	caCerts           string
	dbConfig          timescale.Config
	jaegerURL         string
	thingsAuthURL     string
	usersAuthURL      string
	thingsAuthTimeout time.Duration
	usersAuthTimeout  time.Duration
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	authConn := connectToAuth(cfg, logger)
	defer authConn.Close()
	auth := authapi.NewClient(authTracer, authConn, cfg.usersAuthTimeout)

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsAuthTimeout)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	repo := newService(db, logger)

	errs := make(chan error, 2)

	go startHTTPServer(repo, tc, auth, cfg.port, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Timescale reader service terminated: %s", err))
}

func loadConfig() config {
	dbConfig := timescale.Config{
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

	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authTimeout, err := time.ParseDuration(mainflux.Env(envThingsAuthTimeout, defThingsAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
	}

	return config{
		logLevel:          mainflux.Env(envLogLevel, defLogLevel),
		port:              mainflux.Env(envPort, defPort),
		clientTLS:         tls,
		caCerts:           mainflux.Env(envCACerts, defCACerts),
		dbConfig:          dbConfig,
		jaegerURL:         mainflux.Env(envJaegerURL, defJaegerURL),
		thingsAuthURL:     mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		thingsAuthTimeout: authTimeout,
	}
}

func connectToAuth(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	logger.Info("Connecting to auth via gRPC")
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

	conn, err := grpc.Dial(cfg.usersAuthURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("Established gRPC connection to auth via gRPC: %s", cfg.usersAuthURL))
	return conn
}

func connectToDB(dbConfig timescale.Config, logger logger.Logger) *sqlx.DB {
	db, err := timescale.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Timescale: %s", err))
		os.Exit(1)
	}
	return db
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

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.thingsAuthURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}

func newService(db *sqlx.DB, logger logger.Logger) readers.MessageRepository {
	svc := timescale.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "timescale",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "timescale",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(repo readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Timescale reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, tc, ac, svcName))
}
