//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	"time"

	"github.com/mainflux/mainflux/things/tracing"

	"github.com/jmoiron/sqlx"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/api"
	authgrpcapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	authhttpapi "github.com/mainflux/mainflux/things/api/auth/http"
	thhttpapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/mainflux/mainflux/things/postgres"
	rediscache "github.com/mainflux/mainflux/things/redis"
	localusers "github.com/mainflux/mainflux/things/users"
	"github.com/mainflux/mainflux/things/uuid"
	usersapi "github.com/mainflux/mainflux/users/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
)

const (
	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDBName          = "things"
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defClientTLS       = "false"
	defCACerts         = ""
	defCacheURL        = "localhost:6379"
	defCachePass       = ""
	defCacheDB         = "0"
	defESURL           = "localhost:6379"
	defESPass          = ""
	defESDB            = "0"
	defHTTPPort        = "8180"
	defAuthHTTPPort    = "8989"
	defAuthGRPCPort    = "8181"
	defServerCert      = ""
	defServerKey       = ""
	defUsersURL        = "localhost:8181"
	defSingleUserEmail = ""
	defSingleUserToken = ""
	defJaegerURL       = ""
	defUsersTimeout    = "1" // in seconds

	envLogLevel        = "MF_THINGS_LOG_LEVEL"
	envDBHost          = "MF_THINGS_DB_HOST"
	envDBPort          = "MF_THINGS_DB_PORT"
	envDBUser          = "MF_THINGS_DB_USER"
	envDBPass          = "MF_THINGS_DB_PASS"
	envDBName          = "MF_THINGS_DB"
	envDBSSLMode       = "MF_THINGS_DB_SSL_MODE"
	envDBSSLCert       = "MF_THINGS_DB_SSL_CERT"
	envDBSSLKey        = "MF_THINGS_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_THINGS_DB_SSL_ROOT_CERT"
	envClientTLS       = "MF_THINGS_CLIENT_TLS"
	envCACerts         = "MF_THINGS_CA_CERTS"
	envCacheURL        = "MF_THINGS_CACHE_URL"
	envCachePass       = "MF_THINGS_CACHE_PASS"
	envCacheDB         = "MF_THINGS_CACHE_DB"
	envESURL           = "MF_THINGS_ES_URL"
	envESPass          = "MF_THINGS_ES_PASS"
	envESDB            = "MF_THINGS_ES_DB"
	envHTTPPort        = "MF_THINGS_HTTP_PORT"
	envAuthHTTPPort    = "MF_THINGS_AUTH_HTTP_PORT"
	envAuthGRPCPort    = "MF_THINGS_AUTH_GRPC_PORT"
	envUsersURL        = "MF_USERS_URL"
	envServerCert      = "MF_THINGS_SERVER_CERT"
	envServerKey       = "MF_THINGS_SERVER_KEY"
	envSingleUserEmail = "MF_THINGS_SINGLE_USER_EMAIL"
	envSingleUserToken = "MF_THINGS_SINGLE_USER_TOKEN"
	envJaegerURL       = "MF_JAEGER_URL"
	envUsersTimeout    = "MF_THINGS_USERS_TIMEOUT"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	clientTLS       bool
	caCerts         string
	cacheURL        string
	cachePass       string
	cacheDB         string
	esURL           string
	esPass          string
	esDB            string
	httpPort        string
	authHTTPPort    string
	authGRPCPort    string
	usersURL        string
	serverCert      string
	serverKey       string
	singleUserEmail string
	singleUserToken string
	jaegerURL       string
	usersTimeout    time.Duration
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	cacheClient := connectToRedis(cfg.cacheURL, cfg.cachePass, cfg.cacheDB, logger)

	esClient := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	usersTracer, usersCloser := initJaeger("users", cfg.jaegerURL, logger)
	defer usersCloser.Close()

	users, close := createUsersClient(cfg, usersTracer, logger)
	if close != nil {
		defer close()
	}

	dbTracer, dbCloser := initJaeger("things_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	cacheTracer, cacheCloser := initJaeger("things_cache", cfg.jaegerURL, logger)
	defer cacheCloser.Close()

	svc := newService(users, dbTracer, cacheTracer, db, cacheClient, esClient, logger)
	errs := make(chan error, 2)

	go startHTTPServer(thhttpapi.MakeHandler(thingsTracer, svc), cfg.httpPort, cfg, logger, errs)
	go startHTTPServer(authhttpapi.MakeHandler(thingsTracer, svc), cfg.authHTTPPort, cfg, logger, errs)
	go startGRPCServer(svc, thingsTracer, cfg, logger, errs)

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

	timeout, err := strconv.ParseInt(mainflux.Env(envUsersTimeout, defUsersTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envUsersTimeout, err.Error())
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
		logLevel:        mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:        dbConfig,
		clientTLS:       tls,
		caCerts:         mainflux.Env(envCACerts, defCACerts),
		cacheURL:        mainflux.Env(envCacheURL, defCacheURL),
		cachePass:       mainflux.Env(envCachePass, defCachePass),
		cacheDB:         mainflux.Env(envCacheDB, defCacheDB),
		esURL:           mainflux.Env(envESURL, defESURL),
		esPass:          mainflux.Env(envESPass, defESPass),
		esDB:            mainflux.Env(envESDB, defESDB),
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		authHTTPPort:    mainflux.Env(envAuthHTTPPort, defAuthHTTPPort),
		authGRPCPort:    mainflux.Env(envAuthGRPCPort, defAuthGRPCPort),
		usersURL:        mainflux.Env(envUsersURL, defUsersURL),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		singleUserEmail: mainflux.Env(envSingleUserEmail, defSingleUserEmail),
		singleUserToken: mainflux.Env(envSingleUserToken, defSingleUserToken),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
		usersTimeout:    time.Duration(timeout) * time.Second,
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func connectToRedis(cacheURL, cachePass string, cacheDB string, logger logger.Logger) *redis.Client {
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

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func createUsersClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.UsersServiceClient, func() error) {
	if cfg.singleUserEmail != "" && cfg.singleUserToken != "" {
		return localusers.NewSingleUserService(cfg.singleUserEmail, cfg.singleUserToken), nil
	}

	conn := connectToUsers(cfg, logger)
	return usersapi.NewClient(tracer, conn, cfg.usersTimeout), conn.Close
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

func newService(users mainflux.UsersServiceClient, dbTracer opentracing.Tracer, cacheTracer opentracing.Tracer, db *sqlx.DB, cacheClient *redis.Client, esClient *redis.Client, logger logger.Logger) things.Service {
	thingsRepo := postgres.NewThingRepository(db)
	thingsRepo = tracing.ThingRepositoryMiddleware(dbTracer, thingsRepo)

	channelsRepo := postgres.NewChannelRepository(db)
	channelsRepo = tracing.ChannelRepositoryMiddleware(dbTracer, channelsRepo)

	chanCache := rediscache.NewChannelCache(cacheClient)
	chanCache = tracing.ChannelCacheMiddleware(cacheTracer, chanCache)

	thingCache := rediscache.NewThingCache(cacheClient)
	thingCache = tracing.ThingCacheMiddleware(cacheTracer, thingCache)
	idp := uuid.New()

	svc := things.New(users, thingsRepo, channelsRepo, chanCache, thingCache, idp)
	svc = rediscache.NewEventStoreMiddleware(svc, esClient)
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

func startHTTPServer(handler http.Handler, port string, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("Things service started using https on port %s with cert %s key %s",
			port, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, handler)
		return
	}
	logger.Info(fmt.Sprintf("Things service started using http on port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, handler)
}

func startGRPCServer(svc things.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.authGRPCPort)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", cfg.authGRPCPort, err))
		os.Exit(1)
	}

	var server *grpc.Server
	if cfg.serverCert != "" || cfg.serverKey != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.serverCert, cfg.serverKey)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load things certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Things gRPC service started using https on port %s with cert %s key %s",
			cfg.authGRPCPort, cfg.serverCert, cfg.serverKey))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Things gRPC service started using http on port %s", cfg.authGRPCPort))
		server = grpc.NewServer()
	}

	mainflux.RegisterThingsServiceServer(server, authgrpcapi.NewServer(tracer, svc))
	errs <- server.Serve(listener)
}
