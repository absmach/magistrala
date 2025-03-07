// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/roles"
)

var (
	errID        = errors.New("missing or invalid 'id'")
	errRoleID    = errors.New("missing or invalid 'role_id'")
	errName      = errors.New("missing or invalid 'name'")
	errEntityID  = errors.New("missing or invalid 'entity_id'")
	errActions   = errors.New("missing or invalid 'actions'")
	errMembers   = errors.New("missing or invalid 'members'")
	errCreatedAt = errors.New("failed to parse 'created_at' time")
	errUpdatedAt = errors.New("failed to parse 'updated_at' time")
	errNotString = errors.New("not string type")

	errInvalidRoleProvision = errors.New("invalid 'role_provisions'")
	errRoleProvision        = errors.New("failed to convert role_provisions interface'")
	errRoleProvisionMembers = errors.New("failed to convert role_provisions member interface'")
	errRoleProvisionActions = errors.New("failed to convert role_provisions action interface'")
)

const (
	layout = "2006-01-02T15:04:05.999999Z"
)

func ToRole(data map[string]interface{}) (roles.Role, error) {
	var r roles.Role

	id, ok := data["id"].(string)
	if !ok {
		return roles.Role{}, errID
	}
	r.ID = id

	name, ok := data["name"].(string)
	if !ok {
		return roles.Role{}, errName
	}
	r.Name = name

	eid, ok := data["entity_id"].(string)
	if !ok {
		return roles.Role{}, errEntityID
	}
	r.EntityID = eid

	// Following fields of groups are allowed to be empty.

	cat, ok := data["created_at"].(string)
	if ok {
		ct, err := time.Parse(layout, cat)
		if err != nil {
			return roles.Role{}, errors.Wrap(errCreatedAt, err)
		}
		r.CreatedAt = ct
	}

	cby, ok := data["created_by"].(string)
	if ok {
		r.CreatedBy = cby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return roles.Role{}, errors.Wrap(errUpdatedAt, err)
		}
		r.UpdatedAt = ut
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		r.UpdatedBy = uby
	}

	return r, nil
}

func ToStrings(data []interface{}) ([]string, error) {
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

func ToRoleProvision(data map[string]interface{}) (roles.RoleProvision, error) {
	var rp roles.RoleProvision

	r, err := ToRole(data)
	if err != nil {
		return roles.RoleProvision{}, err
	}
	rp.Role = r

	// Following fields of groups are allowed to be empty.

	opActs, ok := data["optional_actions"].([]interface{})
	if ok {
		a, err := ToStrings(opActs)
		if err != nil {
			return roles.RoleProvision{}, errors.Wrap(errRoleProvisionActions, err)
		}
		rp.OptionalActions = a
	}

	opMems, ok := data["optional_members"].([]interface{})
	if ok {
		m, err := ToStrings(opMems)
		if err != nil {
			return roles.RoleProvision{}, errors.Wrap(errRoleProvisionMembers, err)
		}
		rp.OptionalMembers = m
	}

	return rp, nil
}

func ToRoleProvisions(data []interface{}) ([]roles.RoleProvision, error) {
	var rps []roles.RoleProvision
	for _, d := range data {
		irp, ok := d.(map[string]interface{})
		if !ok {
			return []roles.RoleProvision{}, errInvalidRoleProvision
		}
		rp, err := ToRoleProvision(irp)
		if err != nil {
			return []roles.RoleProvision{}, errors.Wrap(errRoleProvision, err)
		}
		rps = append(rps, rp)
	}
	return rps, nil
}
