// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	groups "github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/roles"
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
	rolesProvisioned []roles.RoleProvision
	authn.Session
	requestID string
}

func (cge createGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":         groupCreate,
		"id":                cge.ID,
		"roles_provisioned": cge.rolesProvisioned,
		"status":            cge.Status.String(),
		"created_at":        cge.CreatedAt,
		"domain":            cge.DomainID,
		"user_id":           cge.UserID,
		"token_type":        cge.Type.String(),
		"super_admin":       cge.SuperAdmin,
		"request_id":        cge.requestID,
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
	authn.Session
	requestID string
}

func (uge updateGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupUpdate,
		"updated_at":  uge.UpdatedAt,
		"updated_by":  uge.UpdatedBy,
		"domain":      uge.DomainID,
		"user_id":     uge.UserID,
		"token_type":  uge.Type.String(),
		"super_admin": uge.SuperAdmin,
		"request_id":  uge.requestID,
	}

	if uge.ID != "" {
		val["id"] = uge.ID
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
	authn.Session
	requestID string
}

func (rge changeStatusGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupChangeStatus,
		"id":          rge.id,
		"status":      rge.status,
		"updated_at":  rge.updatedAt,
		"updated_by":  rge.updatedBy,
		"domain":      rge.DomainID,
		"user_id":     rge.UserID,
		"token_type":  rge.Type.String(),
		"super_admin": rge.SuperAdmin,
		"request_id":  rge.requestID,
	}, nil
}

type viewGroupEvent struct {
	groups.Group
	authn.Session
	requestID string
}

func (vge viewGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupView,
		"id":          vge.ID,
		"domain":      vge.DomainID,
		"user_id":     vge.UserID,
		"token_type":  vge.Type.String(),
		"super_admin": vge.SuperAdmin,
		"request_id":  vge.requestID,
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
	domainID   string
	userID     string
	tokenType  string
	superAdmin bool
	requestID  string
}

func (lge listGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupList,
		"total":       lge.Total,
		"offset":      lge.Offset,
		"limit":       lge.Limit,
		"domain":      lge.domainID,
		"user_id":     lge.userID,
		"token_type":  lge.tokenType,
		"super_admin": lge.superAdmin,
		"request_id":  lge.requestID,
	}

	if lge.Name != "" {
		val["name"] = lge.Name
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
	userID   string
	domainID string
	groups.PageMeta
	tokenType  string
	superAdmin bool
	requestID  string
}

func (luge listUserGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupListUserGroups,
		"user_id":     luge.userID,
		"domain":      luge.domainID,
		"total":       luge.Total,
		"offset":      luge.Offset,
		"limit":       luge.Limit,
		"token_type":  luge.tokenType,
		"super_admin": luge.superAdmin,
		"request_id":  luge.requestID,
	}

	if luge.Name != "" {
		val["name"] = luge.Name
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
	authn.Session
	requestID string
}

func (rge deleteGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupRemove,
		"id":          rge.id,
		"domain":      rge.DomainID,
		"user_id":     rge.UserID,
		"token_type":  rge.Type.String(),
		"super_admin": rge.SuperAdmin,
		"request_id":  rge.requestID,
	}, nil
}

type retrieveGroupHierarchyEvent struct {
	id string
	groups.HierarchyPageMeta
	authn.Session
	requestID string
}

func (vcge retrieveGroupHierarchyEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   groupRetrieveGroupHierarchy,
		"id":          vcge.id,
		"level":       vcge.Level,
		"direction":   vcge.Direction,
		"tree":        vcge.Tree,
		"domain":      vcge.DomainID,
		"user_id":     vcge.UserID,
		"token_type":  vcge.Type.String(),
		"super_admin": vcge.SuperAdmin,
		"request_id":  vcge.requestID,
	}
	return val, nil
}

type addParentGroupEvent struct {
	id       string
	parentID string
	authn.Session
	requestID string
}

func (apge addParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupAddParentGroup,
		"id":          apge.id,
		"parent_id":   apge.parentID,
		"domain":      apge.DomainID,
		"user_id":     apge.UserID,
		"token_type":  apge.Type.String(),
		"super_admin": apge.SuperAdmin,
		"request_id":  apge.requestID,
	}, nil
}

type removeParentGroupEvent struct {
	id string
	authn.Session
	requestID string
}

func (rpge removeParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupRemoveParentGroup,
		"id":          rpge.id,
		"domain":      rpge.DomainID,
		"user_id":     rpge.UserID,
		"token_type":  rpge.Type.String(),
		"super_admin": rpge.SuperAdmin,
		"request_id":  rpge.requestID,
	}, nil
}

type viewParentGroupEvent struct {
	id        string
	domainID  string
	requestID string
}

func (vpge viewParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  groupViewParentGroup,
		"id":         vpge.id,
		"domain":     vpge.domainID,
		"request_id": vpge.requestID,
	}, nil
}

type addChildrenGroupsEvent struct {
	id          string
	childrenIDs []string
	authn.Session
	requestID string
}

func (acge addChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":    groupAddChildrenGroups,
		"id":           acge.id,
		"children_ids": acge.childrenIDs,
		"domain":       acge.DomainID,
		"user_id":      acge.UserID,
		"token_type":   acge.Type.String(),
		"super_admin":  acge.SuperAdmin,
		"request_id":   acge.requestID,
	}, nil
}

type removeChildrenGroupsEvent struct {
	id          string
	childrenIDs []string
	authn.Session
	requestID string
}

func (rcge removeChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":    groupRemoveChildrenGroups,
		"id":           rcge.id,
		"children_ids": rcge.childrenIDs,
		"domain":       rcge.DomainID,
		"user_id":      rcge.UserID,
		"token_type":   rcge.Type.String(),
		"super_admin":  rcge.SuperAdmin,
		"request_id":   rcge.requestID,
	}, nil
}

type removeAllChildrenGroupsEvent struct {
	id string
	authn.Session
	requestID string
}

func (racge removeAllChildrenGroupsEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   groupRemoveAllChildrenGroups,
		"id":          racge.id,
		"domain":      racge.DomainID,
		"user_id":     racge.UserID,
		"token_type":  racge.Type.String(),
		"super_admin": racge.SuperAdmin,
		"request_id":  racge.requestID,
	}, nil
}

type listChildrenGroupsEvent struct {
	id         string
	startLevel int64
	endLevel   int64
	groups.PageMeta
	domainID   string
	userID     string
	tokenType  string
	superAdmin bool
	requestID  string
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
		"domain":      vcge.domainID,
		"user_id":     vcge.userID,
		"token_type":  vcge.tokenType,
		"super_admin": vcge.superAdmin,
		"request_id":  vcge.requestID,
	}
	if vcge.Name != "" {
		val["name"] = vcge.Name
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
