package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/api"
	provsdk "github.com/mainflux/mainflux/provision/sdk"
	mfSDK "github.com/mainflux/mainflux/sdk/go"
)

const (
	defLogLevel        = "debug"
	defClientTLS       = "false"
	defCACerts         = ""
	defServerCert      = ""
	defServerKey       = ""
	defThingsLocation  = "http://localhost"
	defUsersLocation   = "http://localhost"
	defMQTTURL         = "localhost:1883"
	defHTTPPort        = "8091"
	defMfUser          = "test@example.com"
	defMfPass          = "test"
	defThingIDs        = "aa942ec2-6f4e-45ab-a0cc-87cc3c64a55c"
	defMfBSURL         = "http://localhost:8202/things/configs"
	defMfWhiteListURL  = "http://localhost:8202/things/state"
	defMfCertsURL      = "https://k8s-aws.mainflux.com/certs"
	defProvisionCerts  = "false"
	defProvisionBS     = "true"
	defBSAutoWhitelist = "true"
	defBSContent       = `{}`

	envClientTLS        = "MF_ENV_CLIENTS_TLS"
	envCACerts          = "MF_PROVISION_CA_CERTS"
	envServerCert       = "MF_PROVISION_SERVER_CERT"
	envServerKey        = "MF_PROVISION_SERVER_KEY"
	envMQTTURL          = "MF_MQTT_URL"
	envUsersLocation    = "MF_USERS_LOCATION"
	envThingsLocation   = "MF_THINGS_LOCATION"
	envLogLevel         = "MF_PROVISION_LOG_LEVEL"
	envHTTPPort         = "MF_PROVISION_HTTP_PORT"
	envMfUser           = "MF_USER"
	envMfPass           = "MF_PASS"
	envThingIDs         = "MF_THING_IDS"
	envMfBSURL          = "MF_BS_SVC_URL"
	envMfBSWhiteListURL = "MF_BS_SVC_WHITELISTE_URL"
	envMfCertsURL       = "MF_CERTS_SVC_URL"
	envProvisionCerts   = "MF_X509_PROVISIONING"
	envProvisionBS      = "MF_BS_CONFIG_PROVISIONING"
	envBSAutoWhiteList  = "MF_BS_AUTO_WHITELIST"
	envBSContent        = "MF_BS_CONTENT"
)

type config struct {
	provision.Config
	logLevel         string
	clientTLS        bool
	caCerts          string
	serverCert       string
	serverKey        string
	httpPort         string
	mfBSURL          string
	mfCertsURL       string
	mfBSWhiteListURL string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	svc := provision.New(cfg.Config, logger)
	svc = api.NewLoggingMiddleware(svc, logger)

	errs := make(chan error, 2)

	go startHTTPServer(svc, cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Provision service terminated: %s", err))
}

func startHTTPServer(svc provision.Service, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.httpPort)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("Provision service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, api.MakeHandler(svc))
		return
	}
	logger.Info(fmt.Sprintf("Provision service started using http on port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf(fmt.Sprintf("Cannot determine TLS variable %s", err.Error()))
	}
	provisionX509, err := strconv.ParseBool(mainflux.Env(envProvisionCerts, defProvisionCerts))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envProvisionCerts)
	}
	provisionBS, err := strconv.ParseBool(mainflux.Env(envProvisionBS, defProvisionBS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envProvisionBS)
	}
	autoWhiteList, err := strconv.ParseBool(mainflux.Env(envBSAutoWhiteList, defBSAutoWhitelist))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envBSAutoWhiteList)
	}
	if autoWhiteList && !provisionBS {
		log.Fatalf("Can't auto whitelist if auto config save is off")
	}

	cfg := config{
		logLevel:         mainflux.Env(envLogLevel, defLogLevel),
		caCerts:          mainflux.Env(envCACerts, defCACerts),
		serverCert:       mainflux.Env(envServerCert, defServerCert),
		serverKey:        mainflux.Env(envServerKey, defServerKey),
		httpPort:         mainflux.Env(envHTTPPort, defHTTPPort),
		mfBSURL:          mainflux.Env(envMfBSURL, defMfBSURL),
		mfBSWhiteListURL: mainflux.Env(envMfBSWhiteListURL, defMfWhiteListURL),
		mfCertsURL:       mainflux.Env(envMfCertsURL, defMfCertsURL),
		Config: provision.Config{
			MFEmail:          mainflux.Env(envMfUser, defMfUser),
			MFPass:           mainflux.Env(envMfPass, defMfPass),
			PredefinedThings: strings.Split(mainflux.Env(envThingIDs, defThingIDs), ","),
			X509Provision:    provisionX509,
			BSProvision:      provisionBS,
			AutoWhiteList:    autoWhiteList,
			BSContent:        fmt.Sprintf(mainflux.Env(envBSContent, defBSContent), mainflux.Env(envMQTTURL, defMQTTURL)),
		},
	}
	thingSdkCfg := mfSDK.Config{
		BaseURL:           mainflux.Env(envThingsLocation, defThingsLocation),
		HTTPAdapterPrefix: "http",
		MsgContentType:    "application/json",
		TLSVerification:   tls,
	}
	thingsSDK := mfSDK.NewSDK(thingSdkCfg)

	userSdkCfg := mfSDK.Config{
		BaseURL:           mainflux.Env(envUsersLocation, defUsersLocation),
		HTTPAdapterPrefix: "http",
		MsgContentType:    "application/json",
		TLSVerification:   tls,
	}
	userSDK := mfSDK.NewSDK(userSdkCfg)

	cfg.SDK = provsdk.New(cfg.mfCertsURL, cfg.mfBSURL, cfg.mfBSWhiteListURL, thingsSDK, userSDK)
	return cfg
}
