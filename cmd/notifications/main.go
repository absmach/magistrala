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
	"github.com/absmach/supermq"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/notifications/emailer"
	"github.com/absmach/supermq/notifications/events"
	"github.com/absmach/supermq/notifications/middleware"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "notifications"
	envPrefixUsers = "SMQ_USERS_GRPC_"
	defEmailPort   = "25"
)

type config struct {
	LogLevel           string  `env:"SMQ_NOTIFICATIONS_LOG_LEVEL"              envDefault:"info"`
	ESURL              string  `env:"SMQ_ES_URL"                               envDefault:"amqp://guest:guest@localhost:5682/"`
	JaegerURL          url.URL `env:"SMQ_JAEGER_URL"                           envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry      bool    `env:"SMQ_SEND_TELEMETRY"                       envDefault:"true"`
	InstanceID         string  `env:"SMQ_NOTIFICATIONS_INSTANCE_ID"            envDefault:""`
	DomainAltName      string  `env:"SMQ_NOTIFICATIONS_DOMAIN_ALT_NAME"        envDefault:"domain"`
	TraceRatio         float64 `env:"SMQ_JAEGER_TRACE_RATIO"                   envDefault:"1.0"`
	EmailHost          string  `env:"SMQ_EMAIL_HOST"                           envDefault:"localhost"`
	EmailPort          string  `env:"SMQ_EMAIL_PORT"                           envDefault:"25"`
	EmailUsername      string  `env:"SMQ_EMAIL_USERNAME"                       envDefault:""`
	EmailPassword      string  `env:"SMQ_EMAIL_PASSWORD"                       envDefault:""`
	EmailFromAddress   string  `env:"SMQ_EMAIL_FROM_ADDRESS"                   envDefault:"noreply@supermq.com"`
	EmailFromName      string  `env:"SMQ_EMAIL_FROM_NAME"                      envDefault:"SuperMQ Notifications"`
	InvitationTemplate string  `env:"SMQ_EMAIL_INVITATION_TEMPLATE"            envDefault:"docker/templates/invitation-sent-email.tmpl"`
	AcceptanceTemplate string  `env:"SMQ_EMAIL_ACCEPTANCE_TEMPLATE"            envDefault:"docker/templates/invitation-accepted-email.tmpl"`
	RejectionTemplate  string  `env:"SMQ_EMAIL_REJECTION_TEMPLATE"             envDefault:"docker/templates/invitation-rejected-email.tmpl"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := smqlog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer smqlog.ExitWithError(&exitCode)

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

	subscriber, err := store.NewSubscriber(ctx, cfg.ESURL, logger)
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
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}
