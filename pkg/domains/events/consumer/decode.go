// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/roles"
	rconsumer "github.com/absmach/supermq/pkg/roles/rolemanager/events/consumer"
)

const (
	layout = "2006-01-02T15:04:05.999999Z"
)

var (
	errDecodeCreateDomainEvent  = errors.New("failed to decode domain create event")
	errDecodeUpdateDomainEvent  = errors.New("failed to decode domain update event")
	errDecodeEnableDomainEvent  = errors.New("failed to decode domain enable event")
	errDecodeDisableDomainEvent = errors.New("failed to decode domain disable event")
	errDecodeFreezeDomainEvent  = errors.New("failed to decode domain freeze event")
	errDecodeRemoveDomainsEvent = errors.New("failed to decode domain remove  event")

	errID            = errors.New("missing or invalid 'id'")
	errName          = errors.New("missing or invalid 'name'")
	errRoute         = errors.New("missing or invalid 'route'")
	errTags          = errors.New("invalid 'tags'")
	errStatus        = errors.New("missing or invalid 'status'")
	errConvertStatus = errors.New("failed to convert status")
	errCreatedBy     = errors.New("missing or invalid 'created_by'")
	errCreatedAt     = errors.New("failed to parse 'created_at' time")
	errUpdatedAt     = errors.New("failed to parse 'updated_at' time")
	errEntityID      = errors.New("missing or invalid 'entity_id'")
	errMembers       = errors.New("missing or invalid 'members'")
	errNotString     = errors.New("not string type")
)

func ToDomains(data map[string]any) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errID
	}
	d.ID = id

	name, ok := data["name"].(string)
	if !ok {
		return domains.Domain{}, errName
	}
	d.Name = name

	stat, ok := data["status"].(string)
	if !ok {
		return domains.Domain{}, errStatus
	}
	st, err := domains.ToStatus(stat)
	if err != nil {
		return domains.Domain{}, errors.Wrap(errConvertStatus, err)
	}
	d.Status = st

	route, ok := data["route"].(string)
	if !ok {
		return domains.Domain{}, errRoute
	}
	d.Route = route

	cby, ok := data["created_by"].(string)
	if !ok {
		return domains.Domain{}, errCreatedBy
	}
	d.CreatedBy = cby

	cat, ok := data["created_at"].(string)
	if !ok {
		return domains.Domain{}, errCreatedAt
	}
	ct, err := time.Parse(layout, cat)
	if err != nil {
		return domains.Domain{}, errors.Wrap(errCreatedAt, err)
	}
	d.CreatedAt = ct

	// Following fields of groups are allowed to be empty.
	itags, ok := data["tags"].([]any)
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errTags, err)
		}
		d.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]any)
	if ok {
		d.Metadata = meta
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		d.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errUpdatedAt, err)
		}
		d.UpdatedAt = ut
	}

	return d, nil
}

func decodeCreateDomainEvent(data map[string]any) (domains.Domain, []roles.RoleProvision, error) {
	d, err := ToDomains(data)
	if err != nil {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, err)
	}
	irps, ok := data["roles_provisioned"].([]any)
	if !ok {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, errors.New("missing or invalid 'roles_provisioned'"))
	}
	rps, err := rconsumer.ToRoleProvisions(irps)
	if err != nil {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, err)
	}

	return d, rps, nil
}

func decodeUpdateDomainEvent(data map[string]any) (domains.Domain, error) {
	var d domains.Domain

	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeUpdateDomainEvent, errID)
	}
	d.ID = id

	name, ok := data["name"].(string)
	if ok {
		d.Name = name
	}

	route, ok := data["route"].(string)
	if ok {
		d.Route = route
	}

	itags, ok := data["tags"].([]any)
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeUpdateDomainEvent, err)
		}
		d.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]any)
	if ok {
		d.Metadata = meta
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		d.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeUpdateDomainEvent, errors.Wrap(errUpdatedAt, err))
		}
		d.UpdatedAt = ut
	}

	return d, nil
}

func decodeEnableDomainEvent(data map[string]any) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeEnableDomainEvent, errID)
	}
	d.ID = id

	uby, ok := data["updated_by"].(string)
	if ok {
		d.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeEnableDomainEvent, errors.Wrap(errUpdatedAt, err))
		}
		d.UpdatedAt = ut
	}

	return d, nil
}

func decodeDisableDomainEvent(data map[string]any) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeDisableDomainEvent, errID)
	}
	d.ID = id

	uby, ok := data["updated_by"].(string)
	if ok {
		d.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeDisableDomainEvent, errors.Wrap(errUpdatedAt, err))
		}
		d.UpdatedAt = ut
	}

	return d, nil
}

func decodeFreezeDomainEvent(data map[string]any) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeFreezeDomainEvent, errID)
	}
	d.ID = id

	uby, ok := data["updated_by"].(string)
	if ok {
		d.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeFreezeDomainEvent, errors.Wrap(errUpdatedAt, err))
		}
		d.UpdatedAt = ut
	}

	return d, nil
}

func decodeRemoveDomainMembersEvent(data map[string]any) (string, []string, error) {
	entityID, ok := data["entity_id"].(string)
	if !ok {
		return "", nil, errors.Wrap(errRemoveDomainMembersEvent, errEntityID)
	}
	imems, ok := data["members"].([]any)
	if !ok {
		return "", nil, errors.Wrap(errRemoveDomainMembersEvent, errMembers)
	}
	mems, err := toStrings(imems)
	if err != nil {
		return "", nil, errors.Wrap(errRemoveDomainMembersEvent, err)
	}

	return entityID, mems, nil
}

func decodeDeleteDomainEvent(data map[string]any) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeRemoveDomainsEvent, errID)
	}
	d.ID = id
	return d, nil
}

func toStrings(data []any) ([]string, error) {
	var strs []string
	for _, i := range data {
		str, ok := i.(string)
		if !ok {
			return []string{}, errNotString
		}
		strs = append(strs, str)
	}
	return strs, nil
}
