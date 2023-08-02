// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"encoding/json"
	"time"

	"github.com/mainflux/mainflux/internal/clients/redis"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
)

const (
	groupPrefix          = "channel."
	groupCreate          = groupPrefix + "create"
	groupUpdate          = groupPrefix + "update"
	groupRemove          = groupPrefix + "remove"
	groupView            = groupPrefix + "view"
	groupList            = groupPrefix + "list"
	groupListMemberships = groupPrefix + "list_by_group"
)

var (
	_ redis.Event = (*createGroupEvent)(nil)
	_ redis.Event = (*updateGroupEvent)(nil)
	_ redis.Event = (*removeGroupEvent)(nil)
	_ redis.Event = (*viewGroupEvent)(nil)
	_ redis.Event = (*listGroupEvent)(nil)
	_ redis.Event = (*listGroupMembershipEvent)(nil)
)

type createGroupEvent struct {
	mfgroups.Group
}

func (cce createGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupCreate,
		"id":         cce.ID,
		"status":     cce.Status.String(),
		"created_at": cce.CreatedAt,
	}

	if cce.Owner != "" {
		val["owner"] = cce.Owner
	}
	if cce.Parent != "" {
		val["parent"] = cce.Parent
	}
	if cce.Name != "" {
		val["name"] = cce.Name
	}
	if cce.Description != "" {
		val["description"] = cce.Description
	}
	if cce.Metadata != nil {
		metadata, err := json.Marshal(cce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if cce.Status.String() != "" {
		val["status"] = cce.Status.String()
	}
	if !cce.CreatedAt.IsZero() {
		val["created_at"] = cce.CreatedAt
	}
	return val, nil
}

type updateGroupEvent struct {
	mfgroups.Group
}

func (uce updateGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupUpdate,
		"updated_at": uce.UpdatedAt,
		"updated_by": uce.UpdatedBy,
	}

	if uce.ID != "" {
		val["id"] = uce.ID
	}
	if uce.Owner != "" {
		val["owner"] = uce.Owner
	}
	if uce.Parent != "" {
		val["parent"] = uce.Parent
	}
	if uce.Name != "" {
		val["name"] = uce.Name
	}
	if uce.Description != "" {
		val["description"] = uce.Description
	}
	if uce.Metadata != nil {
		metadata, err := json.Marshal(uce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !uce.CreatedAt.IsZero() {
		val["created_at"] = uce.CreatedAt
	}
	if uce.Status.String() != "" {
		val["status"] = uce.Status.String()
	}

	return val, nil
}

type removeGroupEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rce removeGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  groupRemove,
		"id":         rce.id,
		"status":     rce.status,
		"updated_at": rce.updatedAt,
		"updated_by": rce.updatedBy,
	}, nil
}

type viewGroupEvent struct {
	mfgroups.Group
}

func (vce viewGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupView,
		"id":        vce.ID,
	}

	if vce.Owner != "" {
		val["owner"] = vce.Owner
	}
	if vce.Parent != "" {
		val["parent"] = vce.Parent
	}
	if vce.Name != "" {
		val["name"] = vce.Name
	}
	if vce.Description != "" {
		val["description"] = vce.Description
	}
	if vce.Metadata != nil {
		metadata, err := json.Marshal(vce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if !vce.CreatedAt.IsZero() {
		val["created_at"] = vce.CreatedAt
	}
	if !vce.UpdatedAt.IsZero() {
		val["updated_at"] = vce.UpdatedAt
	}
	if vce.UpdatedBy != "" {
		val["updated_by"] = vce.UpdatedBy
	}
	if vce.Status.String() != "" {
		val["status"] = vce.Status.String()
	}

	return val, nil
}

type listGroupEvent struct {
	mfgroups.GroupsPage
}

func (lce listGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupList,
		"total":     lce.Total,
		"offset":    lce.Offset,
		"limit":     lce.Limit,
	}

	if lce.Name != "" {
		val["name"] = lce.Name
	}
	if lce.OwnerID != "" {
		val["owner_id"] = lce.OwnerID
	}
	if lce.Tag != "" {
		val["tag"] = lce.Tag
	}
	if lce.Metadata != nil {
		metadata, err := json.Marshal(lce.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lce.Status.String() != "" {
		val["status"] = lce.Status.String()
	}
	if lce.Action != "" {
		val["action"] = lce.Action
	}
	if lce.Subject != "" {
		val["subject"] = lce.Subject
	}

	return val, nil
}

type listGroupMembershipEvent struct {
	mfgroups.GroupsPage
	channelID string
}

func (lcge listGroupMembershipEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  groupListMemberships,
		"total":      lcge.Total,
		"offset":     lcge.Offset,
		"limit":      lcge.Limit,
		"channel_id": lcge.channelID,
	}

	if lcge.Name != "" {
		val["name"] = lcge.Name
	}
	if lcge.OwnerID != "" {
		val["owner_id"] = lcge.OwnerID
	}
	if lcge.Tag != "" {
		val["tag"] = lcge.Tag
	}
	if lcge.Metadata != nil {
		metadata, err := json.Marshal(lcge.Metadata)
		if err != nil {
			return map[string]interface{}{}, err
		}

		val["metadata"] = metadata
	}
	if lcge.Status.String() != "" {
		val["status"] = lcge.Status.String()
	}
	if lcge.Action != "" {
		val["action"] = lcge.Action
	}
	if lcge.Subject != "" {
		val["subject"] = lcge.Subject
	}

	return val, nil
}
