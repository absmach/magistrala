// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/roles"
	rconsumer "github.com/absmach/magistrala/pkg/roles/rolemanager/events/consumer"
)

const layout = "2006-01-02T15:04:05.999999Z"

var (
	errDecodeCreateChannelEvent       = errors.New("failed to decode channel create event")
	errDecodeUpdateChannelEvent       = errors.New("failed to decode channel update event")
	errDecodeChangeStatusChannelEvent = errors.New("failed to decode channel change status event")
	errDecodeRemoveChannelEvent       = errors.New("failed to decode channel remove event")
	errDecodeSetParentGroupEvent      = errors.New("failed to decode channel set parent event")
	errDecodeRemoveParentGroupEvent   = errors.New("failed to decode channel remove parent event")
	errDecodeConnectEvent             = errors.New("failed to decode channel connect event")
	errDeocodeDisconnectEvent         = errors.New("failed to decode channel disconnect event")

	errID            = errors.New("missing or invalid 'id'")
	errDomain        = errors.New("missing or invalid 'domain'")
	errStatus        = errors.New("missing or invalid 'status'")
	errTags          = errors.New("invalid 'tags'")
	errConvertStatus = errors.New("failed to convert status")
	errChannelIDs    = errors.New("missing or invalid 'channel_ids' in connection")
	errClientIDs     = errors.New("missing or invalid 'client_ids' in connection")
	errConnType      = errors.New("missing or invalid 'type' in connection")
	errCreatedAt     = errors.New("failed to parse 'created_at' time")
	errUpdatedAt     = errors.New("failed to parse 'updated_at' time")
)

func ToChannel(data map[string]any) (channels.Channel, error) {
	var c channels.Channel
	id, ok := data["id"].(string)
	if !ok {
		return channels.Channel{}, errID
	}
	c.ID = id

	dom, ok := data["domain"].(string)
	if !ok {
		return channels.Channel{}, errDomain
	}
	c.Domain = dom

	st, ok := data["status"].(string)
	if !ok {
		return channels.Channel{}, errStatus
	}
	status, err := channels.ToStatus(st)
	if err != nil {
		return channels.Channel{}, errConvertStatus
	}
	c.Status = status

	cat, ok := data["created_at"].(string)
	if !ok {
		return channels.Channel{}, errCreatedAt
	}
	ct, err := time.Parse(layout, cat)
	if err != nil {
		return channels.Channel{}, errors.Wrap(errCreatedAt, err)
	}
	c.CreatedAt = ct

	// Following fields of channels are allowed to be empty.
	name, ok := data["name"].(string)
	if ok {
		c.Name = name
	}

	parent, ok := data["parent_group_id"].(string)
	if ok {
		c.ParentGroup = parent
	}

	itags, ok := data["tags"].([]any)
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return channels.Channel{}, errors.Wrap(errTags, err)
		}
		c.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]any)
	if ok {
		c.Metadata = meta
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		c.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return channels.Channel{}, errors.Wrap(errUpdatedAt, err)
		}
		c.UpdatedAt = ut
	}

	return c, nil
}

func ToConnections(data map[string]any) ([]channels.Connection, error) {
	var connTypes []connections.ConnType
	domain, ok := data["domain"].(string)
	if !ok {
		return nil, errDomain
	}

	ityp, ok := data["types"].([]any)
	if !ok {
		return nil, errConnType
	}
	typs, err := rconsumer.ToStrings(ityp)
	if err != nil {
		return nil, errors.Wrap(errConnType, err)
	}
	for _, typ := range typs {
		connType, err := connections.ParseConnType(typ)
		if err != nil {
			return nil, errors.Wrap(errConnType, err)
		}
		connTypes = append(connTypes, connType)
	}

	ichanIDs, ok := data["channel_ids"].([]any)
	if !ok {
		return []channels.Connection{}, errChannelIDs
	}
	channelIDs, err := rconsumer.ToStrings(ichanIDs)
	if err != nil {
		return []channels.Connection{}, errors.Wrap(errChannelIDs, err)
	}

	iclIDs, ok := data["client_ids"].([]any)
	if !ok {
		return []channels.Connection{}, errClientIDs
	}
	clientIDs, err := rconsumer.ToStrings(iclIDs)
	if err != nil {
		return []channels.Connection{}, errors.Wrap(errClientIDs, err)
	}

	var conns []channels.Connection
	for _, chanID := range channelIDs {
		for _, clientID := range clientIDs {
			for _, connType := range connTypes {
				conns = append(conns, channels.Connection{
					ChannelID: chanID,
					ClientID:  clientID,
					Type:      connType,
					DomainID:  domain,
				})
			}
		}
	}

	return conns, nil
}

