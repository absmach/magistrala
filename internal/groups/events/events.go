// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/events"
	groups "github.com/absmach/magistrala/pkg/groups"
)

const (
	groupPrefix          = "group."
	groupCreate          = groupPrefix + "create"
	groupUpdate          = groupPrefix + "update"
	groupChangeStatus    = groupPrefix + "change_status"
	groupView            = groupPrefix + "view"
	groupViewPerms       = groupPrefix + "view_perms"
	groupList            = groupPrefix + "list"
	groupListMemberships = groupPrefix + "list_by_user"
	groupRemove          = groupPrefix + "remove"
)

var (
	_ events.Event = (*createGroupEvent)(nil)
	_ events.Event = (*updateGroupEvent)(nil)
	_ events.Event = (*changeStatusGroupEvent)(nil)
	_ events.Event = (*viewGroupEvent)(nil)
	_ events.Event = (*deleteGroupEvent)(nil)
	_ events.Event = (*viewGroupEvent)(nil)
	_ events.Event = (*listGroupEvent)(nil)
	_ events.Event = (*listGroupMembershipEvent)(nil)
)

type createGroupEvent struct {
	groups.Group
}

func (cge createGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupCreate,
		"id":         cge.ID,
		"status":     cge.Status.String(),
		"created_at": cge.CreatedAt,
	}

	if cge.Owner != "" {
		val["owner"] = cge.Owner
	}
	if cge.Parent != "" {
		val["parent"] = cge.Parent
	}
	if cge.Name != "" {
		val["name"] = cge.Name
	}
	if cge.Description != "" {
		val["description"] = cge.Description
	}
	if cge.Metadata != nil {
		metadata, err := json.Marshal(cge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if cge.Status.String() != "" {
		val["status"] = cge.Status.String()
	}

	return val, nil
}

type updateGroupEvent struct {
	groups.Group
}

func (uge updateGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupUpdate,
		"updated_at": uge.UpdatedAt,
		"updated_by": uge.UpdatedBy,
	}

	if uge.ID != "" {
		val["id"] = uge.ID
	}
	if uge.Owner != "" {
		val["owner"] = uge.Owner
	}
	if uge.Parent != "" {
		val["parent"] = uge.Parent
	}
	if uge.Name != "" {
		val["name"] = uge.Name
	}
	if uge.Description != "" {
		val["description"] = uge.Description
	}
	if uge.Metadata != nil {
		metadata, err := json.Marshal(uge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !uge.CreatedAt.IsZero() {
		val["created_at"] = uge.CreatedAt
	}
	if uge.Status.String() != "" {
		val["status"] = uge.Status.String()
	}

	return val, nil
}

type changeStatusGroupEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rge changeStatusGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  groupChangeStatus,
		"id":         rge.id,
		"status":     rge.status,
		"updated_at": rge.updatedAt,
		"updated_by": rge.updatedBy,
	}, nil
}

type viewGroupEvent struct {
	groups.Group
}

func (vge viewGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupView,
		"id":        vge.ID,
	}

	if vge.Owner != "" {
		val["owner"] = vge.Owner
	}
	if vge.Parent != "" {
		val["parent"] = vge.Parent
	}
	if vge.Name != "" {
		val["name"] = vge.Name
	}
	if vge.Description != "" {
		val["description"] = vge.Description
	}
	if vge.Metadata != nil {
		metadata, err := json.Marshal(vge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !vge.CreatedAt.IsZero() {
		val["created_at"] = vge.CreatedAt
	}
	if !vge.UpdatedAt.IsZero() {
		val["updated_at"] = vge.UpdatedAt
	}
	if vge.UpdatedBy != "" {
		val["updated_by"] = vge.UpdatedBy
	}
	if vge.Status.String() != "" {
		val["status"] = vge.Status.String()
	}

	return val, nil
}

type viewGroupPermsEvent struct {
	permissions []string
}

func (vgpe viewGroupPermsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupViewPerms,
		"permissions": vgpe.permissions,
	}
	return val, nil
}

type listGroupEvent struct {
	groups.Page
}

func (lge listGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupList,
		"total":     lge.Total,
		"offset":    lge.Offset,
		"limit":     lge.Limit,
	}

	if lge.Name != "" {
		val["name"] = lge.Name
	}
	if lge.OwnerID != "" {
		val["owner_id"] = lge.OwnerID
	}
	if lge.Tag != "" {
		val["tag"] = lge.Tag
	}
	if lge.Metadata != nil {
		metadata, err := json.Marshal(lge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lge.Status.String() != "" {
		val["status"] = lge.Status.String()
	}

	return val, nil
}

type listGroupMembershipEvent struct {
	groupID    string
	permission string
	memberKind string
}

func (lgme listGroupMembershipEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupListMemberships,
		"group_id":    lgme.groupID,
		"permission":  lgme.permission,
		"member_kind": lgme.memberKind,
	}

	return val, nil
}

type deleteGroupEvent struct {
	id string
}

func (rge deleteGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": groupRemove,
		"id":        rge.id,
	}, nil
}
