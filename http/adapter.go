// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/broker"
)

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, token string, msg broker.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	broker broker.Nats
	things mainflux.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(broker broker.Nats, things mainflux.ThingsServiceClient) Service {
	return &adapterService{
		broker: broker,
		things: things,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg broker.Message) error {
	ar := &mainflux.AccessByKeyReq{
		Token:  token,
		ChanID: msg.GetChannel(),
	}
	thid, err := as.things.CanAccessByKey(ctx, ar)
	if err != nil {
		return err
	}
	msg.Publisher = thid.GetValue()

	return as.broker.Publish(ctx, token, msg)
}
