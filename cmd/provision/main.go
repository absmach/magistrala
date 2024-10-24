// Copyright (c) Abstract Machines
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

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/clients"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/provision"
	"github.com/absmach/magistrala/provision/api"
	"github.com/caarlos0/env/v11"
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
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mglog.New(os.Stdout, cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

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

	SDKCfg := mgsdk.Config{
		UsersURL:        cfg.Server.UsersURL,
		ClientsURL:      cfg.Server.ClientsURL,
		BootstrapURL:    cfg.Server.MgBSURL,
		CertsURL:        cfg.Server.MgCertsURL,
		MsgContentType:  contentType,
		TLSVerification: cfg.Server.TLS,
	}
	SDK := mgsdk.NewSDK(SDKCfg)

	svc := provision.New(cfg, SDK, logger)
	svc = api.NewLoggingMiddleware(svc, logger)

	httpServerConfig := server.Config{Host: "", Port: cfg.Server.HTTPPort, KeyFile: cfg.Server.ServerKey, CertFile: cfg.Server.ServerCert}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
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

	cfg.Bootstrap.Content = content
	// This is default conf for provision if there is no config file
	cfg.Channels = []channels.Channel{
		{
			Name:     "control-channel",
			Metadata: map[string]interface{}{"type": "control"},
		}, {
			Name:     "data-channel",
			Metadata: map[string]interface{}{"type": "data"},
		},
	}
	cfg.Clients = []clients.Client{
		{
			Name:     "client",
			Metadata: map[string]interface{}{"external_id": "xxxxxx"},
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
