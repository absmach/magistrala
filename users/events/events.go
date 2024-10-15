// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/users"
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
	_ events.Event = (*createUserEvent)(nil)
	_ events.Event = (*updateUserEvent)(nil)
	_ events.Event = (*removeUserEvent)(nil)
	_ events.Event = (*viewUserEvent)(nil)
	_ events.Event = (*viewProfileEvent)(nil)
	_ events.Event = (*listUserEvent)(nil)
	_ events.Event = (*listUserByGroupEvent)(nil)
	_ events.Event = (*searchUserEvent)(nil)
	_ events.Event = (*identifyUserEvent)(nil)
	_ events.Event = (*generateResetTokenEvent)(nil)
	_ events.Event = (*issueTokenEvent)(nil)
	_ events.Event = (*refreshTokenEvent)(nil)
	_ events.Event = (*resetSecretEvent)(nil)
	_ events.Event = (*sendPasswordResetEvent)(nil)
	_ events.Event = (*oauthCallbackEvent)(nil)
	_ events.Event = (*deleteUserEvent)(nil)
)

type createUserEvent struct {
	users.User
}

func (uce createUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  userCreate,
		"id":         uce.ID,
		"status":     uce.Status.String(),
		"created_at": uce.CreatedAt,
	}

	if uce.FirstName != "" {
		val["first_name"] = uce.FirstName
	}
	if uce.LastName != "" {
		val["last_name"] = uce.LastName
	}
	if len(uce.Tags) > 0 {
		val["tags"] = uce.Tags
	}
	if uce.Metadata != nil {
		val["metadata"] = uce.Metadata
	}
	if uce.Credentials.UserName != "" {
		val["user_name"] = uce.Credentials.UserName
	}
	if uce.Identity != "" {
		val["identity"] = uce.Identity
	}

	return val, nil
}

type updateUserEvent struct {
	users.User
	operation string
}

func (uce updateUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  userUpdate,
		"updated_at": uce.UpdatedAt,
		"updated_by": uce.UpdatedBy,
	}
	if uce.operation != "" {
		val["operation"] = userUpdate + "_" + uce.operation
	}

	if uce.ID != "" {
		val["id"] = uce.ID
	}
	if uce.FirstName != "" {
		val["first_name"] = uce.FirstName
	}
	if uce.LastName != "" {
		val["last_name"] = uce.LastName
	}
	if len(uce.Tags) > 0 {
		val["tags"] = uce.Tags
	}
	if uce.Credentials.UserName != "" {
		val["user_name"] = uce.Credentials.UserName
	}
	if uce.Identity != "" {
		val["identity"] = uce.Identity
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

type updateUserNamesEvent struct {
	users.User
}

func (une updateUserNamesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  userUpdateUserNames,
		"updated_at": une.UpdatedAt,
		"updated_by": une.UpdatedBy,
	}

	if une.ID != "" {
		val["id"] = une.ID
	}
	if une.FirstName != "" {
		val["first_name"] = une.FirstName
	}
	if une.LastName != "" {
		val["last_name"] = une.LastName
	}
	if une.Credentials.UserName != "" {
		val["user_name"] = une.Credentials.UserName
	}

	return val, nil
}

type updateProfilePictureEvent struct {
	users.User
}

func (uppe updateProfilePictureEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  userUpdateProfilePicture,
		"updated_at": uppe.UpdatedAt,
		"updated_by": uppe.UpdatedBy,
	}

	if uppe.ID != "" {
		val["id"] = uppe.ID
	}
	if uppe.ProfilePicture != "" {
		val["profile_picture"] = uppe.ProfilePicture
	}

	return val, nil
}

type removeUserEvent struct {
	id        string
	status    string
	updatedAt time.Time
	updatedBy string
}

func (rce removeUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  userRemove,
		"id":         rce.id,
		"status":     rce.status,
		"updated_at": rce.updatedAt,
		"updated_by": rce.updatedBy,
	}, nil
}

type viewUserEvent struct {
	users.User
}

