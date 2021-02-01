// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
)

var (
	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrCreateID indicates error in creating id for entity creation
	ErrCreateID = errors.New("failed to create id")

	// ErrConflict indicates usage of the existing subscription.
	ErrConflict = errors.New("subscription already exist")

	// ErrSave indicates error saving entity.
	ErrSave = errors.New("failed to subscription")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrSelectEntity indicates problem with scanning data from db.
	ErrSelectEntity = errors.New("failed to select entity")

	// ErrRemoveEntity indicates error in removing entity
	ErrRemoveEntity = errors.New("remove entity failed")

	// ErrMessage indicates an error converting a message to Mainflux message.
	ErrMessage = errors.New("failed to convert to Mainflux message")
)

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

	consumers.Consumer
}

var _ Service = (*notifierService)(nil)

type notifierService struct {
	auth     mainflux.AuthServiceClient
	subs     SubscriptionsRepository
	idp      mainflux.IDProvider
	notifier Notifier
}

// New instantiates the subscriptions service implementation.
func New(auth mainflux.AuthServiceClient, subs SubscriptionsRepository, idp mainflux.IDProvider, notifier Notifier) Service {
	return &notifierService{
		auth:     auth,
		subs:     subs,
		idp:      idp,
		notifier: notifier,
	}
}

func (ns *notifierService) CreateSubscription(ctx context.Context, token string, sub Subscription) (string, error) {
	res, err := ns.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", errors.Wrap(ErrUnauthorizedAccess, err)
	}
	sub.ID, err = ns.idp.ID()
	if err != nil {
		return "", errors.Wrap(ErrCreateID, err)
	}

	sub.OwnerID = res.GetId()
	return ns.subs.Save(ctx, sub)
}

func (ns *notifierService) ViewSubscription(ctx context.Context, token, id string) (Subscription, error) {
	if _, err := ns.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Subscription{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ns.subs.Retrieve(ctx, id)
}

func (ns *notifierService) ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	if _, err := ns.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Page{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ns.subs.RetrieveAll(ctx, pm)
}

func (ns *notifierService) RemoveSubscription(ctx context.Context, token, id string) error {
	if _, err := ns.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return ns.subs.Remove(ctx, id)
}

func (ns *notifierService) Consume(message interface{}) error {
	msg, ok := message.(messaging.Message)
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
	page, err := ns.subs.RetrieveAll(context.Background(), pm)
	if err != nil {
		return err
	}

	var to []string
	for _, sub := range page.Subscriptions {
		to = append(to, sub.Contact)
	}
	if len(to) > 0 {
		err := ns.notifier.Notify("", to, msg)
		if err != nil {
			return errors.Wrap(ErrNotify, err)
		}
	}

	return nil
}
