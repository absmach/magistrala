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
	"time"

	pgadapter "github.com/casbin/casbin-pg-adapter"
	"github.com/casbin/casbin/v2"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/authn/api/grpc"
	"github.com/mainflux/mainflux/authn/postgres"
	"github.com/mainflux/mainflux/authz"
	"github.com/mainflux/mainflux/authz/api"
	grpcapi "github.com/mainflux/mainflux/authz/api/grpc"
	httpapi "github.com/mainflux/mainflux/authz/api/http"
	"github.com/mainflux/mainflux/authz/api/pb"
	"github.com/mainflux/mainflux/logger"
	localusers "github.com/mainflux/mainflux/things/users"
	"github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defLogLevel        = "error"
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = "mainflux"
	defDBPass          = "mainflux"
	defDB              = "authz"
	defDBSSLMode       = "disable"
	defDBSSLCert       = ""
	defDBSSLKey        = ""
	defDBSSLRootCert   = ""
	defHTTPPort        = "8189"
	defGRPCPort        = "8187"
	defClientTLS       = "false"
	defCACerts         = ""
	defSingleUserEmail = ""
	defSingleUserToken = ""
	defServerCert      = ""
	defServerKey       = ""
	defJaegerURL       = ""
	defAuthnURL        = "localhost:8181"
	defAuthnTimeout    = "1s"
	defModelConf       = "model.conf"

	envLogLevel        = "MF_AUTHZ_LOG_LEVEL"
	envDBHost          = "MF_AUTHZ_DB_HOST"
	envDBPort          = "MF_AUTHZ_DB_PORT"
	envDBUser          = "MF_AUTHZ_DB_USER"
	envDBPass          = "MF_AUTHZ_DB_PASS"
	envDB              = "MF_AUTHZ_DB"
	envModelConf       = "MF_AUTHZ_MODEL_CONF"
	envDBSSLMode       = "MF_AUTHZ_DB_SSL_MODE"
	envDBSSLCert       = "MF_AUTHZ_DB_SSL_CERT"
	envDBSSLKey        = "MF_AUTHZ_DB_SSL_KEY"
	envDBSSLRootCert   = "MF_AUTHZ_DB_SSL_ROOT_CERT"
	envClientTLS       = "MF_AUTHZ_CLIENT_TLS"
	envCACerts         = "MF_AUTHZ_CA_CERTS"
	envHTTPPort        = "MF_AUTHZ_HTTP_PORT"
	envGRPCPort        = "MF_AUTHZ_GRPC_PORT"
	envSingleUserEmail = "MF_AUTHZ_SINGLE_USER_EMAIL"
	envSingleUserToken = "MF_AUTHZ_SINGLE_USER_TOKEN"
	envServerCert      = "MF_AUTHZ_SERVER_CERT"
	envServerKey       = "MF_AUTHZ_SERVER_KEY"
	envJaegerURL       = "MF_JAEGER_URL"
	envAuthnURL        = "MF_AUTHN_GRPC_URL"
	envAuthnTimeout    = "MF_AUTHN_GRPC_TIMEOUT"
)

type config struct {
	logLevel        string
	dbConfig        postgres.Config
	httpPort        string
	grpcPort        string
	secret          string
	serverCert      string
	serverKey       string
	jaegerURL       string
	resetURL        string
	clientTLS       bool
	caCerts         string
	modelConf       string
	singleUserToken string
	singleUserEmail string
	authnURL        string
	authnTimeout    time.Duration
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	dbcfg := cfg.dbConfig
	conn := fmt.Sprintf(`postgresql://%s:%s@%s:%s/%s?sslmode=disable`, dbcfg.User, dbcfg.Pass, dbcfg.Host, dbcfg.Port, dbcfg.Name)
	adapter, err := pgadapter.NewAdapter(conn)
	if err != nil {
		log.Fatalf(err.Error())
	}

	authTracer, authCloser := initJaeger("auth", cfg.jaegerURL, logger)
	defer authCloser.Close()

	auth, close := createAuthClient(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	enf, err := casbin.NewSyncedEnforcer(cfg.modelConf, adapter)
	if err != nil {
		log.Fatalf(err.Error())
	}
	enf.EnableAutoSave(true)

	tracer, closer := initJaeger("authz", cfg.jaegerURL, logger)
	defer closer.Close()
	svc := newService(enf, auth, logger)
	errs := make(chan error, 2)

	go startHTTPServer(tracer, svc, cfg.httpPort, cfg.serverCert, cfg.serverKey, logger, errs)
	go startGRPCServer(tracer, svc, cfg.grpcPort, cfg.serverCert, cfg.serverKey, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Authorization service terminated: %s", err))
}

func loadConfig() config {
	authnTimeout, err := time.ParseDuration(mainflux.Env(envAuthnTimeout, defAuthnTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthnTimeout, err.Error())
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

	return config{
		logLevel:     mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:     dbConfig,
		httpPort:     mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:     mainflux.Env(envGRPCPort, defGRPCPort),
		serverCert:   mainflux.Env(envServerCert, defServerCert),
		serverKey:    mainflux.Env(envServerKey, defServerKey),
		jaegerURL:    mainflux.Env(envJaegerURL, defJaegerURL),
		authnURL:     mainflux.Env(envAuthnURL, defAuthnURL),
		modelConf:    mainflux.Env(envModelConf, defModelConf),
		authnTimeout: authnTimeout,
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

func createAuthClient(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthNServiceClient, func() error) {
	if cfg.singleUserEmail != "" && cfg.singleUserToken != "" {
		return localusers.NewSingleUserService(cfg.singleUserEmail, cfg.singleUserToken), nil
	}

	conn := connectToAuth(cfg, logger)
	return authapi.NewClient(tracer, conn, cfg.authnTimeout), conn.Close
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

	conn, err := grpc.Dial(cfg.authnURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to authn service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(enf *casbin.SyncedEnforcer, auth mainflux.AuthNServiceClient, logger logger.Logger) authz.Service {
	svc := authz.New(enf, auth)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "authz",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "authz",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(tracer opentracing.Tracer, svc authz.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	if certFile != "" || keyFile != "" {
		logger.Info(fmt.Sprintf("Authorization service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		errs <- http.ListenAndServeTLS(p, certFile, keyFile, httpapi.MakeHandler(svc, tracer))
		return
	}
	logger.Info(fmt.Sprintf("Authorization service started using http, exposed port %s", port))
	errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc, tracer))
}

func startGRPCServer(tracer opentracing.Tracer, svc authz.Service, port string, certFile string, keyFile string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}

	var server *grpc.Server
	if certFile != "" || keyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load authz certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("Authorization gRPC service started using https on port %s with cert %s key %s", port, certFile, keyFile))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Authorization gRPC service started using http on port %s", port))
		server = grpc.NewServer()
	}

	pb.RegisterAuthZServiceServer(server, grpcapi.NewServer(tracer, svc))
	logger.Info(fmt.Sprintf("Authorization gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
