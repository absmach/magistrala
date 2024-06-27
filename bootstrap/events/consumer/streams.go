// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	thingRemove     = "thing.remove"
	thingConnect    = "group.assign"
	thingDisconnect = "group.unassign"

	channelPrefix = "group."
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"

	memberKind = "things"
	relation   = "group"
)

type eventHandler struct {
	svc bootstrap.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc bootstrap.Service) events.EventHandler {
	return &eventHandler{
		svc: svc,
	}
}

func (es *eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	switch msg["operation"] {
	case thingRemove:
		rte := decodeRemoveThing(msg)
		err = es.svc.RemoveConfigHandler(ctx, rte.id)
	case thingConnect:
		cte := decodeConnectThing(msg)
		if cte.channelID == "" || len(cte.thingIDs) == 0 {
			return svcerr.ErrMalformedEntity
		}
		for _, thingID := range cte.thingIDs {
			if thingID == "" {
				return svcerr.ErrMalformedEntity
			}
			if err := es.svc.ConnectThingHandler(ctx, cte.channelID, thingID); err != nil {
				return err
			}
		}
	case thingDisconnect:
		dte := decodeDisconnectThing(msg)
		if dte.channelID == "" || len(dte.thingIDs) == 0 {
			return svcerr.ErrMalformedEntity
		}
		for _, thingID := range dte.thingIDs {
			if thingID == "" {
				return svcerr.ErrMalformedEntity
			}
		}

		for _, thingID := range dte.thingIDs {
			if err = es.svc.DisconnectThingHandler(ctx, dte.channelID, thingID); err != nil {
				return err
			}
		}
	case channelUpdate:
		uce := decodeUpdateChannel(msg)
		err = es.handleUpdateChannel(ctx, uce)
	case channelRemove:
		rce := decodeRemoveChannel(msg)
		err = es.svc.RemoveChannelHandler(ctx, rce.id)
	}
	if err != nil {
		return err
	}

	return nil
}

func decodeRemoveThing(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: events.Read(event, "id", ""),
	}
}

func decodeUpdateChannel(event map[string]interface{}) updateChannelEvent {
	metadata := events.Read(event, "metadata", map[string]interface{}{})

	return updateChannelEvent{
		id:        events.Read(event, "id", ""),
		name:      events.Read(event, "name", ""),
		metadata:  metadata,
		updatedAt: events.Read(event, "updated_at", time.Now()),
		updatedBy: events.Read(event, "updated_by", ""),
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: events.Read(event, "id", ""),
	}
}

func decodeConnectThing(event map[string]interface{}) connectionEvent {
	if events.Read(event, "memberKind", "") != memberKind && events.Read(event, "relation", "") != relation {
		return connectionEvent{}
	}

	return connectionEvent{
		channelID: events.Read(event, "group_id", ""),
		thingIDs:  events.ReadStringSlice(event, "member_ids"),
	}
}

func decodeDisconnectThing(event map[string]interface{}) connectionEvent {
	if events.Read(event, "memberKind", "") != memberKind && events.Read(event, "relation", "") != relation {
		return connectionEvent{}
	}

	return connectionEvent{
		channelID: events.Read(event, "group_id", ""),
		thingIDs:  events.ReadStringSlice(event, "member_ids"),
	}
}

func (es *eventHandler) handleUpdateChannel(ctx context.Context, uce updateChannelEvent) error {
	channel := bootstrap.Channel{
		ID:        uce.id,
		Name:      uce.name,
		Metadata:  uce.metadata,
		UpdatedAt: uce.updatedAt,
		UpdatedBy: uce.updatedBy,
	}

	return es.svc.UpdateChannelHandler(ctx, channel)
}
