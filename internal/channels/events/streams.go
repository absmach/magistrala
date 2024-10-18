// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/channels"
	entityRolesEvents "github.com/absmach/magistrala/pkg/entityroles/events"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
)

const streamID = "magistrala.things"

var _ channels.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc channels.Service
	entityRolesEvents.RolesSvcEventStoreMiddleware
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc channels.Service, url string) (channels.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	rolesSvcEventStoreMiddleware := entityRolesEvents.NewRolesSvcEventStoreMiddleware("domains", svc, publisher)
	return &eventStore{
		svc:                          svc,
		Publisher:                    publisher,
		RolesSvcEventStoreMiddleware: rolesSvcEventStoreMiddleware,
	}, nil
}

func (es *eventStore) CreateChannels(ctx context.Context, token string, chs ...channels.Channel) ([]channels.Channel, error) {
	chs, err := es.svc.CreateChannels(ctx, token, chs...)
	if err != nil {
		return chs, err
	}

	for _, ch := range chs {
		event := createChannelEvent{
			ch,
		}
		if err := es.Publish(ctx, event); err != nil {
			return chs, err
		}
	}

	return chs, nil
}

func (es *eventStore) UpdateChannel(ctx context.Context, token string, ch channels.Channel) (channels.Channel, error) {
	chann, err := es.svc.UpdateChannel(ctx, token, ch)
	if err != nil {
		return chann, err
	}

	return es.update(ctx, "", chann)
}

func (es *eventStore) UpdateChannelTags(ctx context.Context, token string, ch channels.Channel) (channels.Channel, error) {
	chann, err := es.svc.UpdateChannelTags(ctx, token, ch)
	if err != nil {
		return chann, err
	}

	return es.update(ctx, "tags", chann)
}

func (es *eventStore) update(ctx context.Context, operation string, ch channels.Channel) (channels.Channel, error) {
	event := updateChannelEvent{
		ch, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return ch, err
	}

	return ch, nil
}

func (es *eventStore) ViewChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	chann, err := es.svc.ViewChannel(ctx, token, id)
	if err != nil {
		return chann, err
	}

	event := viewChannelEvent{
		chann,
	}
	if err := es.Publish(ctx, event); err != nil {
		return chann, err
	}

	return chann, nil
}

func (es *eventStore) ListChannels(ctx context.Context, token string, pm channels.PageMetadata) (channels.Page, error) {
	cp, err := es.svc.ListChannels(ctx, token, pm)
	if err != nil {
		return cp, err
	}
	event := listChannelEvent{
		pm,
	}
	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}
func (es *eventStore) ListChannelsByThing(ctx context.Context, token, thID string, pm channels.PageMetadata) (channels.Page, error) {
	cp, err := es.svc.ListChannelsByThing(ctx, token, thID, pm)
	if err != nil {
		return cp, err
	}
	event := listChannelByThingEvent{
		thID,
		pm,
	}
	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}
func (es *eventStore) EnableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	cli, err := es.svc.EnableChannel(ctx, token, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) DisableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	cli, err := es.svc.DisableChannel(ctx, token, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) changeStatus(ctx context.Context, ch channels.Channel) (channels.Channel, error) {
	event := changeStatusChannelEvent{
		id:        ch.ID,
		updatedAt: ch.UpdatedAt,
		updatedBy: ch.UpdatedBy,
		status:    ch.Status.String(),
	}
	if err := es.Publish(ctx, event); err != nil {
		return ch, err
	}

	return ch, nil
}

func (es *eventStore) RemoveChannel(ctx context.Context, token, id string) error {
	if err := es.svc.RemoveChannel(ctx, token, id); err != nil {
		return err
	}

	event := removeChannelEvent{id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	if err := es.svc.Connect(ctx, token, chIDs, thIDs); err != nil {
		return err
	}

	event := connectEvent{chIDs, thIDs}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	if err := es.svc.Disconnect(ctx, token, chIDs, thIDs); err != nil {
		return err
	}

	event := disconnectEvent{chIDs, thIDs}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
