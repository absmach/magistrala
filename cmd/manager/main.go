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
)

const (
	port     int    = 9000
	keyspace string = "manager"
	sep      string = ","
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
		ID:                "",
		Name:              "manager",
		Tags:              []string{"prod"},
		Port:              port,
		Address:           address(),
		EnableTagOverride: false,
	}

	sd := kitconsul.NewClient(consul)
	if err = sd.Register(asr); err != nil {
		status := fmt.Sprintf("Cannot register service due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	hosts := strings.Split(get("cassandra"), sep)

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
	idp := jwt.NewIdentityProvider(get("manager/secret"))

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
		p := fmt.Sprintf(":%d", port)
		logger.Log("status", "Manager started.")
		errs <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	status := fmt.Sprintf("Manager stopped due to %s", <-errs)
	logger.Log("status", status)
	sd.Deregister(asr)
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

// TODO: retrieve proper IP address
func address() string {
	return "127.0.0.1"
}
