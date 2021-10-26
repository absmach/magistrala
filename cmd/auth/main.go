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
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	api "github.com/mainflux/mainflux/auth/api"
	grpcapi "github.com/mainflux/mainflux/auth/api/grpc"
	httpapi "github.com/mainflux/mainflux/auth/api/http"
	"github.com/mainflux/mainflux/auth/jwt"
	"github.com/mainflux/mainflux/auth/keto"
	"github.com/mainflux/mainflux/auth/postgres"
	"github.com/mainflux/mainflux/auth/tracing"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDB            = "auth"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8180"
	defGRPCPort      = "8181"
	defSecret        = "auth"
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""
	defKetoHost      = "mainflux-keto"
	defKetoWritePort = "4467"
	defKetoReadPort  = "4466"

	envLogLevel      = "MF_AUTH_LOG_LEVEL"
	envDBHost        = "MF_AUTH_DB_HOST"
	envDBPort        = "MF_AUTH_DB_PORT"
	envDBUser        = "MF_AUTH_DB_USER"
	envDBPass        = "MF_AUTH_DB_PASS"
	envDB            = "MF_AUTH_DB"
	envDBSSLMode     = "MF_AUTH_DB_SSL_MODE"
	envDBSSLCert     = "MF_AUTH_DB_SSL_CERT"
	envDBSSLKey      = "MF_AUTH_DB_SSL_KEY"
	envDBSSLRootCert = "MF_AUTH_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_AUTH_HTTP_PORT"
	envGRPCPort      = "MF_AUTH_GRPC_PORT"
	envSecret        = "MF_AUTH_SECRET"
	envServerCert    = "MF_AUTH_SERVER_CERT"
	envServerKey     = "MF_AUTH_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"
	envKetoHost      = "MF_KETO_HOST"
	envKetoWritePort = "MF_KETO_WRITE_REMOTE_PORT"
	envKetoReadPort  = "MF_KETO_READ_REMOTE_PORT"
)

type config struct {
	logLevel      string
	dbConfig      postgres.Config
	httpPort      string
	grpcPort      string
	secret        string
	serverCert    string
	serverKey     string
	jaegerURL     string
	resetURL      string
	ketoHost      string
	ketoWritePort string
	ketoReadPort  string
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

	tracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("auth_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	reader, writer := initKeto(cfg.ketoHost, cfg.ketoReadPort, cfg.ketoWritePort, logger)

	svc := newService(db, dbTracer, cfg.secret, logger, reader, writer)
	errs := make(chan error, 2)

	go startHTTPServer(tracer, svc, cfg.httpPort, cfg.serverCert, cfg.serverKey, logger, errs)
	go startGRPCServer(tracer, svc, cfg.grpcPort, cfg.serverCert, cfg.serverKey, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Authentication service terminated: %s", err))
}

func loadConfig() config {
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

	return config{
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:      dbConfig,
		httpPort:      mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:      mainflux.Env(envGRPCPort, defGRPCPort),
		secret:        mainflux.Env(envSecret, defSecret),
		serverCert:    mainflux.Env(envServerCert, defServerCert),
		serverKey:     mainflux.Env(envServerKey, defServerKey),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		ketoHost:      mainflux.Env(envKetoHost, defKetoHost),
		ketoReadPort:  mainflux.Env(envKetoReadPort, defKetoReadPort),
		ketoWritePort: mainflux.Env(envKetoWritePort, defKetoWritePort),
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

func initKeto(hostAddress, readPort, writePort string, logger logger.Logger) (acl.CheckServiceClient, acl.WriteServiceClient) {
	checkConn, err := grpc.Dial(fmt.Sprintf("%s:%s", hostAddress, readPort), grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to dial %s:%s for Keto Check Service: %s", hostAddress, readPort, err))
		os.Exit(1)
	}

	writeConn, err := grpc.Dial(fmt.Sprintf("%s:%s", hostAddress, writePort), grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to dial %s:%s for Keto Write Service: %s", hostAddress, writePort, err))
		os.Exit(1)
	}
	return acl.NewCheckServiceClient(checkConn), acl.NewWriteServiceClient(writeConn)
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, secret string, logger logger.Logger, reader acl.CheckServiceClient, writer acl.WriteServiceClient) auth.Service {
	database := postgres.NewDatabase(db)
	keysRepo := tracing.New(postgres.New(database), tracer)

	groupsRepo := postgres.NewGroupRepo(database)
	groupsRepo = tracing.GroupRepositoryMiddleware(tracer, groupsRepo)

	pa := keto.NewPolicyAgent(reader, writer)

	idProvider := uuid.New()
	t := jwt.New(secret)

	svc := auth.New(keysRepo, groupsRepo, idProvider, t, pa)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "auth",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(tracer opentracing.Tracer, svc auth.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if certFile != "" || keyFile != "" {
		logger.Info(fmt.Sprintf("Authentication service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		errs <- http.ListenAndServeTLS(p, certFile, keyFile, httpapi.MakeHandler(svc, tracer))
		return
	}
	logger.Info(fmt.Sprintf("Authentication service started using http, exposed port %s", port))
	errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc, tracer))

}

func startGRPCServer(tracer opentracing.Tracer, svc auth.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}

	var server *grpc.Server
	if certFile != "" || keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load auth certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Authentication gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Authentication gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	mainflux.RegisterAuthServiceServer(server, grpcapi.NewServer(tracer, svc))
	logger.Info(fmt.Sprintf("Authentication gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