func (vue viewUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": userView,
		"id":        vue.ID,
	}

	if vue.LastName != "" {
		val["last_name"] = vue.LastName
	}
	if vue.FirstName != "" {
		val["first_name"] = vue.FirstName
	}
	if len(vue.Tags) > 0 {
		val["tags"] = vue.Tags
	}
	if vue.DomainID != "" {
		val["domain"] = vue.DomainID
	}
	if vue.Identity != "" {
		val["identity"] = vue.Identity
	}
	if vue.Credentials.UserName != "" {
		val["identity"] = vue.Credentials.UserName
	}
	if vue.Metadata != nil {
		val["metadata"] = vue.Metadata
	}
	if !vue.CreatedAt.IsZero() {
		val["created_at"] = vue.CreatedAt
	}
	if !vue.UpdatedAt.IsZero() {
		val["updated_at"] = vue.UpdatedAt
	}
	if vue.UpdatedBy != "" {
		val["updated_by"] = vue.UpdatedBy
	}
	if vue.Status.String() != "" {
		val["status"] = vue.Status.String()
	}

	return val, nil
}

type viewProfileEvent struct {
	users.User
}

func (vpe viewProfileEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": profileView,
		"id":        vpe.ID,
	}

	if vpe.FirstName != "" {
		val["first_name"] = vpe.FirstName
	}
	if len(vpe.Tags) > 0 {
		val["tags"] = vpe.Tags
	}
	if vpe.DomainID != "" {
		val["domain"] = vpe.DomainID
	}
	if vpe.Credentials.UserName != "" {
		val["user_name"] = vpe.Credentials.UserName
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
	if vpe.Identity != "" {
		val["identity"] = vpe.Identity
	}

	return val, nil
}

type viewUserByUserNameEvent struct {
	users.User
}

func (vue viewUserByUserNameEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": userView,
		"user_name": vue.Credentials.UserName,
	}

	if vue.ID != "" {
		val["id"] = vue.ID
	}

	return val, nil
}

type listUserEvent struct {
	users.Page
}

func (lue listUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": userList,
		"total":     lue.Total,
		"offset":    lue.Offset,
		"limit":     lue.Limit,
	}

	if lue.FirstName != "" {
		val["first_name"] = lue.FirstName
	}
	if lue.LastName != "" {
		val["last_name"] = lue.LastName
	}
	if lue.Order != "" {
		val["order"] = lue.Order
	}
	if lue.Dir != "" {
		val["dir"] = lue.Dir
	}
	if lue.Metadata != nil {
		val["metadata"] = lue.Metadata
	}
	if lue.Domain != "" {
		val["domain"] = lue.Domain
	}
	if lue.Tag != "" {
		val["tag"] = lue.Tag
	}
	if lue.Permission != "" {
		val["permission"] = lue.Permission
	}
	if lue.Status.String() != "" {
		val["status"] = lue.Status.String()
	}
	if lue.UserName != "" {
		val["user_name"] = lue.UserName
	}
	if lue.Identity != "" {
		val["identity"] = lue.Identity
	}

	return val, nil
}

type listUserByGroupEvent struct {
	users.Page
	objectKind string
	objectID   string
}

func (lcge listUserByGroupEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userListByGroup,
		"total":       lcge.Total,
		"offset":      lcge.Offset,
		"limit":       lcge.Limit,
		"object_kind": lcge.objectKind,
		"object_id":   lcge.objectID,
	}

	if lcge.UserName != "" {
		val["user_name"] = lcge.UserName
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
	if lcge.FirstName != "" {
		val["first_name"] = lcge.FirstName
	}
	if lcge.LastName != "" {
		val["last_name"] = lcge.LastName
	}
	if lcge.Identity != "" {
		val["identity"] = lcge.Identity
	}

	return val, nil
}

type searchUserEvent struct {
	users.Page
}

func (sce searchUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": userSearch,
		"total":     sce.Total,
		"offset":    sce.Offset,
		"limit":     sce.Limit,
	}
	if sce.UserName != "" {
		val["user_name"] = sce.UserName
	}
	if sce.FirstName != "" {
		val["first_name"] = sce.FirstName
	}
	if sce.LastName != "" {
		val["last_name"] = sce.LastName
	}
	if sce.Identity != "" {
		val["identity"] = sce.Identity
	}
	if sce.Id != "" {
		val["id"] = sce.Id
	}

	return val, nil
}

type identifyUserEvent struct {
	userID string
}

func (ise identifyUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": userIdentify,
		"id":        ise.userID,
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
	userID string
}

func (oce oauthCallbackEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": oauthCallback,
		"user_id":   oce.userID,
	}, nil
}

type deleteUserEvent struct {
	id string
}

func (dce deleteUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation": deleteUser,
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
