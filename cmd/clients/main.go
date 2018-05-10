package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/clients/api"
	grpcapi "github.com/mainflux/mainflux/clients/api/grpc"
	httpapi "github.com/mainflux/mainflux/clients/api/http"
	"github.com/mainflux/mainflux/clients/bcrypt"
	"github.com/mainflux/mainflux/clients/jwt"
	"github.com/mainflux/mainflux/clients/postgres"
	log "github.com/mainflux/mainflux/logger"
	usersapi "github.com/mainflux/mainflux/users/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defDBHost   = "localhost"
	defDBPort   = "5432"
	defDBUser   = "mainflux"
	defDBPass   = "mainflux"
	defDBName   = "clients"
	defHTTPPort = "8180"
	defGRPCPort = "8181"
	defUsersURL = "localhost:8181"
	defSecret   = "clients"
	envDBHost   = "MF_CLIENTS_DB_HOST"
	envDBPort   = "MF_CLIENTS_DB_PORT"
	envDBUser   = "MF_CLIENTS_DB_USER"
	envDBPass   = "MF_CLIENTS_DB_PASS"
	envDBName   = "MF_CLIENTS_DB"
	envHTTPPort = "MF_CLIENTS_HTTP_PORT"
	envGRPCPort = "MF_CLIENTS_GRPC_PORT"
	envUsersURL = "MF_USERS_URL"
	envSecret   = "MF_CLIENTS_SECRET"
)

type config struct {
	DBHost   string
	DBPort   string
	DBUser   string
	DBPass   string
	DBName   string
	HTTPPort string
	GRPCPort string
	UsersURL string
	Secret   string
}

func main() {
	cfg := loadConfig()

	logger := log.New(os.Stdout)

	db := connectToDB(cfg, logger)
	defer db.Close()

	conn := connectToUsersService(cfg.UsersURL, logger)
	defer conn.Close()

	svc := newService(conn, db, cfg.Secret, logger)
	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg.HTTPPort, logger, errs)
	go startGRPCServer(svc, cfg.GRPCPort, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err := <-errs
	logger.Error(fmt.Sprintf("Clients service terminated: %s", err))
}

func loadConfig() config {
	return config{
		DBHost:   mainflux.Env(envDBHost, defDBHost),
		DBPort:   mainflux.Env(envDBPort, defDBPort),
		DBUser:   mainflux.Env(envDBUser, defDBUser),
		DBPass:   mainflux.Env(envDBPass, defDBPass),
		DBName:   mainflux.Env(envDBName, defDBName),
		HTTPPort: mainflux.Env(envHTTPPort, defHTTPPort),
		GRPCPort: mainflux.Env(envGRPCPort, defGRPCPort),
		UsersURL: mainflux.Env(envUsersURL, defUsersURL),
		Secret:   mainflux.Env(envSecret, defSecret),
	}
}

func connectToDB(cfg config, logger log.Logger) *sql.DB {
	db, err := postgres.Connect(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPass)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToUsersService(usersAddr string, logger log.Logger) *grpc.ClientConn {
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to users service: %s", err))
		os.Exit(1)
	}
	return conn
}

func newService(conn *grpc.ClientConn, db *sql.DB, secret string, logger log.Logger) clients.Service {
	users := usersapi.NewClient(conn)
	clientsRepo := postgres.NewClientRepository(db, logger)
	channelsRepo := postgres.NewChannelRepository(db, logger)
	hasher := bcrypt.New()
	idp := jwt.New(secret)

	svc := clients.New(users, clientsRepo, channelsRepo, hasher, idp)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "clients",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "clients",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	return svc
}

func startHTTPServer(svc clients.Service, port string, logger log.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Clients service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, httpapi.MakeHandler(svc))
}

func startGRPCServer(svc clients.Service, port string, logger log.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", port, err))
	}
	server := grpc.NewServer()
	mainflux.RegisterClientsServiceServer(server, grpcapi.NewServer(svc))
	logger.Info(fmt.Sprintf("Clients gRPC service started, exposed port %s", port))
	errs <- server.Serve(listener)
}
