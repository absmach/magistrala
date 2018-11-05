package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/influxdb"
	thingsapi "github.com/mainflux/mainflux/things/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defThingsURL = "localhost:8181"
	defLogLevel  = "error"
	defPort      = "8180"
	defDBName    = "mainflux"
	defDBHost    = "localhost"
	defDBPort    = "8086"
	defDBUser    = "mainflux"
	defDBPass    = "mainflux"

	envThingsURL = "MF_THINGS_URL"
	envLogLevel  = "MF_INFLUX_READER_LOG_LEVEL"
	envPort      = "MF_INFLUX_READER_PORT"
	envDBName    = "MF_INFLUX_READER_DB_NAME"
	envDBHost    = "MF_INFLUX_READER_DB_HOST"
	envDBPort    = "MF_INFLUX_READER_DB_PORT"
	envDBUser    = "MF_INFLUX_READER_DB_USER"
	envDBPass    = "MF_INFLUX_READER_DB_PASS"
)

type config struct {
	ThingsURL string
	LogLevel  string
	Port      string
	DBName    string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
}

func main() {
	cfg, clientCfg := loadConfigs()
	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	conn := connectToThings(cfg.ThingsURL, logger)
	defer conn.Close()

	tc := thingsapi.NewClient(conn)

	client, err := influxdata.NewHTTPClient(clientCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	repo := newService(client, cfg.DBName, logger)

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPServer(repo, tc, cfg.Port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("InfluxDB writer service terminated: %s", err))
}

func loadConfigs() (config, influxdata.HTTPConfig) {
	cfg := config{
		ThingsURL: mainflux.Env(envThingsURL, defThingsURL),
		LogLevel:  mainflux.Env(envLogLevel, defLogLevel),
		Port:      mainflux.Env(envPort, defPort),
		DBName:    mainflux.Env(envDBName, defDBName),
		DBHost:    mainflux.Env(envDBHost, defDBHost),
		DBPort:    mainflux.Env(envDBPort, defDBPort),
		DBUser:    mainflux.Env(envDBUser, defDBUser),
		DBPass:    mainflux.Env(envDBPass, defDBPass),
	}

	clientCfg := influxdata.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%s", cfg.DBHost, cfg.DBPort),
		Username: cfg.DBUser,
		Password: cfg.DBPass,
	}

	return cfg, clientCfg
}

func connectToThings(url string, logger logger.Logger) *grpc.ClientConn {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(client influxdata.Client, dbName string, logger logger.Logger) readers.MessageRepository {
	repo, err := influxdb.New(client, dbName)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB writer: %s", err))
		os.Exit(1)
	}

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
