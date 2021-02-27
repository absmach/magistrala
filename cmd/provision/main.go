package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfSDK "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/api"
)

const (
	defLogLevel        = "debug"
	defConfigFile      = "config.toml"
	defTLS             = "false"
	defServerCert      = ""
	defServerKey       = ""
	defThingsLocation  = "http://localhost"
	defUsersLocation   = "http://localhost"
	defHTTPPort        = "8190"
	defMfUser          = "test@example.com"
	defMfPass          = "test"
	defMfAPIKey        = ""
	defMfBSURL         = "http://localhost:8202/things/configs"
	defMfWhiteListURL  = "http://localhost:8202/things/state"
	defMfCertsURL      = "http://localhost:8204"
	defProvisionCerts  = "false"
	defProvisionBS     = "true"
	defBSAutoWhitelist = "true"
	defBSContent       = ""
	defCertsHoursValid = "2400h"
	defCertsKeyBits    = "4096"

	envConfigFile       = "MF_PROVISION_CONFIG_FILE"
	envLogLevel         = "MF_PROVISION_LOG_LEVEL"
	envHTTPPort         = "MF_PROVISION_HTTP_PORT"
	envTLS              = "MF_PROVISION_ENV_CLIENTS_TLS"
	envServerCert       = "MF_PROVISION_SERVER_CERT"
	envServerKey        = "MF_PROVISION_SERVER_KEY"
	envUsersLocation    = "MF_PROVISION_USERS_LOCATION"
	envThingsLocation   = "MF_PROVISION_THINGS_LOCATION"
	envMfUser           = "MF_PROVISION_USER"
	envMfPass           = "MF_PROVISION_PASS"
	envMfAPIKey         = "MF_PROVISION_API_KEY"
	envMfCertsURL       = "MF_PROVISION_CERTS_SVC_URL"
	envProvisionCerts   = "MF_PROVISION_X509_PROVISIONING"
	envMfBSURL          = "MF_PROVISION_BS_SVC_URL"
	envMfBSWhiteListURL = "MF_PROVISION_BS_SVC_WHITELIST_URL"
	envProvisionBS      = "MF_PROVISION_BS_CONFIG_PROVISIONING"
	envBSAutoWhiteList  = "MF_PROVISION_BS_AUTO_WHITELIST"
	envBSContent        = "MF_PROVISION_BS_CONTENT"
	envCertsHoursValid  = "MF_PROVISION_CERTS_HOURS_VALID"
	envCertsKeyBits     = "MF_PROVISION_CERTS_RSA_BITS"
)

var (
	errMissingConfigFile            = errors.New("missing config file setting")
	errFailLoadingConfigFile        = errors.New("failed to load config from file")
	errFailGettingAutoWhiteList     = errors.New("failed to get auto whitelist setting")
	errFailGettingCertSettings      = errors.New("failed to get certificate file setting")
	errFailGettingTLSConf           = errors.New("failed to get TLS setting")
	errFailGettingProvBS            = errors.New("failed to get BS url setting")
	errFailSettingKeyBits           = errors.New("failed to set rsa number of bits")
	errFailedToReadBootstrapContent = errors.New("failed to read bootstrap content from envs")
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}
	logger, err := logger.New(os.Stdout, cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	if cfgFromFile, err := loadConfigFromFile(cfg.File); err != nil {
		logger.Warn(fmt.Sprintf("Continue with settings from env, failed to load from: %s: %s", cfg.File, err))
	} else {
		// Merge environment variables and file settings.
		mergeConfigs(&cfgFromFile, &cfg)
		cfg = cfgFromFile
		logger.Info("Continue with settings from file: " + cfg.File)
	}

	SDKCfg := mfSDK.Config{
		BaseURL:           cfg.Server.ThingsLocation,
		BootstrapURL:      cfg.Server.MfBSURL,
		CertsURL:          cfg.Server.MfCertsURL,
		HTTPAdapterPrefix: "http",
		MsgContentType:    "application/json",
		TLSVerification:   cfg.Server.TLS,
	}
	SDK := mfSDK.NewSDK(SDKCfg)

	svc := provision.New(cfg, SDK, logger)
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

func startHTTPServer(svc provision.Service, cfg provision.Config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.Server.HTTPPort)
	if cfg.Server.ServerCert != "" || cfg.Server.ServerKey != "" {
		logger.Info(fmt.Sprintf("Provision service started using https on port %s with cert %s key %s",
			cfg.Server.HTTPPort, cfg.Server.ServerCert, cfg.Server.ServerKey))
		errs <- http.ListenAndServeTLS(p, cfg.Server.ServerCert, cfg.Server.ServerKey, api.MakeHandler(svc))
		return
	}
	logger.Info(fmt.Sprintf("Provision service started using http on port %s", cfg.Server.HTTPPort))
	errs <- http.ListenAndServe(p, api.MakeHandler(svc))
}

