// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/absmach/magistrala/opcua"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	keyType      = "opcua"
	keyNodeID    = "node_id"
	keyServerURI = "server_uri"

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
	errMetadataType = errors.New("metadatada is not of type opcua")

	errMetadataFormat = errors.New("malformed metadata")

	errMetadataServerURI = errors.New("ServerURI not found in channel metadatada")

	errMetadataNodeID = errors.New("NodeID not found in thing metadatada")
)

type eventHandler struct {
	svc opcua.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc opcua.Service) events.EventHandler {
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
		cte, e := decodeCreateThing(msg)
		if e != nil {
			err = e
			break
		}
		err = es.svc.CreateThing(ctx, cte.id, cte.opcuaNodeID)
	case thingUpdate:
		ute, e := decodeCreateThing(msg)
		if e != nil {
			err = e
			break
		}
		err = es.svc.CreateThing(ctx, ute.id, ute.opcuaNodeID)
	case thingRemove:
		rte := decodeRemoveThing(msg)
		err = es.svc.RemoveThing(ctx, rte.id)
	case channelCreate:
		cce, e := decodeCreateChannel(msg)
		if e != nil {
			err = e
			break
		}
		err = es.svc.CreateChannel(ctx, cce.id, cce.opcuaServerURI)
	case channelUpdate:
		uce, e := decodeCreateChannel(msg)
		if e != nil {
			err = e
			break
		}
		err = es.svc.CreateChannel(ctx, uce.id, uce.opcuaServerURI)
	case channelRemove:
		rce := decodeRemoveChannel(msg)
		err = es.svc.RemoveChannel(ctx, rce.id)
	case thingConnect:
		rce := decodeConnectThing(msg)
		err = es.svc.ConnectThing(ctx, rce.chanID, rce.thingID)
	case thingDisconnect:
		rce := decodeDisconnectThing(msg)
		err = es.svc.DisconnectThing(ctx, rce.chanID, rce.thingID)
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

	metadataOpcua, ok := metadata[keyType]
	if !ok {
		return createThingEvent{}, errMetadataType
	}

	metadataVal, ok := metadataOpcua.(map[string]interface{})
	if !ok {
		return createThingEvent{}, errMetadataFormat
	}

	val, ok := metadataVal[keyNodeID].(string)
	if !ok || val == "" {
		return createThingEvent{}, errMetadataNodeID
	}

	cte.opcuaNodeID = val
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

	metadataOpcua, ok := metadata[keyType]
	if !ok {
		return createChannelEvent{}, errMetadataType
	}

	metadataVal, ok := metadataOpcua.(map[string]interface{})
	if !ok {
		return createChannelEvent{}, errMetadataFormat
	}

	val, ok := metadataVal[keyServerURI].(string)
	if !ok || val == "" {
		return createChannelEvent{}, errMetadataServerURI
	}

	cce.opcuaServerURI = val
	return cce, nil
}

func decodeRemoveChannel(event map[string]interface{}) removeChannelEvent {
	return removeChannelEvent{
		id: read(event, "id", ""),
	}
}

func decodeConnectThing(event map[string]interface{}) connectThingEvent {
	return connectThingEvent{
		chanID:  read(event, "chan_id", ""),
		thingID: read(event, "thing_id", ""),
	}
}

func decodeDisconnectThing(event map[string]interface{}) connectThingEvent {
	return connectThingEvent{
		chanID:  read(event, "chan_id", ""),
		thingID: read(event, "thing_id", ""),
	}
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
