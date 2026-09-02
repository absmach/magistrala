// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package writers

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/pkg/messaging"
	broker "github.com/absmach/magistrala/pkg/messaging/fluxmq"
)

const (
	AllTopic = "writers/#"

	prefix = "writers"
)

// InternalMetadata returns an option for a trusted local-service connection
// that carries internal metadata over mTLS and consumes the broker-provisioned
// writers stream.
func InternalMetadata(certFile, keyFile, caFile string) messaging.Option {
	return broker.InternalMetadata(certFile, keyFile, caFile)
}

func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	brokerOpts := []messaging.Option{
		broker.Prefix(prefix),
		broker.ConnectionName("writers-msg-pubsub"),
	}
	brokerOpts = append(brokerOpts, opts...)
	pb, err := broker.NewPubSub(ctx, url, logger, brokerOpts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}

// NewPublisher creates the publisher that feeds the writers stream. Pass
// InternalMetadata so it connects as a trusted local principal: the broker
// stamps its own transport protocol and identity on a publication from an
// untrusted connection, which would leave every stored message recorded as
// having arrived over AMQP instead of the protocol its device spoke.
func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	brokerOpts := []messaging.Option{
		broker.Prefix(prefix),
		broker.ConnectionName("writers-msg-pub"),
	}
	brokerOpts = append(brokerOpts, opts...)
	pb, err := broker.NewPublisher(ctx, url, brokerOpts...)
	if err != nil {
		return nil, err
	}

	return pb, nil
}
