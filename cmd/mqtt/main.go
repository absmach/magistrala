package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/mqtt"
	mqttredis "github.com/mainflux/mainflux/mqtt/redis"
	"github.com/mainflux/mainflux/pkg/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	mqttpub "github.com/mainflux/mainflux/pkg/messaging/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	mp "github.com/mainflux/mproxy/pkg/mqtt"
	"github.com/mainflux/mproxy/pkg/session"
	ws "github.com/mainflux/mproxy/pkg/websocket"
	opentracing "github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// Logging
	defLogLevel = "error"
	envLogLevel = "MF_MQTT_ADAPTER_LOG_LEVEL"
	// MQTT
	defMQTTPort              = "1883"
	defMQTTTargetHost        = "0.0.0.0"
	defMQTTTargetPort        = "1883"
	defMQTTForwarderTimeout  = "30s" // 30 seconds
	defMQTTTargetHealthCheck = ""
	envMQTTPort              = "MF_MQTT_ADAPTER_MQTT_PORT"
	envMQTTTargetHost        = "MF_MQTT_ADAPTER_MQTT_TARGET_HOST"
	envMQTTTargetPort        = "MF_MQTT_ADAPTER_MQTT_TARGET_PORT"
	envMQTTTargetHealthCheck = "MF_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK"
	envMQTTForwarderTimeout  = "MF_MQTT_ADAPTER_FORWARDER_TIMEOUT"
	// HTTP
	defHTTPPort       = "8080"
	defHTTPTargetHost = "localhost"
	defHTTPTargetPort = "8080"
	defHTTPTargetPath = "/mqtt"
	envHTTPPort       = "MF_MQTT_ADAPTER_WS_PORT"
	envHTTPTargetHost = "MF_MQTT_ADAPTER_WS_TARGET_HOST"
	envHTTPTargetPort = "MF_MQTT_ADAPTER_WS_TARGET_PORT"
	envHTTPTargetPath = "MF_MQTT_ADAPTER_WS_TARGET_PATH"
	// Things
	defThingsAuthURL     = "localhost:8183"
	defThingsAuthTimeout = "1s"
	envThingsAuthURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsAuthTimeout = "MF_THINGS_AUTH_GRPC_TIMEOUT"
	// Nats
	defNatsURL = "nats://localhost:4222"
	envNatsURL = "MF_NATS_URL"
	// Jaeger
	defJaegerURL = ""
	envJaegerURL = "MF_JAEGER_URL"
	// TLS
	defClientTLS = "false"
	defCACerts   = ""
	envClientTLS = "MF_MQTT_ADAPTER_CLIENT_TLS"
	envCACerts   = "MF_MQTT_ADAPTER_CA_CERTS"
	// Instance
	envInstance = "MF_MQTT_ADAPTER_INSTANCE"
	defInstance = ""
	// ES
	envESURL  = "MF_MQTT_ADAPTER_ES_URL"
	envESPass = "MF_MQTT_ADAPTER_ES_PASS"
	envESDB   = "MF_MQTT_ADAPTER_ES_DB"
	defESURL  = "localhost:6379"
	defESPass = ""
	defESDB   = "0"
	// Auth cache
	envAuthCacheURL  = "MF_AUTH_CACHE_URL"
	envAuthCachePass = "MF_AUTH_CACHE_PASS"
	envAuthCacheDB   = "MF_AUTH_CACHE_DB"
	defAuthcacheURL  = "localhost:6379"
	defAuthCachePass = ""
	defAuthCacheDB   = "0"
)

