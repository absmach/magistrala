// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package brokers

import (
	"context"
	"log"
	"log/slog"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/fluxmq"
)

// SubjectAllMessages represents subject to subscribe for all the messages.
const SubjectAllMessages = string(messaging.MsgTopicPrefix) + "/#"

func init() {
	log.Println("The binary was built using FluxMQ as the message broker")
}

// ConnectionName returns an option that sets a human-readable connection name
// for identifying this client in the FluxMQ admin UI.
func ConnectionName(name string) messaging.Option {
	return fluxmq.ConnectionName(name)
}

// InternalMetadata returns an option for a trusted local-service connection
// that carries internal metadata over mTLS and consumes the broker-provisioned
// message stream.
func InternalMetadata(certFile, keyFile, caFile string) messaging.Option {
	return fluxmq.InternalMetadata(certFile, keyFile, caFile)
}

func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pb, err := fluxmq.NewPublisher(ctx, url, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	opts = append(opts, fluxmq.DirectTopicIngress())
	pb, err := fluxmq.NewPubSub(ctx, url, logger, opts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
