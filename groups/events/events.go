// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	groups "github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/pkg/events"
)

var (
	groupPrefix                  = "group."
	groupCreate                  = groupPrefix + "create"
	groupUpdate                  = groupPrefix + "update"
	groupChangeStatus            = groupPrefix + "change_status"
	groupView                    = groupPrefix + "view"
	groupList                    = groupPrefix + "list"
	groupListUserGroups          = groupPrefix + "list_user_groups"
	groupRemove                  = groupPrefix + "remove"
	groupRetrieveGroupHierarchy  = groupPrefix + "retrieve_group_hierarchy"
	groupAddParentGroup          = groupPrefix + "add_parent_group"
	groupRemoveParentGroup       = groupPrefix + "remove_parent_group"
	groupViewParentGroup         = groupPrefix + "view_parent_group"
	groupAddChildrenGroups       = groupPrefix + "add_children_groups"
	groupRemoveChildrenGroups    = groupPrefix + "remove_children_groups"
	groupRemoveAllChildrenGroups = groupPrefix + "remove_all_children_groups"
	groupListChildrenGroups      = groupPrefix + "list_children_groups"
)

var (
	_ events.Event = (*createGroupEvent)(nil)
	_ events.Event = (*updateGroupEvent)(nil)
	_ events.Event = (*changeStatusGroupEvent)(nil)
	_ events.Event = (*viewGroupEvent)(nil)
	_ events.Event = (*deleteGroupEvent)(nil)
	_ events.Event = (*viewGroupEvent)(nil)
	_ events.Event = (*listGroupEvent)(nil)
	_ events.Event = (*addParentGroupEvent)(nil)
	_ events.Event = (*removeParentGroupEvent)(nil)
	_ events.Event = (*viewParentGroupEvent)(nil)
	_ events.Event = (*addChildrenGroupsEvent)(nil)
	_ events.Event = (*removeChildrenGroupsEvent)(nil)
	_ events.Event = (*removeAllChildrenGroupsEvent)(nil)
	_ events.Event = (*listChildrenGroupsEvent)(nil)
	_ events.Event = (*retrieveGroupHierarchyEvent)(nil)
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

	if cge.Domain != "" {
		val["domain"] = cge.Domain
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
		val["metadata"] = cge.Metadata
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
	if uge.Domain != "" {
		val["domain"] = uge.Domain
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
		val["metadata"] = uge.Metadata
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

	if vge.Domain != "" {
		val["domain"] = vge.Domain
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
		val["metadata"] = vge.Metadata
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
	groups.PageMeta
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
	if lge.DomainID != "" {
		val["domain_id"] = lge.DomainID
	}
	if lge.Tag != "" {
		val["tag"] = lge.Tag
	}
	if lge.Metadata != nil {
		val["metadata"] = lge.Metadata
	}
	if lge.Status.String() != "" {
		val["status"] = lge.Status.String()
	}

	return val, nil
}

type listUserGroupEvent struct {
	userID string
	groups.PageMeta
}

func (luge listUserGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupListUserGroups,
		"user_id":   luge.userID,
		"total":     luge.Total,
		"offset":    luge.Offset,
		"limit":     luge.Limit,
	}

	if luge.Name != "" {
		val["name"] = luge.Name
	}
	if luge.DomainID != "" {
		val["domain_id"] = luge.DomainID
	}
	if luge.Tag != "" {
		val["tag"] = luge.Tag
	}
	if luge.Metadata != nil {
		val["metadata"] = luge.Metadata
	}
	if luge.Status.String() != "" {
		val["status"] = luge.Status.String()
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

type retrieveGroupHierarchyEvent struct {
	id string
	groups.HierarchyPageMeta
}

func (vcge retrieveGroupHierarchyEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": groupRetrieveGroupHierarchy,
		"id":        vcge.id,
		"level":     vcge.Level,
		"direction": vcge.Direction,
		"tree":      vcge.Tree,
	}
	return val, nil
}

type addParentGroupEvent struct {
	id       string
	parentID string
}

func (apge addParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": groupAddParentGroup,
		"id":        apge.id,
		"parent_id": apge.parentID,
	}, nil
}

type removeParentGroupEvent struct {
	id string
}

func (rpge removeParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": groupRemoveParentGroup,
		"id":        rpge.id,
	}, nil
}

type viewParentGroupEvent struct {
	id string
}

func (vpge viewParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": groupViewParentGroup,
		"id":        vpge.id,
	}, nil
}

type addChildrenGroupsEvent struct {
	id          string
	childrenIDs []string
}

func (acge addChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupAddChildrenGroups,
		"id":          acge.id,
		"childre_ids": acge.childrenIDs,
	}, nil
}

type removeChildrenGroupsEvent struct {
	id          string
	childrenIDs []string
}

func (rcge removeChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":    groupRemoveChildrenGroups,
		"id":           rcge.id,
		"children_ids": rcge.childrenIDs,
	}, nil
}

type removeAllChildrenGroupsEvent struct {
	id string
}

func (racge removeAllChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": groupRemoveAllChildrenGroups,
		"id":        racge.id,
	}, nil
}

type listChildrenGroupsEvent struct {
	id         string
	startLevel int64
	endLevel   int64
	groups.PageMeta
}

func (vcge listChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupListChildrenGroups,
		"id":          vcge.id,
		"start_level": vcge.startLevel,
		"end_level":   vcge.endLevel,
		"total":       vcge.Total,
		"offset":      vcge.Offset,
		"limit":       vcge.Limit,
	}
	if vcge.Name != "" {
		val["name"] = vcge.Name
	}
	if vcge.DomainID != "" {
		val["domain_id"] = vcge.DomainID
	}
	if vcge.Tag != "" {
		val["tag"] = vcge.Tag
	}
	if vcge.Metadata != nil {
		val["metadata"] = vcge.Metadata
	}
	if vcge.Status.String() != "" {
		val["status"] = vcge.Status.String()
	}
	return val, nil
}
