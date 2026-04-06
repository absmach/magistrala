// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !msg_fluxmq && !msg_rabbitmq && !rabbitmq
// +build !msg_fluxmq,!msg_rabbitmq,!rabbitmq

package brokers

import (
	"log"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/nats/tracing"
	"github.com/absmach/magistrala/pkg/server"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	log.Println("The binary was built using NATS as the message broker")
}

func NewPublisher(cfg server.Config, tracer trace.Tracer, publisher messaging.Publisher) messaging.Publisher {
	return tracing.NewPublisher(cfg, tracer, publisher)
}

func NewPubSub(cfg server.Config, tracer trace.Tracer, pubsub messaging.PubSub) messaging.PubSub {
	return tracing.NewPubSub(cfg, tracer, pubsub)
}