func loadConfigFromFile(file string) (provision.Config, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		return provision.Config{}, errors.Wrap(errMissingConfigFile, err)
	}
	c, err := provision.Read(file)
	if err != nil {
		return provision.Config{}, errors.Wrap(errFailLoadingConfigFile, err)
	}
	return c, nil
}

func loadConfig() (provision.Config, error) {
	tls, err := strconv.ParseBool(mainflux.Env(envTLS, defTLS))
	if err != nil {
		return provision.Config{}, errors.Wrap(errFailGettingTLSConf, err)
	}
	provisionX509, err := strconv.ParseBool(mainflux.Env(envProvisionCerts, defProvisionCerts))
	if err != nil {
		return provision.Config{}, errors.Wrap(errFailGettingCertSettings, err)
	}
	provisionBS, err := strconv.ParseBool(mainflux.Env(envProvisionBS, defProvisionBS))
	if err != nil {
		return provision.Config{}, errors.Wrap(errFailGettingProvBS, fmt.Errorf(" for %s", envProvisionBS))
	}

	autoWhiteList, err := strconv.ParseBool(mainflux.Env(envBSAutoWhiteList, defBSAutoWhitelist))
	if err != nil {
		return provision.Config{}, errors.Wrap(errFailGettingAutoWhiteList, fmt.Errorf(" for %s", envBSAutoWhiteList))
	}
	if autoWhiteList && !provisionBS {
		return provision.Config{}, errors.New("Can't auto whitelist if auto config save is off")
	}
	keyBits, err := strconv.Atoi(mainflux.Env(envCertsKeyBits, defCertsKeyBits))
	if err != nil && provisionX509 == true {
		return provision.Config{}, errFailSettingKeyBits
	}

	var content map[string]interface{}
	if c := mainflux.Env(envBSContent, defBSContent); c != "" {
		if err = json.Unmarshal([]byte(c), content); err != nil {
			return provision.Config{}, errFailedToReadBootstrapContent
		}
	}

	cfg := provision.Config{
		Server: provision.ServiceConf{
			LogLevel:       mainflux.Env(envLogLevel, defLogLevel),
			ServerCert:     mainflux.Env(envServerCert, defServerCert),
			ServerKey:      mainflux.Env(envServerKey, defServerKey),
			HTTPPort:       mainflux.Env(envHTTPPort, defHTTPPort),
			MfBSURL:        mainflux.Env(envMfBSURL, defMfBSURL),
			MfWhiteListURL: mainflux.Env(envMfBSWhiteListURL, defMfWhiteListURL),
			MfCertsURL:     mainflux.Env(envMfCertsURL, defMfCertsURL),
			MfUser:         mainflux.Env(envMfUser, defMfUser),
			MfPass:         mainflux.Env(envMfPass, defMfPass),
			MfAPIKey:       mainflux.Env(envMfAPIKey, defMfAPIKey),
			ThingsLocation: mainflux.Env(envThingsLocation, defThingsLocation),
			UsersLocation:  mainflux.Env(envUsersLocation, defUsersLocation),
			TLS:            tls,
		},
		Certs: provision.Certs{
			HoursValid: mainflux.Env(envCertsHoursValid, defCertsHoursValid),
			KeyBits:    keyBits,
		},
		Bootstrap: provision.Bootstrap{
			X509Provision: provisionX509,
			Provision:     provisionBS,
			AutoWhiteList: autoWhiteList,
			Content:       content,
		},

		// This is default conf for provision if there is no config file
		Channels: []provision.Channel{
			{
				Name:     "control-channel",
				Metadata: map[string]interface{}{"type": "control"},
			}, {
				Name:     "data-channel",
				Metadata: map[string]interface{}{"type": "data"},
			},
		},
		Things: []provision.Thing{
			{
				Name:     "thing",
				Metadata: map[string]interface{}{"external_id": "xxxxxx"},
			},
		},
	}

	cfg.File = mainflux.Env(envConfigFile, defConfigFile)
	return cfg, nil
}

func mergeConfigs(dst, src interface{}) interface{} {
	d := reflect.ValueOf(dst).Elem()
	s := reflect.ValueOf(src).Elem()

	for i := 0; i < d.NumField(); i++ {
		dField := d.Field(i)
		sField := s.Field(i)
		switch dField.Kind() {
		case reflect.Struct:
			dst := dField.Addr().Interface()
			src := sField.Addr().Interface()
			m := mergeConfigs(dst, src)
			val := reflect.ValueOf(m).Elem().Interface()
			dField.Set(reflect.ValueOf(val))
		case reflect.Slice:
		case reflect.Bool:
			if dField.Interface() == false {
				dField.Set(reflect.ValueOf(sField.Interface()))
			}
		case reflect.Int:
			if dField.Interface() == 0 {
				dField.Set(reflect.ValueOf(sField.Interface()))
			}
		case reflect.String:
			if dField.Interface() == "" {
				dField.Set(reflect.ValueOf(sField.Interface()))
			}
		}
	}
	return dst
}
