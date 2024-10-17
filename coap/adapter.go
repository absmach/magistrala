// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Magistrala CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/policies"
)

const chansPrefix = "channels"

// Service specifies CoAP service API.
type Service interface {
	// Publish publishes message to specified channel.
	// Key is used to authorize publisher.
	Publish(ctx context.Context, key string, msg *messaging.Message) error

	// Subscribes to channel with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, chanID, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,.
type adapterService struct {
	things magistrala.ThingsServiceClient
	pubsub messaging.PubSub
}

// New instantiates the CoAP adapter implementation.
func New(thingsClient magistrala.ThingsServiceClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		things: thingsClient,
		pubsub: pubsub,
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	ar := &magistrala.ThingsAuthzReq{
		Permission: policies.PublishPermission,
		ThingKey:   key,
		ChannelID:  msg.GetChannel(),
	}
	res, err := svc.things.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}
	msg.Publisher = res.GetId()

	return svc.pubsub.Publish(ctx, msg.GetChannel(), msg)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error {
	ar := &magistrala.ThingsAuthzReq{
		Permission: policies.SubscribePermission,
		ThingKey:   key,
		ChannelID:  chanID,
	}
	res, err := svc.things.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	subCfg := messaging.SubscriberConfig{
		ID:      c.Token(),
		Topic:   subject,
		Handler: c,
	}
	return svc.pubsub.Subscribe(ctx, subCfg)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, chanID, subtopic, token string) error {
	ar := &magistrala.ThingsAuthzReq{
		Permission: policies.SubscribePermission,
		ThingKey:   key,
		ChannelID:  chanID,
	}
	res, err := svc.things.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	return svc.pubsub.Unsubscribe(ctx, token, subject)
}
