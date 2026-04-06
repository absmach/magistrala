// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/roles"
	rconsumer "github.com/absmach/magistrala/pkg/roles/rolemanager/events/consumer"
)

const layout = "2006-01-02T15:04:05.999999Z"

var (
	errDecodeCreateClientEvent       = errors.New("failed to decode client create event")
	errDecodeUpdateClientEvent       = errors.New("failed to decode client update event")
	errDecodeChangeStatusClientEvent = errors.New("failed to decode client change status event")
	errDecodeRemoveClientEvent       = errors.New("failed to decode client remove event")
	errDecodeSetParentGroupEvent     = errors.New("failed to decode client set parent event")
	errDecodeRemoveParentGroupEvent  = errors.New("failed to decode client remove parent event")

	errID            = errors.New("missing or invalid 'id'")
	errDomain        = errors.New("missing or invalid 'domain'")
	errStatus        = errors.New("missing or invalid 'status'")
	errTags          = errors.New("invalid 'tags'")
	errConvertStatus = errors.New("failed to convert status")
	errCreatedAt     = errors.New("failed to parse 'created_at' time")
	errUpdatedAt     = errors.New("failed to parse 'updated_at' time")
)

func ToClient(data map[string]any) (clients.Client, error) {
	var c clients.Client
	id, ok := data["id"].(string)
	if !ok {
		return clients.Client{}, errID
	}
	c.ID = id

	dom, ok := data["domain"].(string)
	if !ok {
		return clients.Client{}, errDomain
	}
	c.Domain = dom

	st, ok := data["status"].(string)
	if !ok {
		return clients.Client{}, errStatus
	}
	status, err := clients.ToStatus(st)
	if err != nil {
		return clients.Client{}, errConvertStatus
	}
	c.Status = status

	cat, ok := data["created_at"].(string)
	if !ok {
		return clients.Client{}, errCreatedAt
	}
	ct, err := time.Parse(layout, cat)
	if err != nil {
		return clients.Client{}, errors.Wrap(errCreatedAt, err)
	}
	c.CreatedAt = ct

	// Following fields of clients are allowed to be empty.
	name, ok := data["name"].(string)
	if ok {
		c.Name = name
	}

	identity, ok := data["identity"].(string)
	if ok {
		c.Identity = identity
	}

	parent, ok := data["parent_group_id"].(string)
	if ok {
		c.ParentGroup = parent
	}

	itags, ok := data["tags"].([]any)
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return clients.Client{}, errors.Wrap(errTags, err)
		}
		c.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]any)
	if ok {
		c.Metadata = meta
	}

	pmeta, ok := data["private_metadata"].(map[string]any)
	if ok {
		c.PrivateMetadata = pmeta
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		c.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return clients.Client{}, errors.Wrap(errUpdatedAt, err)
		}
		c.UpdatedAt = ut
	}

	return c, nil
}

func decodeCreateClientEvent(data map[string]any) (clients.Client, []roles.RoleProvision, error) {
	c, err := ToClient(data)
	if err != nil {
		return clients.Client{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateClientEvent, err)
	}
	irps, ok := data["roles_provisioned"].([]any)
	if !ok {
		return clients.Client{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateClientEvent, errors.New("missing or invalid 'roles_provisioned'"))
	}
	rps, err := rconsumer.ToRoleProvisions(irps)
	if err != nil {
		return clients.Client{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateClientEvent, err)
	}

	return c, rps, nil
}

func decodeUpdateClientEvent(data map[string]any) (clients.Client, error) {
	c, err := ToClient(data)
	if err != nil {
		return clients.Client{}, errors.Wrap(errDecodeUpdateClientEvent, err)
	}
	return c, nil
}

func decodeChangeStatusClientEvent(data map[string]any) (clients.Client, error) {
	c, err := ToClientStatus(data)
	if err != nil {
		return clients.Client{}, errors.Wrap(errDecodeChangeStatusClientEvent, err)
	}
	return c, nil
}

func ToClientStatus(data map[string]any) (clients.Client, error) {
	var c clients.Client
	id, ok := data["id"].(string)
	if !ok {
		return clients.Client{}, errID
	}
	c.ID = id

	stat, ok := data["status"].(string)
	if !ok {
		return clients.Client{}, errStatus
	}
	st, err := clients.ToStatus(stat)
	if err != nil {
		return clients.Client{}, errors.Wrap(errConvertStatus, err)
	}
	c.Status = st

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return clients.Client{}, errors.Wrap(errUpdatedAt, err)
		}
		c.UpdatedAt = ut
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		c.UpdatedBy = uby
	}

	return c, nil
}

func decodeRemoveClientEvent(data map[string]any) (clients.Client, error) {
	var c clients.Client
	id, ok := data["id"].(string)
	if !ok {
		return clients.Client{}, errors.Wrap(errDecodeRemoveClientEvent, errID)
	}
	c.ID = id

	return c, nil
}

func decodeSetParentGroupEvent(data map[string]any) (clients.Client, error) {
	id, ok := data["id"].(string)
	if !ok {
		return clients.Client{}, errors.Wrap(errDecodeSetParentGroupEvent, errID)
	}

	parent, ok := data["parent_group_id"].(string)
	if !ok {
		return clients.Client{}, errors.Wrap(errDecodeSetParentGroupEvent, errID)
	}

	return clients.Client{
		ID:          id,
		ParentGroup: parent,
	}, nil
}

func decodeRemoveParentGroupEvent(data map[string]any) (clients.Client, error) {
	id, ok := data["id"].(string)
	if !ok {
		return clients.Client{}, errors.Wrap(errDecodeRemoveParentGroupEvent, errID)
	}

	return clients.Client{
		ID: id,
	}, nil
}
