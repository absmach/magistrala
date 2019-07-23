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
	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/influxdb"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defThingsURL     = "localhost:8181"
	defLogLevel      = "error"
	defPort          = "8180"
	defDBName        = "mainflux"
	defDBHost        = "localhost"
	defDBPort        = "8086"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defClientTLS     = "false"
	defCACerts       = ""
	defJaegerURL     = ""
	defThingsTimeout = "1" // in seconds

	envThingsURL     = "MF_THINGS_URL"
	envLogLevel      = "MF_INFLUX_READER_LOG_LEVEL"
	envPort          = "MF_INFLUX_READER_PORT"
	envDBName        = "MF_INFLUX_READER_DB_NAME"
	envDBHost        = "MF_INFLUX_READER_DB_HOST"
	envDBPort        = "MF_INFLUX_READER_DB_PORT"
	envDBUser        = "MF_INFLUX_READER_DB_USER"
	envDBPass        = "MF_INFLUX_READER_DB_PASS"
	envClientTLS     = "MF_INFLUX_READER_CLIENT_TLS"
	envCACerts       = "MF_INFLUX_READER_CA_CERTS"
	envJaegerURL     = "MF_JAEGER_URL"
	envThingsTimeout = "MF_INFLUX_READER_THINGS_TIMEOUT"
)

type config struct {
	thingsURL     string
	logLevel      string
	port          string
	dbName        string
	dbHost        string
	dbPort        string
	dbUser        string
	dbPass        string
	clientTLS     bool
	caCerts       string
	jaegerURL     string
	thingsTimeout time.Duration
}

func main() {
	cfg, clientCfg := loadConfigs()
	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	conn := connectToThings(cfg, logger)
	defer conn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsTimeout)

	client, err := influxdata.NewHTTPClient(clientCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	repo := newService(client, cfg.dbName, logger)

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPServer(repo, tc, cfg.port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("InfluxDB writer service terminated: %s", err))
}

func loadConfigs() (config, influxdata.HTTPConfig) {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	timeout, err := strconv.ParseInt(mainflux.Env(envThingsTimeout, defThingsTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	cfg := config{
		thingsURL:     mainflux.Env(envThingsURL, defThingsURL),
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		port:          mainflux.Env(envPort, defPort),
		dbName:        mainflux.Env(envDBName, defDBName),
		dbHost:        mainflux.Env(envDBHost, defDBHost),
		dbPort:        mainflux.Env(envDBPort, defDBPort),
		dbUser:        mainflux.Env(envDBUser, defDBUser),
		dbPass:        mainflux.Env(envDBPass, defDBPass),
		clientTLS:     tls,
		caCerts:       mainflux.Env(envCACerts, defCACerts),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		thingsTimeout: time.Duration(timeout) * time.Second,
	}

	clientCfg := influxdata.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%s", cfg.dbHost, cfg.dbPort),
		Username: cfg.dbUser,
		Password: cfg.dbPass,
	}

	return cfg, clientCfg
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

	conn, err := grpc.Dial(cfg.thingsURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
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

func newService(client influxdata.Client, dbName string, logger logger.Logger) readers.MessageRepository {
	repo := influxdb.New(client, dbName)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "influxdb",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "influxdb",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(repo readers.MessageRepository, tc mainflux.ThingsServiceClient, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("InfluxDB reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, tc, "influxdb-reader"))
}