type config struct {
	mqttPort              string
	mqttTargetHost        string
	mqttTargetPort        string
	mqttForwarderTimeout  time.Duration
	mqttTargetHealthCheck string
	httpPort              string
	httpTargetHost        string
	httpTargetPort        string
	httpTargetPath        string
	jaegerURL             string
	logLevel              string
	thingsURL             string
	thingsAuthURL         string
	thingsAuthTimeout     time.Duration
	natsURL               string
	clientTLS             bool
	caCerts               string
	instance              string
	esURL                 string
	esPass                string
	esDB                  string
	authURL               string
	authPass              string
	authDB                string
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := mflog.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	if cfg.mqttTargetHealthCheck != "" {
		notify := func(e error, next time.Duration) {
			logger.Info(fmt.Sprintf("Broker not ready: %s, next try in %s", e.Error(), next))
		}

		err := backoff.RetryNotify(healthcheck(cfg), backoff.NewExponentialBackOff(), notify)
		if err != nil {
			logger.Info(fmt.Sprintf("MQTT healthcheck limit exceeded, exiting. %s ", err.Error()))
			os.Exit(1)
		}
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	ec := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer ec.Close()

	nps, err := nats.NewPubSub(cfg.natsURL, "mqtt", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nps.Close()

	mpub, err := mqttpub.NewPublisher(fmt.Sprintf("%s:%s", cfg.mqttTargetHost, cfg.mqttTargetPort), cfg.mqttForwarderTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create MQTT publisher: %s", err))
		os.Exit(1)
	}

	fwd := mqtt.NewForwarder(nats.SubjectAllChannels, logger)
	if err := fwd.Forward(nps, mpub); err != nil {
		logger.Error(fmt.Sprintf("Failed to forward NATS messages: %s", err))
		os.Exit(1)
	}

	np, err := nats.NewPublisher(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer np.Close()

	es := mqttredis.NewEventStore(ec, cfg.instance)

	ac := connectToRedis(cfg.authURL, cfg.authPass, cfg.authDB, logger)
	defer ac.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()
	tc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsAuthTimeout)

	authClient := auth.New(ac, tc)

	// Event handler for MQTT hooks
	h := mqtt.NewHandler([]messaging.Publisher{np}, es, logger, authClient)

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.mqttPort))
	g.Go(func() error {
		return proxyMQTT(ctx, cfg, logger, h)
	})

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.httpPort))
	g.Go(func() error {
		return proxyWS(ctx, cfg, logger, h)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("mProxy shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
	}
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authTimeout, err := time.ParseDuration(mainflux.Env(envThingsAuthTimeout, defThingsAuthTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
	}

	mqttTimeout, err := time.ParseDuration(mainflux.Env(envMQTTForwarderTimeout, defMQTTForwarderTimeout))
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envMQTTForwarderTimeout, err.Error())
	}

	return config{
		mqttPort:              mainflux.Env(envMQTTPort, defMQTTPort),
		mqttTargetHost:        mainflux.Env(envMQTTTargetHost, defMQTTTargetHost),
		mqttTargetPort:        mainflux.Env(envMQTTTargetPort, defMQTTTargetPort),
		mqttForwarderTimeout:  mqttTimeout,
		mqttTargetHealthCheck: mainflux.Env(envMQTTTargetHealthCheck, defMQTTTargetHealthCheck),
		httpPort:              mainflux.Env(envHTTPPort, defHTTPPort),
		httpTargetHost:        mainflux.Env(envHTTPTargetHost, defHTTPTargetHost),
		httpTargetPort:        mainflux.Env(envHTTPTargetPort, defHTTPTargetPort),
		httpTargetPath:        mainflux.Env(envHTTPTargetPath, defHTTPTargetPath),
		jaegerURL:             mainflux.Env(envJaegerURL, defJaegerURL),
		thingsAuthURL:         mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		thingsAuthTimeout:     authTimeout,
		thingsURL:             mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		natsURL:               mainflux.Env(envNatsURL, defNatsURL),
		logLevel:              mainflux.Env(envLogLevel, defLogLevel),
		clientTLS:             tls,
		caCerts:               mainflux.Env(envCACerts, defCACerts),
		instance:              mainflux.Env(envInstance, defInstance),
		esURL:                 mainflux.Env(envESURL, defESURL),
		esPass:                mainflux.Env(envESPass, defESPass),
		esDB:                  mainflux.Env(envESDB, defESDB),
		authURL:               mainflux.Env(envAuthCacheURL, defAuthcacheURL),
		authPass:              mainflux.Env(envAuthCachePass, defAuthCachePass),
		authDB:                mainflux.Env(envAuthCacheDB, defAuthCacheDB),
	}
}

func initJaeger(svcName, url string, logger mflog.Logger) (opentracing.Tracer, io.Closer) {
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

func connectToThings(cfg config, logger mflog.Logger) *grpc.ClientConn {
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

	conn, err := grpc.Dial(cfg.thingsAuthURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}

func connectToRedis(redisURL, redisPass, redisDB string, logger mflog.Logger) *redis.Client {
	db, err := strconv.Atoi(redisDB)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to redis: %s", err))
		os.Exit(1)
	}

	return redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisPass,
		DB:       db,
	})
}

func proxyMQTT(ctx context.Context, cfg config, logger mflog.Logger, handler session.Handler) error {
	address := fmt.Sprintf(":%s", cfg.mqttPort)
	target := fmt.Sprintf("%s:%s", cfg.mqttTargetHost, cfg.mqttTargetPort)
	mp := mp.New(address, target, handler, logger)

	errCh := make(chan error)
	go func() {
		errCh <- mp.Listen()
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}

}
func proxyWS(ctx context.Context, cfg config, logger mflog.Logger, handler session.Handler) error {
	target := fmt.Sprintf("%s:%s", cfg.httpTargetHost, cfg.httpTargetPort)
	wp := ws.New(target, cfg.httpTargetPath, "ws", handler, logger)
	http.Handle("/mqtt", wp.Handler())

	errCh := make(chan error)

	go func() {
		errCh <- wp.Listen(cfg.httpPort)
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}

func healthcheck(cfg config) func() error {
	return func() error {
		res, err := http.Get(cfg.mqttTargetHealthCheck)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return errors.New(string(body))
		}
		return nil
	}
}
