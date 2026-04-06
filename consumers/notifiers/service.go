// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/consumers"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
)

var (
	// ErrMessage indicates an error converting a message to Magistrala message.
	ErrMessage = errors.New("failed to convert to Magistrala message")

	// ErrSubscriptionsAlreadyExists indicates subscription already exists.
	ErrSubscriptionsAlreadyExists = errors.NewRequestError("subscription already exists")
)
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
	authn    smqauthn.Authentication
	subs     SubscriptionsRepository
	idp      magistrala.IDProvider
	notifier consumers.Notifier
	errCh    chan error
	from     string
}

// New instantiates the subscriptions service implementation.
func New(authn smqauthn.Authentication, subs SubscriptionsRepository, idp magistrala.IDProvider, notifier consumers.Notifier, from string) Service {
	return &notifierService{
		authn:    authn,
		subs:     subs,
		idp:      idp,
		notifier: notifier,
		errCh:    make(chan error, 1),
		from:     from,
	}
}

func (ns *notifierService) CreateSubscription(ctx context.Context, token string, sub Subscription) (string, error) {
	session, err := ns.authn.Authenticate(ctx, token)
	if err != nil {
		return "", err
	}
	sub.ID, err = ns.idp.ID()
	if err != nil {
		return "", err
	}

	sub.OwnerID = session.DomainUserID
	id, err := ns.subs.Save(ctx, sub)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return id, nil
}

func (ns *notifierService) ViewSubscription(ctx context.Context, token, id string) (Subscription, error) {
	if _, err := ns.authn.Authenticate(ctx, token); err != nil {
		return Subscription{}, err
	}

	return ns.subs.Retrieve(ctx, id)
}

func (ns *notifierService) ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	if _, err := ns.authn.Authenticate(ctx, token); err != nil {
		return Page{}, err
	}

	return ns.subs.RetrieveAll(ctx, pm)
}

func (ns *notifierService) RemoveSubscription(ctx context.Context, token, id string) error {
	if _, err := ns.authn.Authenticate(ctx, token); err != nil {
		return err
	}

	return ns.subs.Remove(ctx, id)
}

func (ns *notifierService) ConsumeBlocking(ctx context.Context, message any) error {
	msg, ok := message.(*messaging.Message)
	if !ok {
		return ErrMessage
	}
	to, err := ns.recipients(ctx, msg)
	if err != nil {
		return err
	}

	if len(to) > 0 {
		err := ns.notifier.Notify(ns.from, to, msg)
		if err != nil {
			return errors.Wrap(consumers.ErrNotify, err)
		}
	}

	return nil
}

func (ns *notifierService) ConsumeAsync(ctx context.Context, message any) {
	msg, ok := message.(*messaging.Message)
	if !ok {
		ns.errCh <- ErrMessage
		return
	}
	to, err := ns.recipients(ctx, msg)
	if err != nil {
		ns.errCh <- err
		return
	}

	if len(to) > 0 {
		if err := ns.notifier.Notify(ns.from, to, msg); err != nil {
			ns.errCh <- errors.Wrap(consumers.ErrNotify, err)
		}
	}
}

func (ns *notifierService) Errors() <-chan error {
	return ns.errCh
}

func (ns *notifierService) recipients(ctx context.Context, msg *messaging.Message) ([]string, error) {
	topic, ok := subscriptionTopic(msg)
	if !ok {
		return nil, nil
	}

	pm := PageMetadata{
		Topic:  topic,
		Offset: 0,
		Limit:  -1,
	}
	page, err := ns.subs.RetrieveAll(ctx, pm)
	if err != nil {
		if errors.Contains(err, repoerr.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	to := make([]string, 0, len(page.Subscriptions))
	for _, sub := range page.Subscriptions {
		to = append(to, sub.Contact)
	}

	return to, nil
}

func subscriptionTopic(msg *messaging.Message) (string, bool) {
	channel := msg.GetChannel()
	if channel == "" {
		return "", false
	}

	subtopic := msg.GetSubtopic()
	if subtopic == "" {
		return channel, true
	}

	return channel + "/" + subtopic, true
}
