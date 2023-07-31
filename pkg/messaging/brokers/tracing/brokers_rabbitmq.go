//go:build rabbitmq
// +build rabbitmq

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package brokers

import (
	"log"

	"github.com/mainflux/mainflux/internal/server"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/rabbitmq/tracing"
	"go.opentelemetry.io/otel/trace"
)

// SubjectAllChannels represents subject to subscribe for all the channels.
const SubjectAllChannels = "channels.#"

func init() {
	log.Println("The binary was build using RabbitMQ as the message broker")
}

func NewPublisher(cfg server.Config, tracer trace.Tracer, pub messaging.Publisher) messaging.Publisher {

	return tracing.NewPublisher(cfg, tracer, pub)
}

func NewPubSub(cfg server.Config, tracer trace.Tracer, pubsub messaging.PubSub) messaging.PubSub {

	return tracing.NewPubSub(cfg, tracer, pubsub)
}
