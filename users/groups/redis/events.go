// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"time"

	mfredis "github.com/mainflux/mainflux/internal/clients/redis"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
)

const (
	groupPrefix          = "group."
	groupCreate          = groupPrefix + "create"
	groupUpdate          = groupPrefix + "update"
	groupRemove          = groupPrefix + "remove"
	groupView            = groupPrefix + "view"
	groupList            = groupPrefix + "list"
	groupListMemberships = groupPrefix + "list_by_group"
)

var (
	_ mfredis.Event = (*createGroupEvent)(nil)
	_ mfredis.Event = (*updateGroupEvent)(nil)
	_ mfredis.Event = (*removeGroupEvent)(nil)
	_ mfredis.Event = (*viewGroupEvent)(nil)
	_ mfredis.Event = (*listGroupEvent)(nil)
	_ mfredis.Event = (*listGroupMembershipEvent)(nil)
)

type createGroupEvent struct {
	mfgroups.Group
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
	mfgroups.Group
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

type removeGroupEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rge removeGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  groupRemove,
		"id":         rge.id,
		"status":     rge.status,
		"updated_at": rge.updatedAt,
		"updated_by": rge.updatedBy,
	}, nil
}

type viewGroupEvent struct {
	mfgroups.Group
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

type listGroupEvent struct {
	mfgroups.GroupsPage
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
	if lge.Action != "" {
		val["action"] = lge.Action
	}
	if lge.Subject != "" {
		val["subject"] = lge.Subject
	}

	return val, nil
}

type listGroupMembershipEvent struct {
	mfgroups.GroupsPage
	channelID string
}

func (lgme listGroupMembershipEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupListMemberships,
		"total":      lgme.Total,
		"offset":     lgme.Offset,
		"limit":      lgme.Limit,
		"channel_id": lgme.channelID,
	}

	if lgme.Name != "" {
		val["name"] = lgme.Name
	}
	if lgme.OwnerID != "" {
		val["owner_id"] = lgme.OwnerID
	}
	if lgme.Tag != "" {
		val["tag"] = lgme.Tag
	}
	if lgme.Metadata != nil {
		metadata, err := json.Marshal(lgme.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lgme.Status.String() != "" {
		val["status"] = lgme.Status.String()
	}
	if lgme.Action != "" {
		val["action"] = lgme.Action
	}
	if lgme.Subject != "" {
		val["subject"] = lgme.Subject
	}

	return val, nil
}
