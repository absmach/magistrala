//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/mainflux/mainflux"
)

var _ mainflux.MessagePublisher = (*adapterService)(nil)

type adapterService struct {
	pub    mainflux.MessagePublisher
	things mainflux.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(pub mainflux.MessagePublisher, things mainflux.ThingsServiceClient) mainflux.MessagePublisher {
	return &adapterService{
		pub:    pub,
		things: things,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg mainflux.RawMessage) error {
	ar := &mainflux.AccessReq{
		Token:  token,
		ChanID: msg.GetChannel(),
	}
	thid, err := as.things.CanAccess(ctx, ar)
	if err != nil {
		return err
	}
	msg.Publisher = thid.GetValue()

	return as.pub.Publish(ctx, token, msg)
}
