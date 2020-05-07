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

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/messaging"
	mqttpub "github.com/mainflux/mainflux/messaging/mqtt"
	"github.com/mainflux/mainflux/messaging/nats"
	"github.com/mainflux/mainflux/mqtt"
	mqttredis "github.com/mainflux/mainflux/mqtt/redis"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	mp "github.com/mainflux/mproxy/pkg/mqtt"
	"github.com/mainflux/mproxy/pkg/session"
	ws "github.com/mainflux/mproxy/pkg/websocket"
	opentracing "github.com/opentracing/opentracing-go"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// Logging
	defLogLevel = "error"
	envLogLevel = "MF_MQTT_ADAPTER_LOG_LEVEL"
	// MQTT
	defMQTTHost             = "0.0.0.0"
	defMQTTPort             = "1883"
	defMQTTTargetHost       = "0.0.0.0"
	defMQTTTargetPort       = "1883"
	defMQTTForwarderTimeout = "30" // in seconds

	envMQTTHost             = "MF_MQTT_ADAPTER_MQTT_HOST"
	envMQTTPort             = "MF_MQTT_ADAPTER_MQTT_PORT"
	envMQTTTargetHost       = "MF_MQTT_ADAPTER_MQTT_TARGET_HOST"
	envMQTTTargetPort       = "MF_MQTT_ADAPTER_MQTT_TARGET_PORT"
	envMQTTForwarderTimeout = "MF_MQTT_ADAPTER_FORWARDER_TIMEOUT"
	// HTTP
	defHTTPHost       = "0.0.0.0"
	defHTTPPort       = "8080"
	defHTTPScheme     = "ws"
	defHTTPTargetHost = "localhost"
	defHTTPTargetPort = "8080"
	defHTTPTargetPath = "/mqtt"
	envHTTPHost       = "MF_MQTT_ADAPTER_WS_HOST"
	envHTTPPort       = "MF_MQTT_ADAPTER_WS_PORT"
	envHTTPScheme     = "MF_MQTT_ADAPTER_WS_SCHEMA"
	envHTTPTargetHost = "MF_MQTT_ADAPTER_WS_TARGET_HOST"
	envHTTPTargetPort = "MF_MQTT_ADAPTER_WS_TARGET_PORT"
	envHTTPTargetPath = "MF_MQTT_ADAPTER_WS_TARGET_PATH"
	// Things
	defThingsAuthURL     = "localhost:8181"
	defThingsAuthTimeout = "1" // in seconds
	envThingsAuthURL     = "MF_THINGS_AUTH_GRPC_URL"
	envThingsAuthTimeout = "MF_THINGS_AUTH_GRPC_TIMMEOUT"
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
)

type config struct {
	mqttHost             string
	mqttPort             string
	mqttTargetHost       string
	mqttTargetPort       string
	mqttForwarderTimeout time.Duration
	httpHost             string
	httpPort             string
	httpScheme           string
	httpTargetHost       string
	httpTargetPort       string
	httpTargetPath       string
	jaegerURL            string
	logLevel             string
	thingsURL            string
	thingsAuthURL        string
	thingsAuthTimeout    time.Duration
	natsURL              string
	clientTLS            bool
	caCerts              string
	instance             string
	esURL                string
	esPass               string
	esDB                 string
}

