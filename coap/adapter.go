// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Mainflux CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"fmt"
	"sync"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
)

const chansPrefix = "channels"

// ErrUnsubscribe indicates an error to unsubscribe.
var ErrUnsubscribe = errors.New("unable to unsubscribe")

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
	auth    mainflux.AuthzServiceClient
	pubsub  messaging.PubSub
	obsLock sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(auth mainflux.AuthzServiceClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		auth:    auth,
		pubsub:  pubsub,
		obsLock: sync.Mutex{},
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	ar := &mainflux.AuthorizeReq{
		Namespace:   "",
		SubjectType: "thing",
		Permission:  "publish",
		Subject:     key,
		Object:      msg.Channel,
		ObjectType:  "group",
	}
	res, err := svc.auth.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	msg.Publisher = res.GetId()

	return svc.pubsub.Publish(ctx, msg.Channel, msg)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error {
	ar := &mainflux.AuthorizeReq{
		Namespace:   "",
		SubjectType: "thing",
		Permission:  "subscribe",
		Subject:     key,
		Object:      chanID,
		ObjectType:  "group",
	}
	res, err := svc.auth.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	return svc.pubsub.Subscribe(ctx, c.Token(), subject, c)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, chanID, subtopic, token string) error {
	ar := &mainflux.AuthorizeReq{
		Namespace:   "",
		SubjectType: "thing",
		Permission:  "subscribe",
		Subject:     key,
		Object:      chanID,
		ObjectType:  "group",
	}
	res, err := svc.auth.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	return svc.pubsub.Unsubscribe(ctx, token, subject)
}
