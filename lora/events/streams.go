// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"

	"github.com/absmach/magistrala/lora"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	keyType   = "lora"
	keyDevEUI = "dev_eui"
	keyAppID  = "app_id"

	thingPrefix     = "thing."
	thingCreate     = thingPrefix + "create"
	thingUpdate     = thingPrefix + "update"
	thingRemove     = thingPrefix + "remove"
	thingConnect    = thingPrefix + "connect"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "group."
	channelCreate = channelPrefix + "create"
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"
)

var (
	errMetadataType = errors.New("field lora is missing in the metadata")

	errMetadataFormat = errors.New("malformed metadata")

	errMetadataAppID = errors.New("application ID not found in channel metadatada")

	errMetadataDevEUI = errors.New("device EUI not found in thing metadatada")
)

type eventHandler struct {
	svc lora.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc lora.Service) events.EventHandler {
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
	case thingCreate, thingUpdate:
		cte, derr := decodeCreateThing(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateThing(ctx, cte.id, cte.loraDevEUI)
	case channelCreate, channelUpdate:
		cce, derr := decodeCreateChannel(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateChannel(ctx, cce.id, cce.loraAppID)
	case thingRemove:
		rte := decodeRemoveThing(msg)
		err = es.svc.RemoveThing(ctx, rte.id)
	case channelRemove:
		rce := decodeRemoveChannel(msg)
		err = es.svc.RemoveChannel(ctx, rce.id)
	case thingConnect:
		tce := decodeConnectionThing(msg)

		for _, thingID := range tce.thingIDs {
			err = es.svc.ConnectThing(ctx, tce.chanID, thingID)
			if err != nil {
				return err
			}
		}
	case thingDisconnect:
		tde := decodeConnectionThing(msg)

		for _, thingID := range tde.thingIDs {
			err = es.svc.DisconnectThing(ctx, tde.chanID, thingID)
			if err != nil {
				return err
			}
		}
	}
	if err != nil && err != errMetadataType {
		return err
	}

	return nil
}

func decodeCreateThing(event map[string]interface{}) (createThingEvent, error) {
	metadata := events.Read(event, "metadata", map[string]interface{}{})

	cte := createThingEvent{
		id: events.Read(event, "id", ""),
	}

	m, ok := metadata[keyType]
	if !ok {
		return createThingEvent{}, errMetadataType
	}

	lm, ok := m.(map[string]interface{})
	if !ok {
		return createThingEvent{}, errMetadataFormat
	}

	val, ok := lm[keyDevEUI].(string)
	if !ok {
		return createThingEvent{}, errMetadataDevEUI
	}

	cte.loraDevEUI = val
	return cte, nil
}

func decodeRemoveThing(event map[string]interface{}) removeThingEvent {
	return removeThingEvent{
		id: events.Read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) (createChannelEvent, error) {
	metadata := events.Read(event, "metadata", map[string]interface{}{})

	cce := createChannelEvent{
		id: events.Read(event, "id", ""),
	}

	m, ok := metadata[keyType]
	if !ok {
		return createChannelEvent{}, errMetadataType
	}

	lm, ok := m.(map[string]interface{})
	if !ok {
		return createChannelEvent{}, errMetadataFormat
	}

	val, ok := lm[keyAppID].(string)
	if !ok {
		return createChannelEvent{}, errMetadataAppID
	}

	cce.loraAppID = val
	return cce, nil
}

func decodeConnectionThing(event map[string]interface{}) connectionThingEvent {
	return connectionThingEvent{
		chanID:   events.Read(event, "group_id", ""),
		thingIDs: events.ReadStringSlice(event, "member_ids"),
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: events.Read(event, "id", ""),
	}
}
