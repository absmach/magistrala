// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/roles"
)

const (
	clientPrefix       = "client."
	clientCreate       = clientPrefix + "create"
	clientUpdate       = clientPrefix + "update"
	clientChangeStatus = clientPrefix + "change_status"
	clientRemove       = clientPrefix + "remove"
	clientView         = clientPrefix + "view"
	clientViewPerms    = clientPrefix + "view_perms"
	clientList         = clientPrefix + "list"
	clientListByGroup  = clientPrefix + "list_by_channel"
	clientIdentify     = clientPrefix + "identify"
	clientAuthorize    = clientPrefix + "authorize"
	clientSetParent    = clientPrefix + "set_parent"
	clientRemoveParent = clientPrefix + "remove_parent"
)

var (
	_ events.Event = (*createClientEvent)(nil)
	_ events.Event = (*updateClientEvent)(nil)
	_ events.Event = (*changeStatusClientEvent)(nil)
	_ events.Event = (*viewClientEvent)(nil)
	_ events.Event = (*viewClientPermsEvent)(nil)
	_ events.Event = (*listClientEvent)(nil)
	_ events.Event = (*listClientByGroupEvent)(nil)
	_ events.Event = (*identifyClientEvent)(nil)
	_ events.Event = (*authorizeClientEvent)(nil)
	_ events.Event = (*shareClientEvent)(nil)
	_ events.Event = (*removeClientEvent)(nil)
)

type createClientEvent struct {
	clients.Client
	rolesProvisioned []roles.RoleProvision
	authn.Session
	requestID string
}

func (cce createClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":         clientCreate,
		"id":                cce.ID,
		"roles_provisioned": cce.rolesProvisioned,
		"status":            cce.Status.String(),
		"created_at":        cce.CreatedAt,
		"domain":            cce.DomainID,
		"user_id":           cce.UserID,
		"token_type":        cce.Type.String(),
		"super_admin":       cce.SuperAdmin,
		"request_id":        cce.requestID,
	}

	if cce.Name != "" {
		val["name"] = cce.Name
	}
	if len(cce.Tags) > 0 {
		val["tags"] = cce.Tags
	}
	if cce.Metadata != nil {
		val["metadata"] = cce.Metadata
	}
	if cce.Credentials.Identity != "" {
		val["identity"] = cce.Credentials.Identity
	}

	return val, nil
}

type updateClientEvent struct {
	clients.Client
	operation string
	authn.Session
	requestID string
}

func (uce updateClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientUpdate,
		"updated_at":  uce.UpdatedAt,
		"updated_by":  uce.UpdatedBy,
		"domain":      uce.DomainID,
		"user_id":     uce.UserID,
		"token_type":  uce.Type.String(),
		"super_admin": uce.SuperAdmin,
		"request_id":  uce.requestID,
	}
	if uce.operation != "" {
		val["operation"] = clientUpdate + "_" + uce.operation
	}

	if uce.ID != "" {
		val["id"] = uce.ID
	}
	if uce.Name != "" {
		val["name"] = uce.Name
	}
	if len(uce.Tags) > 0 {
		val["tags"] = uce.Tags
	}
	if uce.Credentials.Identity != "" {
		val["identity"] = uce.Credentials.Identity
	}
	if uce.Metadata != nil {
		val["metadata"] = uce.Metadata
	}
	if !uce.CreatedAt.IsZero() {
		val["created_at"] = uce.CreatedAt
	}
	if uce.Status.String() != "" {
		val["status"] = uce.Status.String()
	}

	return val, nil
}

type changeStatusClientEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (rce changeStatusClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   clientChangeStatus,
		"id":          rce.id,
		"status":      rce.status,
		"updated_at":  rce.updatedAt,
		"updated_by":  rce.updatedBy,
		"domain":      rce.DomainID,
		"user_id":     rce.UserID,
		"token_type":  rce.Type.String(),
		"super_admin": rce.SuperAdmin,
		"request_id":  rce.requestID,
	}, nil
}

type viewClientEvent struct {
	clients.Client
	authn.Session
	requestID string
}

func (vce viewClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientView,
		"id":          vce.ID,
		"domain":      vce.DomainID,
		"user_id":     vce.UserID,
		"token_type":  vce.Type.String(),
		"super_admin": vce.SuperAdmin,
		"request_id":  vce.requestID,
	}

	if vce.Name != "" {
		val["name"] = vce.Name
	}
	if len(vce.Tags) > 0 {
		val["tags"] = vce.Tags
	}
	if vce.Credentials.Identity != "" {
		val["identity"] = vce.Credentials.Identity
	}
	if vce.Metadata != nil {
		val["metadata"] = vce.Metadata
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

type viewClientPermsEvent struct {
	permissions []string
	authn.Session
	requestID string
}

func (vcpe viewClientPermsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientViewPerms,
		"permissions": vcpe.permissions,
		"domain":      vcpe.DomainID,
		"user_id":     vcpe.UserID,
		"token_type":  vcpe.Type.String(),
		"super_admin": vcpe.SuperAdmin,
		"request_id":  vcpe.requestID,
	}
	return val, nil
}

type listClientEvent struct {
	clients.Page
	authn.Session
	requestID string
}

