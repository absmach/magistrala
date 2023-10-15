// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains provision main function to start the provision service.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/provision"
	"github.com/mainflux/mainflux/provision/api"
	"golang.org/x/sync/errgroup"
)

const (
	svcName     = "provision"
	contentType = "application/json"
)

var (
	errMissingConfigFile            = errors.New("missing config file setting")
	errFailLoadingConfigFile        = errors.New("failed to load config from file")
	errFailedToReadBootstrapContent = errors.New("failed to read bootstrap content from envs")
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf(fmt.Sprintf("failed to load %s configuration : %s", svcName, err))
	}

	logger, err := mflog.New(os.Stdout, cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	if cfgFromFile, err := loadConfigFromFile(cfg.File); err != nil {
		logger.Warn(fmt.Sprintf("Continue with settings from env, failed to load from: %s: %s", cfg.File, err))
	} else {
		// Merge environment variables and file settings.
		mergeConfigs(&cfgFromFile, &cfg)
		cfg = cfgFromFile
		logger.Info("Continue with settings from file: " + cfg.File)
	}

	SDKCfg := mfsdk.Config{
		UsersURL:        cfg.Server.UsersURL,
		ThingsURL:       cfg.Server.ThingsURL,
		BootstrapURL:    cfg.Server.MfBSURL,
		CertsURL:        cfg.Server.MfCertsURL,
		MsgContentType:  contentType,
		TLSVerification: cfg.Server.TLS,
	}
	SDK := mfsdk.NewSDK(SDKCfg)

	svc := provision.New(cfg, SDK, logger)
	svc = api.NewLoggingMiddleware(svc, logger)

	httpServerConfig := server.Config{Host: "", Port: cfg.Server.HTTPPort, KeyFile: cfg.Server.ServerKey, CertFile: cfg.Server.ServerCert}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

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
	cfg := provision.Config{}
	if err := env.Parse(&cfg); err != nil {
		return provision.Config{}, err
	}

	if cfg.Bootstrap.AutoWhiteList && !cfg.Bootstrap.Provision {
		return provision.Config{}, errors.New("Can't auto whitelist if auto config save is off")
	}

	var content map[string]interface{}
	if cfg.BSContent != "" {
		if err := json.Unmarshal([]byte(cfg.BSContent), &content); err != nil {
			return provision.Config{}, errFailedToReadBootstrapContent
		}
	}

	cfg = provision.Config{
		Bootstrap: provision.Bootstrap{
			Content: content,
		},
		// This is default conf for provision if there is no config file
		Channels: []mfgroups.Group{
			{
				Name:     "control-channel",
				Metadata: map[string]interface{}{"type": "control"},
			}, {
				Name:     "data-channel",
				Metadata: map[string]interface{}{"type": "data"},
			},
		},
		Things: []mfclients.Client{
			{
				Name:     "thing",
				Metadata: map[string]interface{}{"external_id": "xxxxxx"},
			},
		},
	}

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
