// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

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

var (
	// errFailedMessagePublish indicates that message publishing failed.
	errFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// errFailedUnsubscribe indicates that client couldn't unsubscribe from specified channel.
	errFailedUnsubscribe = errors.New("failed to unsubscribe from a channel")

	// ErrEmptyTopic indicate absence of thingKey in the request.
	ErrEmptyTopic = errors.New("empty topic")
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
	things magistrala.ThingsServiceClient
	pubsub messaging.PubSub
}

// New instantiates the WS adapter implementation.
func New(thingsClient magistrala.ThingsServiceClient, pubsub messaging.PubSub) Service {
	return &adapterService{
		things: thingsClient,
		pubsub: pubsub,
	}
}

func (svc *adapterService) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *Client) error {
	if chanID == "" || thingKey == "" {
		return svcerr.ErrAuthentication
	}

	thingID, err := svc.authorize(ctx, thingKey, chanID, policies.SubscribePermission)
	if err != nil {
		return svcerr.ErrAuthorization
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
	ar := &magistrala.ThingsAuthzReq{
		Permission: action,
		ThingKey:   thingKey,
		ChannelID:  chanID,
	}
	res, err := svc.things.Authorize(ctx, ar)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return res.GetId(), nil
}
