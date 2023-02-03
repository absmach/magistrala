package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfSDK "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/api"
	"github.com/mainflux/mainflux/things"
	"golang.org/x/sync/errgroup"
)

const (
	svcName = "provision"

	defLogLevel        = "error"
	defConfigFile      = "config.toml"
	defTLS             = "false"
	defServerCert      = ""
	defServerKey       = ""
	defThingsURL       = "http://localhost"
	defUsersURL        = "http://localhost"
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

	envConfigFile       = "MF_PROVISION_CONFIG_FILE"
	envLogLevel         = "MF_PROVISION_LOG_LEVEL"
	envHTTPPort         = "MF_PROVISION_HTTP_PORT"
	envTLS              = "MF_PROVISION_ENV_CLIENTS_TLS"
	envServerCert       = "MF_PROVISION_SERVER_CERT"
	envServerKey        = "MF_PROVISION_SERVER_KEY"
	envUsersURL         = "MF_PROVISION_USERS_LOCATION"
	envThingsURL        = "MF_PROVISION_THINGS_LOCATION"
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

	contentType = "application/json"
)

var (
	errMissingConfigFile            = errors.New("missing config file setting")
	errFailLoadingConfigFile        = errors.New("failed to load config from file")
	errFailGettingAutoWhiteList     = errors.New("failed to get auto whitelist setting")
	errFailGettingCertSettings      = errors.New("failed to get certificate file setting")
	errFailGettingTLSConf           = errors.New("failed to get TLS setting")
	errFailGettingProvBS            = errors.New("failed to get BS url setting")
	errFailedToReadBootstrapContent = errors.New("failed to read bootstrap content from envs")
)

func main() {
	cfg, err := loadConfig()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

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
		ThingsURL:       cfg.Server.ThingsURL,
		BootstrapURL:    cfg.Server.MfBSURL,
		CertsURL:        cfg.Server.MfCertsURL,
		MsgContentType:  contentType,
		TLSVerification: cfg.Server.TLS,
	}
	SDK := mfSDK.NewSDK(SDKCfg)

	svc := provision.New(cfg, SDK, logger)
	svc = api.NewLoggingMiddleware(svc, logger)

	httpServerConfig := server.Config{Host: "", Port: cfg.Server.HTTPPort, KeyFile: cfg.Server.ServerKey, CertFile: cfg.Server.ServerCert}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Provision service terminated: %s", err))
	}

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

	var content map[string]interface{}
	if c := mainflux.Env(envBSContent, defBSContent); c != "" {
		if err = json.Unmarshal([]byte(c), &content); err != nil {
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
			ThingsURL:      mainflux.Env(envThingsURL, defThingsURL),
			UsersURL:       mainflux.Env(envUsersURL, defUsersURL),
			TLS:            tls,
		},
		Cert: provision.Cert{
			TTL: mainflux.Env(envCertsHoursValid, defCertsHoursValid),
		},
		Bootstrap: provision.Bootstrap{
			X509Provision: provisionX509,
			Provision:     provisionBS,
			AutoWhiteList: autoWhiteList,
			Content:       content,
		},

		// This is default conf for provision if there is no config file
		Channels: []things.Channel{
			{
				Name:     "control-channel",
				Metadata: map[string]interface{}{"type": "control"},
			}, {
				Name:     "data-channel",
				Metadata: map[string]interface{}{"type": "data"},
			},
		},
		Things: []things.Thing{
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
