// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/emailer"
	"github.com/mainflux/mainflux/users/tracing"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	authapi "github.com/mainflux/mainflux/auth/api/grpc"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/postgres"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
)

const (
	stopWaitTime = 5 * time.Second

	defLogLevel      = "error"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDB            = "users"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defHTTPPort      = "8180"
	defServerCert    = ""
	defServerKey     = ""
	defJaegerURL     = ""

	defEmailHost        = "localhost"
	defEmailPort        = "25"
	defEmailUsername    = "root"
	defEmailPassword    = ""
	defEmailFromAddress = ""
	defEmailFromName    = ""
	defEmailTemplate    = "email.tmpl"
	defAdminEmail       = ""
	defAdminPassword    = ""
	defPassRegex        = "^.{8,}$"
	defAdminGroup       = "mainflux"

	defTokenResetEndpoint = "/reset-request" // URL where user lands after click on the reset link from email

	defAuthTLS     = "false"
	defAuthCACerts = ""
	defAuthURL     = "localhost:8181"
	defAuthTimeout = "1s"

	defSelfRegister = "true" // By default, everybody can create a user. Otherwise, only admin can create a user.

	envLogLevel      = "MF_USERS_LOG_LEVEL"
	envDBHost        = "MF_USERS_DB_HOST"
	envDBPort        = "MF_USERS_DB_PORT"
	envDBUser        = "MF_USERS_DB_USER"
	envDBPass        = "MF_USERS_DB_PASS"
	envDB            = "MF_USERS_DB"
	envDBSSLMode     = "MF_USERS_DB_SSL_MODE"
	envDBSSLCert     = "MF_USERS_DB_SSL_CERT"
	envDBSSLKey      = "MF_USERS_DB_SSL_KEY"
	envDBSSLRootCert = "MF_USERS_DB_SSL_ROOT_CERT"
	envHTTPPort      = "MF_USERS_HTTP_PORT"
	envServerCert    = "MF_USERS_SERVER_CERT"
	envServerKey     = "MF_USERS_SERVER_KEY"
	envJaegerURL     = "MF_JAEGER_URL"

	envAdminEmail    = "MF_USERS_ADMIN_EMAIL"
	envAdminPassword = "MF_USERS_ADMIN_PASSWORD"
	envPassRegex     = "MF_USERS_PASS_REGEX"

	envEmailHost        = "MF_EMAIL_HOST"
	envEmailPort        = "MF_EMAIL_PORT"
	envEmailUsername    = "MF_EMAIL_USERNAME"
	envEmailPassword    = "MF_EMAIL_PASSWORD"
	envEmailFromAddress = "MF_EMAIL_FROM_ADDRESS"
	envEmailFromName    = "MF_EMAIL_FROM_NAME"
	envEmailLogLevel    = "MF_EMAIL_LOG_LEVEL"
	envEmailTemplate    = "MF_EMAIL_TEMPLATE"

	envTokenResetEndpoint = "MF_TOKEN_RESET_ENDPOINT"

	envAuthTLS     = "MF_AUTH_CLIENT_TLS"
	envAuthCACerts = "MF_AUTH_CA_CERTS"
	envAuthURL     = "MF_AUTH_GRPC_URL"
	envAuthTimeout = "MF_AUTH_GRPC_TIMEOUT"

	envSelfRegister = "MF_USERS_ALLOW_SELF_REGISTER"
)

