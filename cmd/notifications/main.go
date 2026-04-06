// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains notifications main function to start the notifications service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/notifications/emailer"
	"github.com/absmach/magistrala/notifications/events"
	"github.com/absmach/magistrala/notifications/middleware"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "notifications"
	envPrefixUsers = "MG_USERS_GRPC_"
	defEmailPort   = "25"
)

type config struct {
	LogLevel           string  `env:"MG_NOTIFICATIONS_LOG_LEVEL"              envDefault:"info"`
	ESURL              string  `env:"MG_ES_URL"                               envDefault:"amqp://guest:guest@localhost:5682/"`
	JaegerURL          url.URL `env:"MG_JAEGER_URL"                           envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry      bool    `env:"MG_SEND_TELEMETRY"                       envDefault:"true"`
	InstanceID         string  `env:"MG_NOTIFICATIONS_INSTANCE_ID"            envDefault:""`
	DomainAltName      string  `env:"MG_NOTIFICATIONS_DOMAIN_ALT_NAME"        envDefault:"domain"`
	TraceRatio         float64 `env:"MG_JAEGER_TRACE_RATIO"                   envDefault:"1.0"`
	EmailHost          string  `env:"MG_EMAIL_HOST"                           envDefault:"localhost"`
	EmailPort          string  `env:"MG_EMAIL_PORT"                           envDefault:"25"`
	EmailUsername      string  `env:"MG_EMAIL_USERNAME"                       envDefault:""`
	EmailPassword      string  `env:"MG_EMAIL_PASSWORD"                       envDefault:""`
	EmailFromAddress   string  `env:"MG_EMAIL_FROM_ADDRESS"                   envDefault:"noreply@magistrala.com"`
	EmailFromName      string  `env:"MG_EMAIL_FROM_NAME"                      envDefault:"Magistrala Notifications"`
	InvitationTemplate string  `env:"MG_EMAIL_INVITATION_TEMPLATE"            envDefault:"docker/templates/invitation-sent-email.tmpl"`
	AcceptanceTemplate string  `env:"MG_EMAIL_ACCEPTANCE_TEMPLATE"            envDefault:"docker/templates/invitation-accepted-email.tmpl"`
	RejectionTemplate  string  `env:"MG_EMAIL_REJECTION_TEMPLATE"             envDefault:"docker/templates/invitation-rejected-email.tmpl"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
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

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %s", err))
		}
	}()

	usersClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&usersClientCfg, env.Options{Prefix: envPrefixUsers}); err != nil {
		logger.Error(fmt.Sprintf("failed to load users gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	usersClient, usersHandler, err := grpcclient.SetupUsersClient(ctx, usersClientCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup users gRPC client: %s", err))
		exitCode = 1
		return
	}
	defer usersHandler.Close()
	logger.Info("Successfully connected to users gRPC server " + usersHandler.Secure())

	emailerCfg := emailer.Config{
		FromAddress:        cfg.EmailFromAddress,
		FromName:           cfg.EmailFromName,
		DomainAltName:      cfg.DomainAltName,
		InvitationTemplate: cfg.InvitationTemplate,
		AcceptanceTemplate: cfg.AcceptanceTemplate,
		RejectionTemplate:  cfg.RejectionTemplate,
		EmailHost:          cfg.EmailHost,
		EmailPort:          cfg.EmailPort,
		EmailUsername:      cfg.EmailUsername,
		EmailPassword:      cfg.EmailPassword,
	}

	notifier, err := emailer.New(usersClient, emailerCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create emailer: %s", err))
		exitCode = 1
		return
	}

	// Wrap notifier with middleware
	notifier = middleware.NewLogging(notifier, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "notifier")
	notifier = middleware.NewMetrics(notifier, counter, latency)
	notifier = middleware.NewTracing(notifier, tp.Tracer(svcName))

	subscriber, err := store.NewSubscriber(ctx, cfg.ESURL, "notifications-es-sub", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create subscriber: %s", err))
		exitCode = 1
		return
	}

	logger.Info("Subscribed to Event Store")

	if err := events.Start(ctx, svcName, subscriber, notifier); err != nil {
		logger.Error(fmt.Sprintf("failed to start %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}
