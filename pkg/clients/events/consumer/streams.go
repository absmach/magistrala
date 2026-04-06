// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rconsumer "github.com/absmach/magistrala/pkg/roles/rolemanager/events/consumer"
)

const (
	stream = "events.magistrala.client.*"

	create            = "client.create"
	update            = "client.update"
	updateTags        = "client.update_tags"
	enable            = "client.enable"
	disable           = "client.disable"
	remove            = "client.remove"
	setParentGroup    = "client.set_parent"
	removeParentGroup = "client.remove_parent"
)

var (
	errNoOperationKey          = errors.New("operation key is not found in event message")
	errCreateClientEvent       = errors.New("failed to consume client create event")
	errUpdateClientEvent       = errors.New("failed to consume client update event")
	errChangeStatusClientEvent = errors.New("failed to consume client change status event")
	errRemoveClientEvent       = errors.New("failed to consume client remove event")
	errSetParentGroupEvent     = errors.New("failed to consume client add parent group event")
	errRemoveParentGroupEvent  = errors.New("failed to consume client remove parent group event")
)

type eventHandler struct {
	repo              clients.Repository
	rolesEventHandler rconsumer.EventHandler
}

func ClientsEventsSubscribe(ctx context.Context, repo clients.Repository, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, "clients-es-sub", logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:   stream,
		Consumer: esConsumerName,
		Handler:  NewEventHandler(repo),
		Ordered:  true,
	}
	return subscriber.Subscribe(ctx, subConfig)
}

// NewEventHandler returns new event store handler.
func NewEventHandler(repo clients.Repository) events.EventHandler {
	reh := rconsumer.NewEventHandler("client", repo)
	return &eventHandler{
		repo:              repo,
		rolesEventHandler: reh,
	}
}

func (es *eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	op, ok := msg["operation"]

	if !ok {
		return errNoOperationKey
	}
	switch op {
	case create:
		return es.createClientHandler(ctx, msg)
	case update:
		return es.updateClientHandler(ctx, msg)
	case updateTags:
		return es.updateClientTagsHandler(ctx, msg)
	case enable, disable:
		return es.changeStatusClientHandler(ctx, msg)
	case remove:
		return es.removeClientHandler(ctx, msg)
	case setParentGroup:
		return es.setParentGroupHandler(ctx, msg)
	case removeParentGroup:
		return es.removeParentGroupHandler(ctx, msg)
	}

	return es.rolesEventHandler.Handle(ctx, op, msg)
}

func (es *eventHandler) createClientHandler(ctx context.Context, data map[string]any) error {
	c, rps, err := decodeCreateClientEvent(data)
	if err != nil {
		return errors.Wrap(errCreateClientEvent, err)
	}

	if _, err := es.repo.Save(ctx, c); err != nil {
		return errors.Wrap(errCreateClientEvent, err)
	}
	if _, err := es.repo.AddRoles(ctx, rps); err != nil {
		return errors.Wrap(errCreateClientEvent, err)
	}

	return nil
}

func (es *eventHandler) updateClientHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeUpdateClientEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateClientEvent, err)
	}

	if _, err := es.repo.Update(ctx, c); err != nil {
		return errors.Wrap(errUpdateClientEvent, err)
	}

	return nil
}

func (es *eventHandler) updateClientTagsHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeUpdateClientEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateClientEvent, err)
	}

	if _, err := es.repo.UpdateTags(ctx, c); err != nil {
		return errors.Wrap(errUpdateClientEvent, err)
	}

	return nil
}

func (es *eventHandler) changeStatusClientHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeChangeStatusClientEvent(data)
	if err != nil {
		return errors.Wrap(errChangeStatusClientEvent, err)
	}

	if _, err := es.repo.ChangeStatus(ctx, c); err != nil {
		return errors.Wrap(errChangeStatusClientEvent, err)
	}

	return nil
}

func (es *eventHandler) removeClientHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeRemoveClientEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveClientEvent, err)
	}

	if err := es.repo.Delete(ctx, c.ID); err != nil {
		return errors.Wrap(errRemoveClientEvent, err)
	}
	return nil
}

func (es *eventHandler) setParentGroupHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeSetParentGroupEvent(data)
	if err != nil {
		return errors.Wrap(errSetParentGroupEvent, err)
	}
	if err := es.repo.SetParentGroup(ctx, c); err != nil {
		return errors.Wrap(errSetParentGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) removeParentGroupHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeRemoveParentGroupEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveParentGroupEvent, err)
	}
	if err := es.repo.RemoveParentGroup(ctx, c); err != nil {
		return errors.Wrap(errRemoveParentGroupEvent, err)
	}
	return nil
}
