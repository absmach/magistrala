// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
)

// ErrMessage indicates an error converting a message to Magistrala message.
var ErrMessage = errors.New("failed to convert to Magistrala message")

var _ consumers.AsyncConsumer = (*notifierService)(nil)

// Service reprents a notification service.
type Service interface {
	// CreateSubscription persists a subscription.
	// Successful operation is indicated by non-nil error response.
	CreateSubscription(ctx context.Context, token string, sub Subscription) (string, error)

	// ViewSubscription retrieves the subscription for the given user and id.
	ViewSubscription(ctx context.Context, token, id string) (Subscription, error)

	// ListSubscriptions lists subscriptions having the provided user token and search params.
	ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error)

	// RemoveSubscription removes the subscription having the provided identifier.
	RemoveSubscription(ctx context.Context, token, id string) error

	consumers.BlockingConsumer
}

var _ Service = (*notifierService)(nil)

type notifierService struct {
	auth     magistrala.AuthServiceClient
	subs     SubscriptionsRepository
	idp      magistrala.IDProvider
	notifier Notifier
	errCh    chan error
	from     string
}

// New instantiates the subscriptions service implementation.
func New(auth magistrala.AuthServiceClient, subs SubscriptionsRepository, idp magistrala.IDProvider, notifier Notifier, from string) Service {
	return &notifierService{
		auth:     auth,
		subs:     subs,
		idp:      idp,
		notifier: notifier,
		errCh:    make(chan error, 1),
		from:     from,
	}
}

func (ns *notifierService) CreateSubscription(ctx context.Context, token string, sub Subscription) (string, error) {
	res, err := ns.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", err
	}
	sub.ID, err = ns.idp.ID()
	if err != nil {
		return "", err
	}

	sub.OwnerID = res.GetId()
	return ns.subs.Save(ctx, sub)
}

func (ns *notifierService) ViewSubscription(ctx context.Context, token, id string) (Subscription, error) {
	if _, err := ns.auth.Identify(ctx, &magistrala.IdentityReq{Token: token}); err != nil {
		return Subscription{}, err
	}

	return ns.subs.Retrieve(ctx, id)
}

func (ns *notifierService) ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	if _, err := ns.auth.Identify(ctx, &magistrala.IdentityReq{Token: token}); err != nil {
		return Page{}, err
	}

	return ns.subs.RetrieveAll(ctx, pm)
}

func (ns *notifierService) RemoveSubscription(ctx context.Context, token, id string) error {
	if _, err := ns.auth.Identify(ctx, &magistrala.IdentityReq{Token: token}); err != nil {
		return err
	}

	return ns.subs.Remove(ctx, id)
}

func (ns *notifierService) ConsumeBlocking(ctx context.Context, message interface{}) error {
	msg, ok := message.(*messaging.Message)
	if !ok {
		return ErrMessage
	}
	topic := msg.Channel
	if msg.Subtopic != "" {
		topic = fmt.Sprintf("%s.%s", msg.Channel, msg.Subtopic)
	}
	pm := PageMetadata{
		Topic:  topic,
		Offset: 0,
		Limit:  -1,
	}
	page, err := ns.subs.RetrieveAll(ctx, pm)
	if err != nil {
		return err
	}

	var to []string
	for _, sub := range page.Subscriptions {
		to = append(to, sub.Contact)
	}
	if len(to) > 0 {
		err := ns.notifier.Notify(ns.from, to, msg)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
	}

	return nil
}

func (ns *notifierService) ConsumeAsync(ctx context.Context, message interface{}) {
	msg, ok := message.(*messaging.Message)
	if !ok {
		ns.errCh <- ErrMessage
		return
	}
	topic := msg.Channel
	if msg.Subtopic != "" {
		topic = fmt.Sprintf("%s.%s", msg.Channel, msg.Subtopic)
	}
	pm := PageMetadata{
		Topic:  topic,
		Offset: 0,
		Limit:  -1,
	}
	page, err := ns.subs.RetrieveAll(ctx, pm)
	if err != nil {
		ns.errCh <- err
		return
	}

	var to []string
	for _, sub := range page.Subscriptions {
		to = append(to, sub.Contact)
	}
	if len(to) > 0 {
		if err := ns.notifier.Notify(ns.from, to, msg); err != nil {
			ns.errCh <- errors.Wrap(ErrNotify, err)
		}
	}
}

func (ns *notifierService) Errors() <-chan error {
	return ns.errCh
}
