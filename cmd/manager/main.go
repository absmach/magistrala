package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/api"
	"github.com/mainflux/mainflux/manager/bcrypt"
	"github.com/mainflux/mainflux/manager/cassandra"
	"github.com/mainflux/mainflux/manager/jwt"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	port        int    = 8180
	sep         string = ","
	defCluster  string = "127.0.0.1"
	defKeyspace string = "manager"
	defSecret   string = "manager"
	envCluster  string = "MANAGER_DB_CLUSTER"
	envKeyspace string = "MANAGER_DB_KEYSPACE"
	envSecret   string = "MANAGER_SECRET"
)

type config struct {
	Port     int
	Cluster  string
	Keyspace string
	Secret   string
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func main() {
	cfg := config{
		Port:     port,
		Cluster:  getenv(envCluster, defCluster),
		Keyspace: getenv(envKeyspace, defKeyspace),
		Secret:   getenv(envSecret, defSecret),
	}

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	session, err := cassandra.Connect(strings.Split(cfg.Cluster, sep), cfg.Keyspace)
	if err != nil {
		logger.Log("error", err)
		os.Exit(1)
	}
	defer session.Close()

	if err := cassandra.Initialize(session); err != nil {
		logger.Log("error", err)
		os.Exit(1)
	}

	users := cassandra.NewUserRepository(session)
	clients := cassandra.NewClientRepository(session)
	channels := cassandra.NewChannelRepository(session)
	hasher := bcrypt.NewHasher()
	idp := jwt.NewIdentityProvider(cfg.Secret)

	var svc manager.Service
	svc = manager.NewService(users, clients, channels, hasher, idp)
	svc = api.NewLoggingService(logger, svc)

	fields := []string{"method"}
	svc = api.NewMetricService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "manager",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fields),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "manager",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fields),
		svc,
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%d", cfg.Port)
		errs <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	logger.Log("terminated", <-errs)
}
