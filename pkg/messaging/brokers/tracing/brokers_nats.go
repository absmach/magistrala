// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !rabbitmq
// +build !rabbitmq

package brokers

import (
	"log"

	"github.com/absmach/magistrala/internal/server"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/nats/tracing"
	"go.opentelemetry.io/otel/trace"
)

// SubjectAllChannels represents subject to subscribe for all the channels.
const SubjectAllChannels = "channels.>"

func init() {
	log.Println("The binary was build using Nats as the message broker")
}

func NewPublisher(cfg server.Config, tracer trace.Tracer, publisher messaging.Publisher) messaging.Publisher {
	return tracing.NewPublisher(cfg, tracer, publisher)
}

func NewPubSub(cfg server.Config, tracer trace.Tracer, pubsub messaging.PubSub) messaging.PubSub {
	return tracing.NewPubSub(cfg, tracer, pubsub)
}
