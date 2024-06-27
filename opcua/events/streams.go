// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"

	"github.com/absmach/magistrala/opcua"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	keyType      = "opcua"
	keyNodeID    = "node_id"
	keyServerURI = "server_uri"

	thingPrefix = "thing."
	thingCreate = thingPrefix + "create"
	thingUpdate = thingPrefix + "update"
	thingRemove = thingPrefix + "remove"

	channelPrefix     = "channel."
	channelCreate     = channelPrefix + "create"
	channelUpdate     = channelPrefix + "update"
	channelRemove     = channelPrefix + "remove"
	channelConnect    = channelPrefix + "assign"
	channelDisconnect = channelPrefix + "unassign"
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
	case channelConnect:
		rce := decodeConnectThing(msg)
		err = es.svc.ConnectThing(ctx, rce.chanID, rce.thingIDs)
	case channelDisconnect:
		rce := decodeDisconnectThing(msg)
		err = es.svc.DisconnectThing(ctx, rce.chanID, rce.thingIDs)
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
		id: events.Read(event, "id", ""),
	}
}

func decodeCreateChannel(event map[string]interface{}) (createChannelEvent, error) {
	metadata := events.Read(event, "metadata", map[string]interface{}{})

	cce := createChannelEvent{
		id: events.Read(event, "id", ""),
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
		id: events.Read(event, "id", ""),
	}
}

func decodeConnectThing(event map[string]interface{}) connectThingEvent {
	return connectThingEvent{
		chanID:   events.Read(event, "group_id", ""),
		thingIDs: events.ReadStringSlice(event, "member_ids"),
	}
}

func decodeDisconnectThing(event map[string]interface{}) connectThingEvent {
	return connectThingEvent{
		chanID:   events.Read(event, "group_id", ""),
		thingIDs: events.ReadStringSlice(event, "member_ids"),
	}
}
