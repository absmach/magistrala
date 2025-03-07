// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"fmt"
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
	errAlias         = errors.New("missing or invalid 'alias'")
	errTags          = errors.New("invalid 'tags'")
	errStatus        = errors.New("missing or invalid 'status'")
	errConvertStatus = errors.New("failed to convert status")
	errCreatedBy     = errors.New("missing or invalid 'created_by'")
	errCreatedAt     = errors.New("failed to parse 'created_at' time")
	errUpdatedAt     = errors.New("failed to parse 'updated_at' time")
)

func ToDomains(data map[string]interface{}) (domains.Domain, error) {
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

	alias, ok := data["alias"].(string)
	if !ok {
		return domains.Domain{}, errAlias
	}
	d.Alias = alias

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
	itags, ok := data["tags"].([]interface{})
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errTags, err)
		}
		d.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]interface{})
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

func decodeCreateDomainEvent(data map[string]interface{}) (domains.Domain, []roles.RoleProvision, error) {
	d, err := ToDomains(data)
	if err != nil {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, err)
	}
	irps, ok := data["roles_provisioned"].([]interface{})
	if !ok {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, errors.New("missing or invalid 'roles_provisioned'"))
	}
	rps, err := rconsumer.ToRoleProvisions(irps)
	if err != nil {
		return domains.Domain{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateDomainEvent, err)
	}

	return d, rps, nil
}

func decodeUpdateDomainEvent(data map[string]interface{}) (domains.Domain, error) {
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

	alias, ok := data["alias"].(string)
	if ok {
		d.Alias = alias
	}

	itags, ok := data["tags"].([]interface{})
	if ok {
		tags, err := rconsumer.ToStrings(itags)
		if err != nil {
			return domains.Domain{}, errors.Wrap(errDecodeUpdateDomainEvent, err)
		}
		d.Tags = tags
	}

	meta, ok := data["metadata"].(map[string]interface{})
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

func decodeEnableDomainEvent(data map[string]interface{}) (domains.Domain, error) {
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

func decodeDisableDomainEvent(data map[string]interface{}) (domains.Domain, error) {
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

func decodeFreezeDomainEvent(data map[string]interface{}) (domains.Domain, error) {
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

func decodeUserDeleteDomainEvent(_ map[string]interface{}) (domains.Domain, error) {
	return domains.Domain{}, fmt.Errorf("not implemented decode domain user delete event ")
}

func decodeDeleteDomainEvent(data map[string]interface{}) (domains.Domain, error) {
	var d domains.Domain
	id, ok := data["id"].(string)
	if !ok {
		return domains.Domain{}, errors.Wrap(errDecodeRemoveDomainsEvent, errID)
	}
	d.ID = id
	return d, nil
}
