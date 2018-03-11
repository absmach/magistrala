package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/api"
	"github.com/mainflux/mainflux/manager/bcrypt"
	"github.com/mainflux/mainflux/manager/jwt"
	"github.com/mainflux/mainflux/manager/postgres"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	defDBHost string = "localhost"
	defDBPort string = "5432"
	defDBUser string = "mainflux"
	defDBPass string = "mainflux"
	defDBName string = "manager"
	defPort   string = "8180"
	defSecret string = "manager"
	envDBHost string = "MF_DB_HOST"
	envDBPort string = "MF_DB_PORT"
	envDBUser string = "MF_DB_USER"
	envDBPass string = "MF_DB_PASS"
	envDBName string = "MF_MANAGER_DB"
	envPort   string = "MF_MANAGER_PORT"
	envSecret string = "MF_MANAGER_SECRET"
)

type config struct {
	DBHost string
	DBPort string
	DBUser string
	DBPass string
	DBName string
	Port   string
	Secret string
}

func main() {
	cfg := config{
		DBHost: mainflux.Env(envDBHost, defDBHost),
		DBPort: mainflux.Env(envDBPort, defDBPort),
		DBUser: mainflux.Env(envDBUser, defDBUser),
		DBPass: mainflux.Env(envDBPass, defDBPass),
		DBName: mainflux.Env(envDBName, defDBName),
		Port:   mainflux.Env(envPort, defPort),
		Secret: mainflux.Env(envSecret, defSecret),
	}

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	db, err := postgres.Connect(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPass)
	if err != nil {
		logger.Log("error", err)
		os.Exit(1)
	}
	defer db.Close()

	users := postgres.NewUserRepository(db)
	clients := postgres.NewClientRepository(db)
	channels := postgres.NewChannelRepository(db)
	hasher := bcrypt.New()
	idp := jwt.New(cfg.Secret)

	svc := manager.New(users, clients, channels, hasher, idp)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "manager",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "manager",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%s", cfg.Port)
		errs <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	logger.Log("terminated", <-errs)
}
