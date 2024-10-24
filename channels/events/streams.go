// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rmEvents "github.com/absmach/magistrala/pkg/roles/rolemanager/events"
)

const streamID = "magistrala.clients"

var _ channels.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc channels.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around clients service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc channels.Service, url string) (channels.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	rolesSvcEventStoreMiddleware := rmEvents.NewRoleManagerEventStore("channels", svc, publisher)
	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: rolesSvcEventStoreMiddleware,
	}, nil
}

func (es *eventStore) CreateChannels(ctx context.Context, session authn.Session, chs ...channels.Channel) ([]channels.Channel, error) {
	chs, err := es.svc.CreateChannels(ctx, session, chs...)
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

func (es *eventStore) UpdateChannel(ctx context.Context, session authn.Session, ch channels.Channel) (channels.Channel, error) {
	chann, err := es.svc.UpdateChannel(ctx, session, ch)
	if err != nil {
		return chann, err
	}

	return es.update(ctx, "", chann)
}

func (es *eventStore) UpdateChannelTags(ctx context.Context, session authn.Session, ch channels.Channel) (channels.Channel, error) {
	chann, err := es.svc.UpdateChannelTags(ctx, session, ch)
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

func (es *eventStore) ViewChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	chann, err := es.svc.ViewChannel(ctx, session, id)
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

func (es *eventStore) ListChannels(ctx context.Context, session authn.Session, pm channels.PageMetadata) (channels.Page, error) {
	cp, err := es.svc.ListChannels(ctx, session, pm)
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

func (es *eventStore) ListChannelsByClient(ctx context.Context, session authn.Session, clientID string, pm channels.PageMetadata) (channels.Page, error) {
	cp, err := es.svc.ListChannelsByClient(ctx, session, clientID, pm)
	if err != nil {
		return cp, err
	}
	event := listChannelByClientEvent{
		clientID,
		pm,
	}
	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) EnableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	cli, err := es.svc.EnableChannel(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) DisableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	cli, err := es.svc.DisableChannel(ctx, session, id)
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

func (es *eventStore) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.RemoveChannel(ctx, session, id); err != nil {
		return err
	}

	event := removeChannelEvent{id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if err := es.svc.Connect(ctx, session, chIDs, thIDs, connTypes); err != nil {
		return err
	}

	event := connectEvent{chIDs, thIDs, connTypes}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if err := es.svc.Disconnect(ctx, session, chIDs, thIDs, connTypes); err != nil {
		return err
	}

	event := disconnectEvent{chIDs, thIDs, connTypes}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	if err := es.svc.SetParentGroup(ctx, session, parentGroupID, id); err != nil {
		return err
	}

	event := setParentGroupEvent{parentGroupID: parentGroupID, id: id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	if err := es.svc.RemoveParentGroup(ctx, session, id); err != nil {
		return err
	}

	event := removeParentGroupEvent{id: id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
