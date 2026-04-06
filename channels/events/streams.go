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
	"github.com/absmach/magistrala/pkg/roles"
	rmEvents "github.com/absmach/magistrala/pkg/roles/rolemanager/events"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	magistralaPrefix   = "magistrala."
	createStream       = magistralaPrefix + channelCreate
	updateStream       = magistralaPrefix + channelUpdate
	updateTagsStream   = magistralaPrefix + channelUpdateTags
	enableStream       = magistralaPrefix + channelEnable
	disableStream      = magistralaPrefix + channelDisable
	removeStream       = magistralaPrefix + channelRemove
	viewStream         = magistralaPrefix + channelView
	listStream         = magistralaPrefix + channelList
	listByUserStream   = magistralaPrefix + channelListByUser
	connectStream      = magistralaPrefix + channelConnect
	disconnectStream   = magistralaPrefix + channelDisconnect
	setParentStream    = magistralaPrefix + channelSetParent
	removeParentStream = magistralaPrefix + channelRemoveParent
)

var _ channels.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc channels.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around clients service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc channels.Service, url string) (channels.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, "channels-es-pub")
	if err != nil {
		return nil, err
	}

	rolesSvcEventStoreMiddleware := rmEvents.NewRoleManagerEventStore("channels", channelPrefix, svc, publisher)
	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: rolesSvcEventStoreMiddleware,
	}, nil
}

func (es *eventStore) CreateChannels(ctx context.Context, session authn.Session, chs ...channels.Channel) ([]channels.Channel, []roles.RoleProvision, error) {
	chs, rps, err := es.svc.CreateChannels(ctx, session, chs...)
	if err != nil {
		return chs, rps, err
	}

	for _, ch := range chs {
		event := createChannelEvent{
			Channel:          ch,
			rolesProvisioned: rps,
			Session:          session,
			requestID:        middleware.GetReqID(ctx),
		}
		if err := es.Publish(ctx, createStream, event); err != nil {
			return chs, rps, err
		}
	}

	return chs, rps, nil
}

func (es *eventStore) UpdateChannel(ctx context.Context, session authn.Session, ch channels.Channel) (channels.Channel, error) {
	ch, err := es.svc.UpdateChannel(ctx, session, ch)
	if err != nil {
		return ch, err
	}

	event := updateChannelEvent{
		Channel:   ch,
		Session:   session,
		operation: channelUpdate,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, updateStream, event); err != nil {
		return ch, err
	}

	return ch, nil
}

func (es *eventStore) UpdateChannelTags(ctx context.Context, session authn.Session, ch channels.Channel) (channels.Channel, error) {
	ch, err := es.svc.UpdateChannelTags(ctx, session, ch)
	if err != nil {
		return ch, err
	}

	event := updateChannelEvent{
		Channel:   ch,
		Session:   session,
		operation: channelUpdateTags,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, updateTagsStream, event); err != nil {
		return ch, err
	}

	return ch, nil
}

func (es *eventStore) ViewChannel(ctx context.Context, session authn.Session, id string, withRoles bool) (channels.Channel, error) {
	chann, err := es.svc.ViewChannel(ctx, session, id, withRoles)
	if err != nil {
		return chann, err
	}

	event := viewChannelEvent{
		Channel:   chann,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, viewStream, event); err != nil {
		return chann, err
	}

	return chann, nil
}

func (es *eventStore) ListChannels(ctx context.Context, session authn.Session, pm channels.Page) (channels.ChannelsPage, error) {
	cp, err := es.svc.ListChannels(ctx, session, pm)
	if err != nil {
		return cp, err
	}
	event := listChannelEvent{
		Page:      pm,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, listStream, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) ListUserChannels(ctx context.Context, session authn.Session, userID string, pm channels.Page) (channels.ChannelsPage, error) {
	cp, err := es.svc.ListUserChannels(ctx, session, userID, pm)
	if err != nil {
		return cp, err
	}
	event := listUserChannelsEvent{
		userID:    userID,
		Page:      pm,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, listByUserStream, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) EnableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	ch, err := es.svc.EnableChannel(ctx, session, id)
	if err != nil {
		return ch, err
	}

	return es.changeStatus(ctx, session, channelEnable, enableStream, ch)
}

func (es *eventStore) DisableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	ch, err := es.svc.DisableChannel(ctx, session, id)
	if err != nil {
		return ch, err
	}

	return es.changeStatus(ctx, session, channelDisable, disableStream, ch)
}

func (es *eventStore) changeStatus(ctx context.Context, session authn.Session, operation, stream string, ch channels.Channel) (channels.Channel, error) {
	event := changeChannelStatusEvent{
		id:        ch.ID,
		operation: operation,
		updatedAt: ch.UpdatedAt,
		updatedBy: ch.UpdatedBy,
		status:    ch.Status.String(),
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}
	if err := es.Publish(ctx, stream, event); err != nil {
		return ch, err
	}

	return ch, nil
}

func (es *eventStore) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.RemoveChannel(ctx, session, id); err != nil {
		return err
	}

	event := removeChannelEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, removeStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if err := es.svc.Connect(ctx, session, chIDs, thIDs, connTypes); err != nil {
		return err
	}

	event := connectEvent{
		chIDs:     chIDs,
		thIDs:     thIDs,
		types:     connTypes,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, connectStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	if err := es.svc.Disconnect(ctx, session, chIDs, thIDs, connTypes); err != nil {
		return err
	}

	event := disconnectEvent{
		chIDs:     chIDs,
		thIDs:     thIDs,
		types:     connTypes,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, disconnectStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	if err := es.svc.SetParentGroup(ctx, session, parentGroupID, id); err != nil {
		return err
	}

	event := setParentGroupEvent{
		parentGroupID: parentGroupID,
		id:            id,
		Session:       session,
		requestID:     middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, setParentStream, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	if err := es.svc.RemoveParentGroup(ctx, session, id); err != nil {
		return err
	}

	event := removeParentGroupEvent{
		id:        id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, removeParentStream, event); err != nil {
		return err
	}

	return nil
}
