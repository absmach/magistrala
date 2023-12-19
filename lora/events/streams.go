// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"encoding/json"
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
	case thingCreate:
		cte, derr := decodeCreateThing(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateThing(ctx, cte.id, cte.loraDevEUI)
	case thingUpdate:
		ute, derr := decodeCreateThing(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateThing(ctx, ute.id, ute.loraDevEUI)

	case channelCreate:
		cce, derr := decodeCreateChannel(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateChannel(ctx, cce.id, cce.loraAppID)
	case channelUpdate:
		uce, derr := decodeCreateChannel(msg)
		if derr != nil {
			err = derr
			break
		}
		err = es.svc.CreateChannel(ctx, uce.id, uce.loraAppID)
	case thingRemove:
		rte := decodeRemoveThing(msg)
		err = es.svc.RemoveThing(ctx, rte.id)
	case channelRemove:
		rce := decodeRemoveChannel(msg)
		err = es.svc.RemoveChannel(ctx, rce.id)
	case thingConnect:
		tce := decodeConnectionThing(msg)
		err = es.svc.ConnectThing(ctx, tce.chanID, tce.thingID)
	case thingDisconnect:
		tde := decodeConnectionThing(msg)
		err = es.svc.DisconnectThing(ctx, tde.chanID, tde.thingID)
	}
	if err != nil && err != errMetadataType {
		return err
	}

	return nil
}

func decodeCreateThing(event map[string]interface{}) (createThingEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createThingEvent{}, err
	}

	cte := createThingEvent{
		id: read(event, "id", ""),
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
		id: read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) (createChannelEvent, error) {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), &metadata); err != nil {
		return createChannelEvent{}, err
	}

	cce := createChannelEvent{
		id: read(event, "id", ""),
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
		chanID:  read(event, "chan_id", ""),
		thingID: read(event, "thing_id", ""),
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
