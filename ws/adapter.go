// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
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
	// Publish publishes the message to the internal message broker.
	// ThingKey is used for authorization.
	// If the message is published successfully, nil is returned otherwise
	// error is returned.
	Publish(ctx context.Context, thingKey string, msg *messaging.Message) error

	// Subscribe subscribes message from the broker using the thingKey for authorization,
	// and the channelID for subscription. Subtopic is optional.
	// If the subscription is successful, nil is returned otherwise error is returned.
	Subscribe(ctx context.Context, thingKey, chanID, subtopic string, client *Client) error

	// Unsubscribe unsubscribes message from the broker using the thingKey for authorization,
	// and the channelID for subscription. Subtopic is optional.
	// If the unsubscription is successful, nil is returned otherwise error is returned.
	Unsubscribe(ctx context.Context, thingKey, chanID, subtopic string) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	auth   mainflux.AuthzServiceClient
	pubsub messaging.PubSub
}

// New instantiates the WS adapter implementation.
func New(auth mainflux.AuthzServiceClient, pubsub messaging.PubSub) Service {
	return &adapterService{
		auth:   auth,
		pubsub: pubsub,
	}
}

func (svc *adapterService) Publish(ctx context.Context, thingKey string, msg *messaging.Message) error {
	thid, err := svc.authorize(ctx, thingKey, msg.GetChannel(), "publish")
	if err != nil {
		return ErrUnauthorizedAccess
	}

	if len(msg.Payload) == 0 {
		return ErrFailedMessagePublish
	}

	msg.Publisher = thid

	if err := svc.pubsub.Publish(ctx, msg.GetChannel(), msg); err != nil {
		return ErrFailedMessagePublish
	}

	return nil
}

func (svc *adapterService) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *Client) error {
	if chanID == "" || thingKey == "" {
		return ErrUnauthorizedAccess
	}

	thingID, err := svc.authorize(ctx, thingKey, chanID, "subscribe")
	if err != nil {
		return ErrUnauthorizedAccess
	}

	c.id = thingID

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	if err := svc.pubsub.Subscribe(ctx, thingID, subject, c); err != nil {
		return errors.Wrap(ErrFailedSubscription, err)
	}

	return nil
}

func (svc *adapterService) Unsubscribe(ctx context.Context, thingKey, chanID, subtopic string) error {
	if chanID == "" || thingKey == "" {
		return ErrUnauthorizedAccess
	}

	thid, err := svc.authorize(ctx, thingKey, chanID, "subscribe")
	if err != nil {
		return ErrUnauthorizedAccess
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	return svc.pubsub.Unsubscribe(ctx, thid, subject)
}

// authorize checks if the thingKey is authorized to access the channel
// and returns the thingID if it is.
func (svc *adapterService) authorize(ctx context.Context, thingKey, chanID, action string) (string, error) {
	ar := &mainflux.AuthorizeReq{
		Namespace:   "",
		SubjectType: "thing",
		Permission:  action,
		Subject:     thingKey,
		Object:      chanID,
		ObjectType:  "group",
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