func decodeCreateChannelEvent(data map[string]any) (channels.Channel, []roles.RoleProvision, error) {
	c, err := ToChannel(data)
	if err != nil {
		return channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateChannelEvent, err)
	}
	irps, ok := data["roles_provisioned"].([]any)
	if !ok {
		return channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateChannelEvent, errors.New("missing or invalid 'roles_provisioned'"))
	}
	rps, err := rconsumer.ToRoleProvisions(irps)
	if err != nil {
		return channels.Channel{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateChannelEvent, err)
	}

	return c, rps, nil
}

func decodeUpdateChannelEvent(data map[string]any) (channels.Channel, error) {
	c, err := ToChannel(data)
	if err != nil {
		return channels.Channel{}, errors.Wrap(errDecodeUpdateChannelEvent, err)
	}
	return c, nil
}

func decodeChangeStatusChannelEvent(data map[string]any) (channels.Channel, error) {
	c, err := ToChannelStatus(data)
	if err != nil {
		return channels.Channel{}, errors.Wrap(errDecodeChangeStatusChannelEvent, err)
	}
	return c, nil
}

func ToChannelStatus(data map[string]any) (channels.Channel, error) {
	var c channels.Channel
	id, ok := data["id"].(string)
	if !ok {
		return channels.Channel{}, errID
	}
	c.ID = id

	stat, ok := data["status"].(string)
	if !ok {
		return channels.Channel{}, errStatus
	}
	st, err := channels.ToStatus(stat)
	if err != nil {
		return channels.Channel{}, errors.Wrap(errConvertStatus, err)
	}
	c.Status = st

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return channels.Channel{}, errors.Wrap(errUpdatedAt, err)
		}
		c.UpdatedAt = ut
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		c.UpdatedBy = uby
	}

	return c, nil
}

func decodeRemoveChannelEvent(data map[string]any) (channels.Channel, error) {
	var c channels.Channel
	id, ok := data["id"].(string)
	if !ok {
		return channels.Channel{}, errors.Wrap(errDecodeRemoveChannelEvent, errID)
	}
	c.ID = id

	return c, nil
}

func decodeConnectEvent(data map[string]any) ([]channels.Connection, error) {
	conns, err := ToConnections(data)
	if err != nil {
		return []channels.Connection{}, errors.Wrap(errDecodeConnectEvent, err)
	}

	return conns, nil
}

func decodeDisconnectEvent(data map[string]any) ([]channels.Connection, error) {
	conns, err := ToConnections(data)
	if err != nil {
		return []channels.Connection{}, errors.Wrap(errDeocodeDisconnectEvent, err)
	}

	return conns, nil
}

func decodeSetParentGroupEvent(data map[string]any) (channels.Channel, error) {
	id, ok := data["id"].(string)
	if !ok {
		return channels.Channel{}, errors.Wrap(errDecodeSetParentGroupEvent, errID)
	}

	parent, ok := data["parent_group_id"].(string)
	if !ok {
		return channels.Channel{}, errors.Wrap(errDecodeSetParentGroupEvent, errID)
	}

	return channels.Channel{
		ID:          id,
		ParentGroup: parent,
	}, nil
}

func decodeRemoveParentGroupEvent(data map[string]any) (channels.Channel, error) {
	id, ok := data["id"].(string)
	if !ok {
		return channels.Channel{}, errors.Wrap(errDecodeRemoveParentGroupEvent, errID)
	}

	return channels.Channel{
		ID: id,
	}, nil
}
