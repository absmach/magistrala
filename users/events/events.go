// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/events"
)

const (
	clientPrefix       = "user."
	clientCreate       = clientPrefix + "create"
	clientUpdate       = clientPrefix + "update"
	clientRemove       = clientPrefix + "remove"
	clientView         = clientPrefix + "view"
	profileView        = clientPrefix + "view_profile"
	clientList         = clientPrefix + "list"
	clientSearch       = clientPrefix + "search"
	clientListByGroup  = clientPrefix + "list_by_group"
	clientIdentify     = clientPrefix + "identify"
	generateResetToken = clientPrefix + "generate_reset_token"
	issueToken         = clientPrefix + "issue_token"
	refreshToken       = clientPrefix + "refresh_token"
	resetSecret        = clientPrefix + "reset_secret"
	sendPasswordReset  = clientPrefix + "send_password_reset"
	oauthCallback      = clientPrefix + "oauth_callback"
	deleteClient       = clientPrefix + "delete"
	addClientPolicy    = clientPrefix + "add_policy"
)

var (
	_ events.Event = (*createClientEvent)(nil)
	_ events.Event = (*updateClientEvent)(nil)
	_ events.Event = (*removeClientEvent)(nil)
	_ events.Event = (*viewClientEvent)(nil)
	_ events.Event = (*viewProfileEvent)(nil)
	_ events.Event = (*listClientEvent)(nil)
	_ events.Event = (*listClientByGroupEvent)(nil)
	_ events.Event = (*searchClientEvent)(nil)
	_ events.Event = (*identifyClientEvent)(nil)
	_ events.Event = (*generateResetTokenEvent)(nil)
	_ events.Event = (*issueTokenEvent)(nil)
	_ events.Event = (*refreshTokenEvent)(nil)
	_ events.Event = (*resetSecretEvent)(nil)
	_ events.Event = (*sendPasswordResetEvent)(nil)
	_ events.Event = (*oauthCallbackEvent)(nil)
	_ events.Event = (*deleteClientEvent)(nil)
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

type removeClientEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rce removeClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  clientRemove,
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

type viewProfileEvent struct {
	mgclients.Client
}

func (vpe viewProfileEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": profileView,
		"id":        vpe.ID,
	}

	if vpe.Name != "" {
		val["name"] = vpe.Name
	}
	if len(vpe.Tags) > 0 {
		val["tags"] = vpe.Tags
	}
	if vpe.Domain != "" {
		val["domain"] = vpe.Domain
	}
	if vpe.Credentials.Identity != "" {
		val["identity"] = vpe.Credentials.Identity
	}
	if vpe.Metadata != nil {
		val["metadata"] = vpe.Metadata
	}
	if !vpe.CreatedAt.IsZero() {
		val["created_at"] = vpe.CreatedAt
	}
	if !vpe.UpdatedAt.IsZero() {
		val["updated_at"] = vpe.UpdatedAt
	}
	if vpe.UpdatedBy != "" {
		val["updated_by"] = vpe.UpdatedBy
	}
	if vpe.Status.String() != "" {
		val["status"] = vpe.Status.String()
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
	if lce.Identity != "" {
		val["identity"] = lce.Identity
	}

	return val, nil
}

type listClientByGroupEvent struct {
	mgclients.Page
	objectKind string
	objectID   string
}

func (lcge listClientByGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   clientListByGroup,
		"total":       lcge.Total,
		"offset":      lcge.Offset,
		"limit":       lcge.Limit,
		"object_kind": lcge.objectKind,
		"object_id":   lcge.objectID,
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
	if lcge.Domain != "" {
		val["domain"] = lcge.Domain
	}
	if lcge.Tag != "" {
		val["tag"] = lcge.Tag
	}
	if lcge.Permission != "" {
		val["permission"] = lcge.Permission
	}
	if lcge.Status.String() != "" {
		val["status"] = lcge.Status.String()
	}
	if lcge.Identity != "" {
		val["identity"] = lcge.Identity
	}

	return val, nil
}

type searchClientEvent struct {
	mgclients.Page
}

func (sce searchClientEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": clientSearch,
		"total":     sce.Total,
		"offset":    sce.Offset,
		"limit":     sce.Limit,
	}
	if sce.Name != "" {
		val["name"] = sce.Name
	}
	if sce.Identity != "" {
		val["identity"] = sce.Identity
	}
	if sce.Id != "" {
		val["id"] = sce.Id
	}

	return val, nil
}

type identifyClientEvent struct {
	userID string
}

func (ice identifyClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": clientIdentify,
		"id":        ice.userID,
	}, nil
}

type generateResetTokenEvent struct {
	email string
	host  string
}

func (grte generateResetTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": generateResetToken,
		"email":     grte.email,
		"host":      grte.host,
	}, nil
}

type issueTokenEvent struct {
	identity string
	domainID string
}

func (ite issueTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": issueToken,
		"identity":  ite.identity,
		"domain_id": ite.domainID,
	}, nil
}

type refreshTokenEvent struct {
	domainID string
}

func (rte refreshTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": refreshToken,
		"domain_id": rte.domainID,
	}, nil
}

type resetSecretEvent struct{}

func (rse resetSecretEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": resetSecret,
	}, nil
}

type sendPasswordResetEvent struct {
	host  string
	email string
	user  string
}

func (spre sendPasswordResetEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": sendPasswordReset,
		"host":      spre.host,
		"email":     spre.email,
		"user":      spre.user,
	}, nil
}

type oauthCallbackEvent struct {
	clientID string
}

func (oce oauthCallbackEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": oauthCallback,
		"client_id": oce.clientID,
	}, nil
}

type deleteClientEvent struct {
	id string
}

func (dce deleteClientEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": deleteClient,
		"id":        dce.id,
	}, nil
}

type addClientPolicyEvent struct {
	id   string
	role string
}

func (acpe addClientPolicyEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": addClientPolicy,
		"id":        acpe.id,
		"role":      acpe.role,
	}, nil
}
