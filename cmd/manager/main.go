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
	kitconsul "github.com/go-kit/kit/sd/consul"
	stdconsul "github.com/hashicorp/consul/api"
	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/api"
	"github.com/mainflux/mainflux/manager/bcrypt"
	"github.com/mainflux/mainflux/manager/cassandra"
	"github.com/mainflux/mainflux/manager/jwt"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	uuid "github.com/satori/go.uuid"
)

const (
	port      int    = 9000
	dbKey     string = "cassandra"
	secretKey string = "manager/secret"
	keyspace  string = "manager"
	sep       string = ","
)

var (
	kv     *stdconsul.KV
	logger log.Logger
)

func main() {
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		logger.Log("status", "Cannot start the service: CONSUL_ADDR not set.")
		os.Exit(1)
	}

	consul, err := stdconsul.NewClient(&stdconsul.Config{
		Address: consulAddr,
	})

	if err != nil {
		status := fmt.Sprintf("Cannot connect to Consul due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	kv = consul.KV()

	asr := &stdconsul.AgentServiceRegistration{
		ID:                uuid.NewV4().String(),
		Name:              "manager",
		Tags:              []string{},
		Port:              port,
		Address:           "",
		EnableTagOverride: false,
	}

	sd := kitconsul.NewClient(consul)
	if err = sd.Register(asr); err != nil {
		status := fmt.Sprintf("Cannot register service due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	hosts := strings.Split(get(dbKey), sep)

	session, err := cassandra.Connect(hosts, keyspace)
	if err != nil {
		status := fmt.Sprintf("Cannot connect to Cassandra due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}
	defer session.Close()

	if err := cassandra.Initialize(session); err != nil {
		status := fmt.Sprintf("Cannot initialize Cassandra session due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	users := cassandra.NewUserRepository(session)
	clients := cassandra.NewClientRepository(session)
	channels := cassandra.NewChannelRepository(session)
	hasher := bcrypt.NewHasher()
	idp := jwt.NewIdentityProvider(get(secretKey))

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

	errChan := make(chan error, 10)

	go func() {
		p := fmt.Sprintf(":%d", port)
		logger.Log("status", "Manager started.")
		errChan <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			status := fmt.Sprintf("Manager stopped due to %s", err)
			logger.Log("status", status)
			sd.Deregister(asr)
			os.Exit(1)
		case <-sigChan:
			status := fmt.Sprintf("Manager terminated.")
			logger.Log("status", status)
			sd.Deregister(asr)
			os.Exit(0)
		}
	}
}

func get(key string) string {
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		status := fmt.Sprintf("Cannot retrieve %s due to %s", key, err)
		logger.Log("status", status)
		os.Exit(1)
	}

	return string(pair.Value)
}