func (lce listClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientList,
		"total":       lce.Total,
		"offset":      lce.Offset,
		"limit":       lce.Limit,
		"domain":      lce.DomainID,
		"user_id":     lce.UserID,
		"token_type":  lce.Type.String(),
		"super_admin": lce.SuperAdmin,
		"request_id":  lce.requestID,
	}

	if lce.Name != "" {
		val["name"] = lce.Name
	}
	if lce.Order != "" {
		val["order"] = lce.Order
	}
	if lce.Dir != "" {
		val["dir"] = lce.Dir
	}
	if lce.Metadata != nil {
		val["metadata"] = lce.Metadata
	}
	if lce.Tag != "" {
		val["tag"] = lce.Tag
	}
	if lce.Status.String() != "" {
		val["status"] = lce.Status.String()
	}
	if len(lce.IDs) > 0 {
		val["ids"] = lce.IDs
	}
	if lce.Identity != "" {
		val["identity"] = lce.Identity
	}
	return val, nil
}

type listUserClientEvent struct {
	userID string
	clients.Page
	authn.Session
	requestID string
}

func (lce listUserClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientList,
		"req_user_id": lce.userID,
		"total":       lce.Total,
		"offset":      lce.Offset,
		"limit":       lce.Limit,
		"domain":      lce.DomainID,
		"user_id":     lce.UserID,
		"token_type":  lce.Type.String(),
		"super_admin": lce.SuperAdmin,
		"request_id":  lce.requestID,
	}

	if lce.Name != "" {
		val["name"] = lce.Name
	}
	if lce.Order != "" {
		val["order"] = lce.Order
	}
	if lce.Dir != "" {
		val["dir"] = lce.Dir
	}
	if lce.Metadata != nil {
		val["metadata"] = lce.Metadata
	}
	if lce.Tag != "" {
		val["tag"] = lce.Tag
	}
	if lce.Status.String() != "" {
		val["status"] = lce.Status.String()
	}
	if len(lce.IDs) > 0 {
		val["ids"] = lce.IDs
	}
	if lce.Identity != "" {
		val["identity"] = lce.Identity
	}

	return val, nil
}

type listClientByGroupEvent struct {
	clients.Page
	channelID string
	authn.Session
	requestID string
}

func (lcge listClientByGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientListByGroup,
		"total":       lcge.Total,
		"offset":      lcge.Offset,
		"limit":       lcge.Limit,
		"channel_id":  lcge.channelID,
		"domain":      lcge.DomainID,
		"user_id":     lcge.UserID,
		"token_type":  lcge.Type.String(),
		"super_admin": lcge.SuperAdmin,
		"request_id":  lcge.requestID,
	}

	if lcge.Name != "" {
		val["name"] = lcge.Name
	}
	if lcge.Order != "" {
		val["order"] = lcge.Order
	}
	if lcge.Dir != "" {
		val["dir"] = lcge.Dir
	}
	if lcge.Metadata != nil {
		val["metadata"] = lcge.Metadata
	}
	if lcge.Tag != "" {
		val["tag"] = lcge.Tag
	}
	if lcge.Status.String() != "" {
		val["status"] = lcge.Status.String()
	}
	if lcge.Identity != "" {
		val["identity"] = lcge.Identity
	}

	return val, nil
}

type identifyClientEvent struct {
	clientID string
	authn.Session
	requestID string
}

func (ice identifyClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   clientIdentify,
		"id":          ice.clientID,
		"domain":      ice.DomainID,
		"user_id":     ice.UserID,
		"token_type":  ice.Type.String(),
		"super_admin": ice.SuperAdmin,
		"request_id":  ice.requestID,
	}, nil
}

type authorizeClientEvent struct {
	clientID   string
	channelID  string
	permission string
	authn.Session
	requestID string
}

func (ice authorizeClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientAuthorize,
		"id":          ice.clientID,
		"domain":      ice.DomainID,
		"user_id":     ice.UserID,
		"token_type":  ice.Type.String(),
		"super_admin": ice.SuperAdmin,
		"request_id":  ice.requestID,
	}

	if ice.permission != "" {
		val["permission"] = ice.permission
	}
	if ice.channelID != "" {
		val["channelID"] = ice.channelID
	}

	return val, nil
}

type shareClientEvent struct {
	action   string
	id       string
	relation string
	userIDs  []string
	authn.Session
	requestID string
}

func (sce shareClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   clientPrefix + sce.action,
		"id":          sce.id,
		"relation":    sce.relation,
		"user_ids":    sce.userIDs,
		"domain":      sce.DomainID,
		"user_id":     sce.UserID,
		"token_type":  sce.Type.String(),
		"super_admin": sce.SuperAdmin,
		"request_id":  sce.requestID,
	}, nil
}

type removeClientEvent struct {
	id string
	authn.Session
	requestID string
}

func (dce removeClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   clientRemove,
		"id":          dce.id,
		"domain":      dce.DomainID,
		"user_id":     dce.UserID,
		"token_type":  dce.Type.String(),
		"super_admin": dce.SuperAdmin,
		"request_id":  dce.requestID,
	}, nil
}

type setParentGroupEvent struct {
	id            string
	parentGroupID string
	authn.Session
	requestID string
}

func (spge setParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":       clientSetParent,
		"id":              spge.id,
		"parent_group_id": spge.parentGroupID,
		"domain":          spge.DomainID,
		"user_id":         spge.UserID,
		"token_type":      spge.Type.String(),
		"super_admin":     spge.SuperAdmin,
		"request_id":      spge.requestID,
	}, nil
}

type removeParentGroupEvent struct {
	id string
	authn.Session
	requestID string
}

func (rpge removeParentGroupEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   clientRemoveParent,
		"id":          rpge.id,
		"domain":      rpge.DomainID,
		"user_id":     rpge.UserID,
		"token_type":  rpge.Type.String(),
		"super_admin": rpge.SuperAdmin,
		"request_id":  rpge.requestID,
	}, nil
}