func main() {
	cfg := loadConfig()

	logger, err := mflog.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	tracer, closer := initJaeger("mproxy", cfg.jaegerURL, logger)
	defer closer.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	rc := connectToRedis(cfg.esURL, cfg.esPass, cfg.esDB, logger)
	defer rc.Close()

	cc := thingsapi.NewClient(conn, thingsTracer, cfg.thingsAuthTimeout)

	nps, err := nats.NewPubSub(cfg.natsURL, "mqtt", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nps.Close()
	mp, err := mqttpub.NewPublisher(fmt.Sprintf("%s:%s", cfg.mqttTargetHost, cfg.mqttTargetPort), cfg.mqttForwarderTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create MQTT publisher: %s", err))
		os.Exit(1)
	}
	fwd := mqtt.NewForwarder(nats.SubjectAllChannels, logger)
	if err := fwd.Forward(nps, mp); err != nil {
		logger.Error(fmt.Sprintf("Failed to forward NATS messages: %s", err))
		os.Exit(1)
	}

	np, err := nats.NewPublisher(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer np.Close()

	es := mqttredis.NewEventStore(rc, cfg.instance)

	// Event handler for MQTT hooks
	h := mqtt.NewHandler([]messaging.Publisher{np}, cc, es, logger, tracer)

	errs := make(chan error, 2)

	logger.Info(fmt.Sprintf("Starting MQTT proxy on port %s", cfg.mqttPort))
	go proxyMQTT(cfg, logger, h, errs)

	logger.Info(fmt.Sprintf("Starting MQTT over WS  proxy on port %s", cfg.httpPort))
	go proxyWS(cfg, logger, h, errs)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("mProxy terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	authTimeout, err := strconv.ParseInt(mainflux.Env(envThingsAuthTimeout, defThingsAuthTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
	}

	mqttTimeout, err := strconv.ParseInt(mainflux.Env(envMQTTForwarderTimeout, defMQTTForwarderTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsAuthTimeout, err.Error())
	}

	return config{
		mqttHost:             mainflux.Env(envMQTTHost, defMQTTHost),
		mqttPort:             mainflux.Env(envMQTTPort, defMQTTPort),
		mqttTargetHost:       mainflux.Env(envMQTTTargetHost, defMQTTTargetHost),
		mqttTargetPort:       mainflux.Env(envMQTTTargetPort, defMQTTTargetPort),
		mqttForwarderTimeout: time.Duration(mqttTimeout) * time.Second,
		httpHost:             mainflux.Env(envHTTPHost, defHTTPHost),
		httpPort:             mainflux.Env(envHTTPPort, defHTTPPort),
		httpScheme:           mainflux.Env(envHTTPScheme, defHTTPScheme),
		httpTargetHost:       mainflux.Env(envHTTPTargetHost, defHTTPTargetHost),
		httpTargetPort:       mainflux.Env(envHTTPTargetPort, defHTTPTargetPort),
		httpTargetPath:       mainflux.Env(envHTTPTargetPath, defHTTPTargetPath),
		jaegerURL:            mainflux.Env(envJaegerURL, defJaegerURL),
		thingsAuthURL:        mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		thingsAuthTimeout:    time.Duration(authTimeout) * time.Second,
		thingsURL:            mainflux.Env(envThingsAuthURL, defThingsAuthURL),
		natsURL:              mainflux.Env(envNatsURL, defNatsURL),
		logLevel:             mainflux.Env(envLogLevel, defLogLevel),
		clientTLS:            tls,
		caCerts:              mainflux.Env(envCACerts, defCACerts),
		instance:             mainflux.Env(envInstance, defInstance),
		esURL:                mainflux.Env(envESURL, defESURL),
		esPass:               mainflux.Env(envESPass, defESPass),
		esDB:                 mainflux.Env(envESDB, defESDB),
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

func proxyMQTT(cfg config, logger mflog.Logger, handler session.Handler, errs chan error) {
	address := fmt.Sprintf("%s:%s", cfg.mqttHost, cfg.mqttPort)
	target := fmt.Sprintf("%s:%s", cfg.mqttTargetHost, cfg.mqttTargetPort)
	mp := mp.New(address, target, handler, logger)

	errs <- mp.Proxy()
}
func proxyWS(cfg config, logger mflog.Logger, handler session.Handler, errs chan error) {
	target := fmt.Sprintf("%s:%s", cfg.httpTargetHost, cfg.httpTargetPort)
	wp := ws.New(target, cfg.httpTargetPath, cfg.httpScheme, handler, logger)
	http.Handle("/mqtt", wp.Handler())

	p := fmt.Sprintf(":%s", cfg.httpPort)
	errs <- http.ListenAndServe(p, nil)
}
