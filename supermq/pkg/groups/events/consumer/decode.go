// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"time"

	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/roles"
	rconsumer "github.com/absmach/supermq/pkg/roles/rolemanager/events/consumer"
)

var (
	errDecodeCreateGroupEvent          = errors.New("failed to decode group create event")
	errDecodeUpdateGroupEvent          = errors.New("failed to decode group update event")
	errDecodeChangeStatusGroupEvent    = errors.New("failed to decode group change status event")
	errDecodeRemoveGroupEvent          = errors.New("failed to decode group remove event")
	errDecodeAddParentGroupEvent       = errors.New("failed to decode group add parent event")
	errDecodeRemoveParentGroupEvent    = errors.New("failed to decode group remove parent event")
	errDecodeAddChildrenGroupsEvent    = errors.New("failed to decode group add children groups event")
	errDecodeRemoveChildrenGroupsEvent = errors.New("failed to decode group remove children groups event")

	errID            = errors.New("missing or invalid 'id'")
	errName          = errors.New("missing or invalid 'name'")
	errDomain        = errors.New("missing or invalid 'domain'")
	errParent        = errors.New("missing or invalid 'parent'")
	errChildrenIDs   = errors.New("missing or invalid 'children_ids'")
	errStatus        = errors.New("missing or invalid 'status'")
	errConvertStatus = errors.New("failed to convert status")
	errCreatedAt     = errors.New("failed to parse 'created_at' time")
	errUpdatedAt     = errors.New("failed to parse 'updated_at' time")
)

const (
	layout = "2006-01-02T15:04:05.999999Z"
)

func ToGroups(data map[string]interface{}) (groups.Group, error) {
	var g groups.Group
	id, ok := data["id"].(string)
	if !ok {
		return groups.Group{}, errID
	}
	g.ID = id

	name, ok := data["name"].(string)
	if !ok {
		return groups.Group{}, errName
	}
	g.Name = name

	dom, ok := data["domain"].(string)
	if !ok {
		return groups.Group{}, errDomain
	}
	g.Domain = dom

	stat, ok := data["status"].(string)
	if !ok {
		return groups.Group{}, errStatus
	}
	st, err := groups.ToStatus(stat)
	if err != nil {
		return groups.Group{}, errors.Wrap(errConvertStatus, err)
	}
	g.Status = st

	cat, ok := data["created_at"].(string)
	if !ok {
		return groups.Group{}, errCreatedAt
	}
	ct, err := time.Parse(layout, cat)
	if err != nil {
		return groups.Group{}, errors.Wrap(errCreatedAt, err)
	}
	g.CreatedAt = ct

	// Following fields of groups are allowed to be empty.

	desc, ok := data["description"].(string)
	if ok {
		g.Description = desc
	}

	parent, ok := data["parent"].(string)
	if ok {
		g.Parent = parent
	}

	meta, ok := data["metadata"].(map[string]interface{})
	if ok {
		g.Metadata = meta
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		g.UpdatedBy = uby
	}

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return groups.Group{}, errors.Wrap(errUpdatedAt, err)
		}
		g.UpdatedAt = ut
	}

	return g, nil
}

func decodeCreateGroupEvent(data map[string]interface{}) (groups.Group, []roles.RoleProvision, error) {
	g, err := ToGroups(data)
	if err != nil {
		return groups.Group{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateGroupEvent, err)
	}
	irps, ok := data["roles_provisioned"].([]interface{})
	if !ok {
		return groups.Group{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateGroupEvent, errors.New("missing or invalid 'roles_provisioned'"))
	}
	rps, err := rconsumer.ToRoleProvisions(irps)
	if err != nil {
		return groups.Group{}, []roles.RoleProvision{}, errors.Wrap(errDecodeCreateGroupEvent, err)
	}

	return g, rps, nil
}

func decodeUpdateGroupEvent(data map[string]interface{}) (groups.Group, error) {
	g, err := ToGroups(data)
	if err != nil {
		return groups.Group{}, errors.Wrap(errDecodeUpdateGroupEvent, err)
	}
	return g, nil
}

func ToGroupStatus(data map[string]interface{}) (groups.Group, error) {
	var g groups.Group
	id, ok := data["id"].(string)
	if !ok {
		return groups.Group{}, errID
	}
	g.ID = id

	stat, ok := data["status"].(string)
	if !ok {
		return groups.Group{}, errStatus
	}
	st, err := groups.ToStatus(stat)
	if err != nil {
		return groups.Group{}, errors.Wrap(errConvertStatus, err)
	}
	g.Status = st

	uat, ok := data["updated_at"].(string)
	if ok {
		ut, err := time.Parse(layout, uat)
		if err != nil {
			return groups.Group{}, errors.Wrap(errUpdatedAt, err)
		}
		g.UpdatedAt = ut
	}

	uby, ok := data["updated_by"].(string)
	if ok {
		g.UpdatedBy = uby
	}

	return g, nil
}

func decodeChangeStatusGroupEvent(data map[string]interface{}) (groups.Group, error) {
	g, err := ToGroupStatus(data)
	if err != nil {
		return groups.Group{}, errors.Wrap(errDecodeChangeStatusGroupEvent, err)
	}
	return g, nil
}

func decodeRemoveGroupEvent(data map[string]interface{}) (groups.Group, error) {
	var g groups.Group
	id, ok := data["id"].(string)
	if !ok {
		return groups.Group{}, errors.Wrap(errDecodeRemoveGroupEvent, errID)
	}
	g.ID = id

	return g, nil
}

func decodeAddParentGroupEvent(data map[string]interface{}) (id string, parent string, err error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", "", errors.Wrap(errAddParentGroupEvent, errID)
	}

	parent, ok = data["parent_id"].(string)
	if !ok {
		return "", "", errors.Wrap(errDecodeAddParentGroupEvent, errParent)
	}

	return id, parent, nil
}

func decodeRemoveParentGroupEvent(data map[string]interface{}) (id string, err error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", errors.Wrap(errDecodeRemoveParentGroupEvent, errID)
	}

	return id, nil
}

func decodeAddChildrenGroupEvent(data map[string]interface{}) (id string, childrenIDs []string, err error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", []string{}, errors.Wrap(errDecodeAddChildrenGroupsEvent, errID)
	}
	chIDs, ok := data["children_ids"].([]interface{})
	if !ok {
		return "", []string{}, errors.Wrap(errDecodeAddChildrenGroupsEvent, errChildrenIDs)
	}
	cids, err := rconsumer.ToStrings(chIDs)
	if err != nil {
		return "", []string{}, errors.Wrap(errDecodeAddChildrenGroupsEvent, errors.Wrap(errChildrenIDs, err))
	}
	return id, cids, nil
}

func decodeRemoveChildrenGroupEvent(data map[string]interface{}) (id string, childrenIDs []string, err error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", []string{}, errors.Wrap(errDecodeRemoveChildrenGroupsEvent, errID)
	}
	chIDs, ok := data["children_ids"].([]interface{})
	if !ok {
		return "", []string{}, errors.Wrap(errDecodeRemoveChildrenGroupsEvent, errChildrenIDs)
	}
	cids, err := rconsumer.ToStrings(chIDs)
	if err != nil {
		return "", []string{}, errors.Wrap(errDecodeRemoveChildrenGroupsEvent, errors.Wrap(errChildrenIDs, err))
	}
	return id, cids, nil
}

func decodeRemoveAllChildrenGroupEvent(data map[string]interface{}) (id string, err error) {
	id, ok := data["id"].(string)
	if !ok {
		return "", errors.Wrap(errDecodeRemoveChildrenGroupsEvent, errID)
	}

	return id, nil
}
