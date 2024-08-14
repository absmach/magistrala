// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	clientPrefix       = "thing."
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
)

var (
	_ events.Event = (*createClientEvent)(nil)
	_ events.Event = (*updateClientEvent)(nil)
	_ events.Event = (*changeStatusClientEvent)(nil)
	_ events.Event = (*viewClientEvent)(nil)
	_ events.Event = (*viewClientPermsEvent)(nil)
	_ events.Event = (*listClientEvent)(nil)
	_ events.Event = (*identifyClientEvent)(nil)
	_ events.Event = (*authorizeClientEvent)(nil)
	_ events.Event = (*shareClientEvent)(nil)
	_ events.Event = (*removeClientEvent)(nil)
)

type createClientEvent struct {
	mgclients.Client
}

func (cce createClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  clientCreate,
		"id":         cce.ID,
		"status":     cce.Status.String(),
		"created_at": cce.CreatedAt,
	}

	if cce.Name != "" {
		val["name"] = cce.Name
	}
	if len(cce.Tags) > 0 {
		val["tags"] = cce.Tags
	}
	if cce.Domain != "" {
		val["domain"] = cce.Domain
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
	mgclients.Client
	operation string
}

func (uce updateClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  clientUpdate,
		"updated_at": uce.UpdatedAt,
		"updated_by": uce.UpdatedBy,
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
	if uce.Domain != "" {
		val["domain"] = uce.Domain
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
}

func (rce changeStatusClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  clientChangeStatus,
		"id":         rce.id,
		"status":     rce.status,
		"updated_at": rce.updatedAt,
		"updated_by": rce.updatedBy,
	}, nil
}

type viewClientEvent struct {
	mgclients.Client
}

func (vce viewClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientView,
		"id":        vce.ID,
	}

	if vce.Name != "" {
		val["name"] = vce.Name
	}
	if len(vce.Tags) > 0 {
		val["tags"] = vce.Tags
	}
	if vce.Domain != "" {
		val["domain"] = vce.Domain
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
}

func (vcpe viewClientPermsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientViewPerms,
		"permissions": vcpe.permissions,
	}
	return val, nil
}

type listClientEvent struct {
	mgclients.Page
}

func (lce listClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientList,
		"total":     lce.Total,
		"offset":    lce.Offset,
		"limit":     lce.Limit,
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
	if lce.Domain != "" {
		val["domain"] = lce.Domain
	}
	if lce.Tag != "" {
		val["tag"] = lce.Tag
	}
	if lce.Permission != "" {
		val["permission"] = lce.Permission
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

type identifyClientEvent struct {
	thingID string
}

func (ice identifyClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": clientIdentify,
		"id":        ice.thingID,
	}, nil
}

type authorizeClientEvent struct {
	thingID         string
	namespace       string
	subjectType     string
	subjectKind     string
	subjectRelation string
	subject         string
	relation        string
	permission      string
	object          string
	objectType      string
}

func (ice authorizeClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientAuthorize,
		"id":        ice.thingID,
	}
	if ice.namespace != "" {
		val["namespace"] = ice.namespace
	}
	if ice.subjectType != "" {
		val["subject_type"] = ice.subjectType
	}
	if ice.subjectKind != "" {
		val["subject_kind"] = ice.subjectKind
	}
	if ice.subjectRelation != "" {
		val["subject_relation"] = ice.subjectRelation
	}
	if ice.subject != "" {
		val["subject"] = ice.subject
	}
	if ice.relation != "" {
		val["relation"] = ice.relation
	}
	if ice.permission != "" {
		val["permission"] = ice.permission
	}
	if ice.object != "" {
		val["object"] = ice.object
	}
	if ice.objectType != "" {
		val["object_type"] = ice.objectType
	}

	return val, nil
}

type shareClientEvent struct {
	action   string
	id       string
	relation string
	userIDs  []string
}

func (sce shareClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": clientPrefix + sce.action,
		"id":        sce.id,
		"relation":  sce.relation,
		"user_ids":  sce.userIDs,
	}, nil
}

type removeClientEvent struct {
	id string
}

func (dce removeClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": clientRemove,
		"id":        dce.id,
	}, nil
}
