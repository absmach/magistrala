// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
)

const chansPrefix = "channels"

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedUnsubscribe indicates that client couldn't unsubscribe from specified channel.
	ErrFailedUnsubscribe = errors.New("failed to unsubscribe from a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")

	// ErrInvalidConnection indicates that client couldn't subscribe to message broker.
	ErrInvalidConnection = errors.New("nats: invalid connection")

	// ErrUnauthorizedAccess indicates that client provided missing or invalid credentials.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrEmptyTopic indicate absence of thingKey in the request.
	ErrEmptyTopic = errors.New("empty topic")

	// ErrEmptyID indicate absence of channelID in the request.
	ErrEmptyID = errors.New("empty id")
)

// Service specifies web socket service API.
type Service interface {
	// Subscribe subscribes message from the broker using the thingKey for authorization,
	// and the channelID for subscription. Subtopic is optional.
	// If the subscription is successful, nil is returned otherwise error is returned.
	Subscribe(ctx context.Context, thingKey, chanID, subtopic string, client *Client) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	auth   magistrala.AuthzServiceClient
	pubsub messaging.PubSub
}

// New instantiates the WS adapter implementation.
func New(authClient magistrala.AuthzServiceClient, pubsub messaging.PubSub) Service {
	return &adapterService{
		auth:   authClient,
		pubsub: pubsub,
	}
}

func (svc *adapterService) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *Client) error {
	if chanID == "" || thingKey == "" {
		return ErrUnauthorizedAccess
	}

	thingID, err := svc.authorize(ctx, thingKey, chanID, auth.SubscribePermission)
	if err != nil {
		return ErrUnauthorizedAccess
	}

	c.id = thingID

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	subCfg := messaging.SubscriberConfig{
		ID:      thingID,
		Topic:   subject,
		Handler: c,
	}
	if err := svc.pubsub.Subscribe(ctx, subCfg); err != nil {
		return ErrFailedSubscription
	}

	return nil
}

// authorize checks if the thingKey is authorized to access the channel
// and returns the thingID if it is.
func (svc *adapterService) authorize(ctx context.Context, thingKey, chanID, action string) (string, error) {
	ar := &magistrala.AuthorizeReq{
		SubjectType: auth.ThingType,
		Permission:  action,
		Subject:     thingKey,
		Object:      chanID,
		ObjectType:  auth.GroupType,
	}
	res, err := svc.auth.Authorize(ctx, ar)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return res.GetId(), nil
}
