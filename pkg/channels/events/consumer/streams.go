// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	rconsumer "github.com/absmach/supermq/pkg/roles/rolemanager/events/consumer"
)

const (
	stream = "events.supermq.channel.*"

	create            = "channel.create"
	update            = "channel.update"
	updateTags        = "channel.update_tags"
	enable            = "channel.enable"
	disable           = "channel.disable"
	remove            = "channel.remove"
	connect           = "channel.connect"
	disconnect        = "channel.disconnect"
	setParentGroup    = "channel.set_parent"
	removeParentGroup = "channel.remove_parent"
)

var (
	errNoOperationKey           = errors.New("operation key is not found in event message")
	errCreateChannelEvent       = errors.New("failed to consume channel create event")
	errUpdateChannelEvent       = errors.New("failed to consume channel update event")
	errChangeStatusChannelEvent = errors.New("failed to consume channel change status event")
	errRemoveChannelEvent       = errors.New("failed to consume channel remove event")
	errConnectEvent             = errors.New("failed to consume channel connect event")
	errDisconnectEvent          = errors.New("failed to consume channel disconnect event")
	errSetParentGroupEvent      = errors.New("failed to consume channel add parent group event")
	errRemoveParentGroupEvent   = errors.New("failed to consume channel remove parent group event")
)

type eventHandler struct {
	repo              channels.Repository
	rolesEventHandler rconsumer.EventHandler
}

func ChannelsEventsSubscribe(ctx context.Context, repo channels.Repository, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, "channels-es-sub", logger)
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
func NewEventHandler(repo channels.Repository) events.EventHandler {
	reh := rconsumer.NewEventHandler("channel", repo)
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
		return es.createChannelHandler(ctx, msg)
	case update:
		return es.updateChannelHandler(ctx, msg)
	case updateTags:
		return es.updateChannelTagsHandler(ctx, msg)
	case enable, disable:
		return es.changeStatusChannelHandler(ctx, msg)
	case remove:
		return es.removeChannelHandler(ctx, msg)
	case connect:
		return es.connectChannelHandler(ctx, msg)
	case disconnect:
		return es.disconnectChannelHandler(ctx, msg)
	case setParentGroup:
		return es.setParentGroupHandler(ctx, msg)
	case removeParentGroup:
		return es.removeParentGroupHandler(ctx, msg)
	}

	return es.rolesEventHandler.Handle(ctx, op, msg)
}

func (es *eventHandler) createChannelHandler(ctx context.Context, data map[string]any) error {
	c, rps, err := decodeCreateChannelEvent(data)
	if err != nil {
		return errors.Wrap(errCreateChannelEvent, err)
	}

	if _, err := es.repo.Save(ctx, c); err != nil {
		return errors.Wrap(errCreateChannelEvent, err)
	}
	if _, err := es.repo.AddRoles(ctx, rps); err != nil {
		return errors.Wrap(errCreateChannelEvent, err)
	}

	return nil
}

func (es *eventHandler) updateChannelHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeUpdateChannelEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateChannelEvent, err)
	}

	if _, err := es.repo.Update(ctx, c); err != nil {
		return errors.Wrap(errUpdateChannelEvent, err)
	}

	return nil
}

func (es *eventHandler) updateChannelTagsHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeUpdateChannelEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateChannelEvent, err)
	}

	if _, err := es.repo.UpdateTags(ctx, c); err != nil {
		return errors.Wrap(errUpdateChannelEvent, err)
	}

	return nil
}

func (es *eventHandler) changeStatusChannelHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeChangeStatusChannelEvent(data)
	if err != nil {
		return errors.Wrap(errChangeStatusChannelEvent, err)
	}

	if _, err := es.repo.ChangeStatus(ctx, c); err != nil {
		return errors.Wrap(errChangeStatusChannelEvent, err)
	}

	return nil
}

func (es *eventHandler) removeChannelHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeRemoveChannelEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveChannelEvent, err)
	}

	if err := es.repo.Remove(ctx, c.ID); err != nil {
		return errors.Wrap(errRemoveChannelEvent, err)
	}
	return nil
}

func (es *eventHandler) connectChannelHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeConnectEvent(data)
	if err != nil {
		return errors.Wrap(errConnectEvent, err)
	}
	if err := es.repo.AddConnections(ctx, c); err != nil {
		return errors.Wrap(errConnectEvent, err)
	}
	return nil
}

func (es *eventHandler) disconnectChannelHandler(ctx context.Context, data map[string]any) error {
	c, err := decodeDisconnectEvent(data)
	if err != nil {
		return errors.Wrap(errDisconnectEvent, err)
	}
	if err := es.repo.RemoveConnections(ctx, c); err != nil {
		return errors.Wrap(errDisconnectEvent, err)
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
