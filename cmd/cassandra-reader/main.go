package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/cassandra"
	thingsapi "github.com/mainflux/mainflux/things/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	sep = ","

	defPort      = "8180"
	defCluster   = "127.0.0.1"
	defKeyspace  = "mainflux"
	defThingsURL = "localhost:8181"

	envPort      = "MF_CASSANDRA_READER_PORT"
	envCluster   = "MF_CASSANDRA_READER_DB_CLUSTER"
	envKeyspace  = "MF_CASSANDRA_READER_DB_KEYSPACE"
	envThingsURL = "MF_THINGS_URL"
)

type config struct {
	port      string
	cluster   string
	keyspace  string
	thingsURL string
}

func main() {
	cfg := loadConfig()

	logger := log.New(os.Stdout)

	session := connectToCassandra(cfg.cluster, cfg.keyspace, logger)
	defer session.Close()

	conn := connectToThings(cfg.thingsURL, logger)
	defer conn.Close()

	tc := thingsapi.NewClient(conn)
	repo := newService(session, logger)

	errs := make(chan error, 2)

	go startHTTPServer(repo, tc, cfg.port, errs, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err := <-errs
	logger.Error(fmt.Sprintf("Cassandra reader service terminated: %s", err))
}

func loadConfig() config {
	return config{
		port:      mainflux.Env(envPort, defPort),
		cluster:   mainflux.Env(envCluster, defCluster),
		keyspace:  mainflux.Env(envKeyspace, defKeyspace),
		thingsURL: mainflux.Env(envThingsURL, defThingsURL),
	}
}

func connectToCassandra(cluster, keyspace string, logger log.Logger) *gocql.Session {
	session, err := cassandra.Connect(strings.Split(cluster, sep), keyspace)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
}

func connectToThings(url string, logger log.Logger) *grpc.ClientConn {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(session *gocql.Session, logger log.Logger) readers.MessageRepository {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(repo readers.MessageRepository, tc mainflux.ThingsServiceClient, port string, errs chan error, logger log.Logger) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Cassandra reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, tc, "cassandra-reader"))
}