type config struct {
	logLevel      string
	dbConfig      postgres.Config
	emailConf     email.Config
	httpPort      string
	serverCert    string
	serverKey     string
	jaegerURL     string
	resetURL      string
	authTLS       bool
	authCACerts   string
	authURL       string
	authTimeout   time.Duration
	adminEmail    string
	adminPassword string
	passRegex     *regexp.Regexp
	selfRegister  bool
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	authTracer, closer := initJaeger("auth", cfg.jaegerURL, logger)
	defer closer.Close()

	auth, close := connectToAuth(cfg, authTracer, logger)
	if close != nil {
		defer close()
	}

	tracer, closer := initJaeger("users", cfg.jaegerURL, logger)
	defer closer.Close()

	dbTracer, dbCloser := initJaeger("users_db", cfg.jaegerURL, logger)
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, cfg, logger)

	g.Go(func() error {
		return startHTTPServer(ctx, tracer, svc, cfg.httpPort, cfg.serverCert, cfg.serverKey, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Users service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Users service terminated: %s", err))
	}
}

func loadConfig() config {
	authTimeout, err := time.ParseDuration(mainflux.Env(envAuthTimeout, defAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthTimeout, err.Error())
	}

	tls, err := strconv.ParseBool(mainflux.Env(envAuthTLS, defAuthTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envAuthTLS)
	}

	passRegex, err := regexp.Compile(mainflux.Env(envPassRegex, defPassRegex))
	if err != nil {
		log.Fatalf("Invalid password validation rules %s\n", envPassRegex)
	}

	selfRegister, err := strconv.ParseBool(mainflux.Env(envSelfRegister, defSelfRegister))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envSelfRegister, err.Error())
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

	emailConf := email.Config{
		FromAddress: mainflux.Env(envEmailFromAddress, defEmailFromAddress),
		FromName:    mainflux.Env(envEmailFromName, defEmailFromName),
		Host:        mainflux.Env(envEmailHost, defEmailHost),
		Port:        mainflux.Env(envEmailPort, defEmailPort),
		Username:    mainflux.Env(envEmailUsername, defEmailUsername),
		Password:    mainflux.Env(envEmailPassword, defEmailPassword),
		Template:    mainflux.Env(envEmailTemplate, defEmailTemplate),
	}

	return config{
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		dbConfig:      dbConfig,
		emailConf:     emailConf,
		httpPort:      mainflux.Env(envHTTPPort, defHTTPPort),
		serverCert:    mainflux.Env(envServerCert, defServerCert),
		serverKey:     mainflux.Env(envServerKey, defServerKey),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		resetURL:      mainflux.Env(envTokenResetEndpoint, defTokenResetEndpoint),
		authTLS:       tls,
		authCACerts:   mainflux.Env(envAuthCACerts, defAuthCACerts),
		authURL:       mainflux.Env(envAuthURL, defAuthURL),
		authTimeout:   authTimeout,
		adminEmail:    mainflux.Env(envAdminEmail, defAdminEmail),
		adminPassword: mainflux.Env(envAdminPassword, defAdminPassword),
		passRegex:     passRegex,
		selfRegister:  selfRegister,
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
func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func connectToAuth(cfg config, tracer opentracing.Tracer, logger logger.Logger) (mainflux.AuthServiceClient, func() error) {
	var opts []grpc.DialOption
	if cfg.authTLS {
		if cfg.authCACerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.authCACerts, "")
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

	conn, err := grpc.Dial(cfg.authURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to auth service: %s", err))
		os.Exit(1)
	}

	return authapi.NewClient(tracer, conn, cfg.authTimeout), conn.Close
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, auth mainflux.AuthServiceClient, c config, logger logger.Logger) users.Service {
	database := postgres.NewDatabase(db)
	hasher := bcrypt.New()
	userRepo := tracing.UserRepositoryMiddleware(postgres.NewUserRepo(database), tracer)

	emailer, err := emailer.New(c.resetURL, &c.emailConf)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	idProvider := uuid.New()

	svc := users.New(userRepo, hasher, auth, emailer, idProvider, c.passRegex)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "users",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)
	if err := createAdmin(svc, userRepo, c, auth); err != nil {
		logger.Error("failed to create admin user: " + err.Error())
		os.Exit(1)
	}

	switch c.selfRegister {
	case true:
		// If MF_USERS_ALLOW_SELF_REGISTER environment variable is "true",
		// everybody can create a new user. Here, check the existence of that
		// policy. If the policy does not exist, create it; otherwise, there is
		// no need to do anything further.
		_, err := auth.Authorize(context.Background(), &mainflux.AuthorizeReq{Obj: "user", Act: "create", Sub: "*"})
		if err != nil {
			// Add a policy that allows anybody to create a user
			apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "user", Act: "create", Sub: "*"})
			if err != nil {
				logger.Error("failed to add the policy related to MF_USERS_ALLOW_SELF_REGISTER: " + err.Error())
				os.Exit(1)
			}
			if !apr.GetAuthorized() {
				logger.Error("failed to authorized the policy result related to MF_USERS_ALLOW_SELF_REGISTER: " + errors.ErrAuthorization.Error())
				os.Exit(1)
			}
		}
	default:
		// If MF_USERS_ALLOW_SELF_REGISTER environment variable is "false",
		// everybody cannot create a new user. Therefore, delete a policy that
		// allows everybody to create a new user.
		dpr, err := auth.DeletePolicy(context.Background(), &mainflux.DeletePolicyReq{Obj: "user", Act: "create", Sub: "*"})
		if err != nil {
			logger.Error("failed to delete a policy: " + err.Error())
			os.Exit(1)
		}
		if !dpr.GetDeleted() {
			logger.Error("deleting a policy expected to succeed.")
			os.Exit(1)
		}
	}

	return svc
}

func createAdmin(svc users.Service, userRepo users.UserRepository, c config, auth mainflux.AuthServiceClient) error {
	user := users.User{
		Email:    c.adminEmail,
		Password: c.adminPassword,
	}

	if admin, err := userRepo.RetrieveByEmail(context.Background(), user.Email); err == nil {
		// The admin is already created. Check existence of the admin policy.
		_, err := auth.Authorize(context.Background(), &mainflux.AuthorizeReq{Obj: "authorities", Act: "member", Sub: admin.ID})
		if err != nil {
			apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "authorities", Act: "member", Sub: admin.ID})
			if err != nil {
				return err
			}
			if !apr.GetAuthorized() {
				return errors.ErrAuthorization
			}
		}
		return nil
	}

	// Add a policy that allows anybody to create a user
	apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "user", Act: "create", Sub: "*"})
	if err != nil {
		return err
	}
	if !apr.GetAuthorized() {
		return errors.ErrAuthorization
	}

	// Create an admin
	uid, err := svc.Register(context.Background(), "", user)
	if err != nil {
		return err
	}

	apr, err = auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "authorities", Act: "member", Sub: uid})
	if err != nil {
		return err
	}
	if !apr.GetAuthorized() {
		return errors.ErrAuthorization
	}

	return nil
}

func startHTTPServer(ctx context.Context, tracer opentracing.Tracer, svc users.Service, port string, certFile string, keyFile string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svc, tracer, logger)}

	switch {
	case certFile != "" || keyFile != "":
		logger.Info(fmt.Sprintf("Users service started using https, cert %s key %s, exposed port %s", certFile, keyFile, port))
		go func() {
			errCh <- server.ListenAndServeTLS(certFile, keyFile)
		}()
	default:
		logger.Info(fmt.Sprintf("Users service started using http, exposed port %s", port))
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Users service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("users service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("Users service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}

}
