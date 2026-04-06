// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/roles"
)

const (
	clientPrefix       = "client."
	clientCreate       = clientPrefix + "create"
	clientUpdate       = clientPrefix + "update"
	clientUpdateTags   = clientPrefix + "update_tags"
	clientUpdateSecret = clientPrefix + "update_secret"
	clientEnable       = clientPrefix + "enable"
	clientDisable      = clientPrefix + "disable"
	clientRemove       = clientPrefix + "remove"
	clientView         = clientPrefix + "view"
	clientList         = clientPrefix + "list"
	clientListByUser   = clientPrefix + "list_by_user"
	clientSetParent    = clientPrefix + "set_parent"
	clientRemoveParent = clientPrefix + "remove_parent"
)

var (
	_ events.Event = (*createClientEvent)(nil)
	_ events.Event = (*updateClientEvent)(nil)
	_ events.Event = (*changeClientStatusEvent)(nil)
	_ events.Event = (*viewClientEvent)(nil)
	_ events.Event = (*listClientEvent)(nil)
	_ events.Event = (*listUserClientEvent)(nil)
	_ events.Event = (*removeClientEvent)(nil)
	_ events.Event = (*setParentGroupEvent)(nil)
	_ events.Event = (*removeParentGroupEvent)(nil)
)

type createClientEvent struct {
	clients.Client
	rolesProvisioned []roles.RoleProvision
	authn.Session
	requestID string
}

func (cce createClientEvent) Encode() (map[string]any, error) {
	val := map[string]any{
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
	if cce.PrivateMetadata != nil {
		val["private_metadata"] = cce.PrivateMetadata
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

func (uce updateClientEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   uce.operation,
		"updated_at":  uce.UpdatedAt,
		"updated_by":  uce.UpdatedBy,
		"domain":      uce.DomainID,
		"user_id":     uce.UserID,
		"token_type":  uce.Type.String(),
		"super_admin": uce.SuperAdmin,
		"request_id":  uce.requestID,
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
	if uce.PrivateMetadata != nil {
		val["private_metadata"] = uce.PrivateMetadata
	}
	if !uce.CreatedAt.IsZero() {
		val["created_at"] = uce.CreatedAt
	}
	if uce.Status.String() != "" {
		val["status"] = uce.Status.String()
	}

	return val, nil
}

type changeClientStatusEvent struct {
	id        string
	operation string
	status    string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (cse changeClientStatusEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   cse.operation,
		"id":          cse.id,
		"status":      cse.status,
		"updated_at":  cse.updatedAt,
		"updated_by":  cse.updatedBy,
		"domain":      cse.DomainID,
		"user_id":     cse.UserID,
		"token_type":  cse.Type.String(),
		"super_admin": cse.SuperAdmin,
		"request_id":  cse.requestID,
	}, nil
}

type viewClientEvent struct {
	clients.Client
	authn.Session
	requestID string
}

func (vce viewClientEvent) Encode() (map[string]any, error) {
	val := map[string]any{
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
	if vce.PrivateMetadata != nil {
		val["private_metadata"] = vce.PrivateMetadata
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

type listClientEvent struct {
	clients.Page
	authn.Session
	requestID string
}

func (lce listClientEvent) Encode() (map[string]any, error) {
	val := map[string]any{
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
	if len(lce.Tags.Elements) > 0 {
		val["tag"] = lce.Tags.Elements
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

func (lce listUserClientEvent) Encode() (map[string]any, error) {
	val := map[string]any{
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
	if len(lce.Tags.Elements) > 0 {
		val["tag"] = lce.Tags.Elements
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

type removeClientEvent struct {
	id string
	authn.Session
	requestID string
}

func (dce removeClientEvent) Encode() (map[string]any, error) {
	return map[string]any{
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

func (spge setParentGroupEvent) Encode() (map[string]any, error) {
	return map[string]any{
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

func (rpge removeParentGroupEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   clientRemoveParent,
		"id":          rpge.id,
		"domain":      rpge.DomainID,
		"user_id":     rpge.UserID,
		"token_type":  rpge.Type.String(),
		"super_admin": rpge.SuperAdmin,
		"request_id":  rpge.requestID,
	}, nil
}
