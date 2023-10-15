// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
)

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, token string, msg *messaging.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	auth      mainflux.AuthzServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, auth mainflux.AuthzServiceClient) Service {
	return &adapterService{
		publisher: publisher,
		auth:      auth,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg *messaging.Message) error {
	ar := &mainflux.AuthorizeReq{
		Namespace:   "",
		SubjectType: "thing",
		Permission:  "publish",
		Subject:     token,
		Object:      msg.Channel,
		ObjectType:  "group",
	}

	res, err := as.auth.Authorize(ctx, ar)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	msg.Publisher = res.GetId()

	return as.publisher.Publish(ctx, msg.Channel, msg)
}
