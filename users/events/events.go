// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/users"
)

const (
	userPrefix               = "user."
	userCreate               = userPrefix + "create"
	userUpdate               = userPrefix + "update"
	userRemove               = userPrefix + "remove"
	userView                 = userPrefix + "view"
	profileView              = userPrefix + "view_profile"
	userList                 = userPrefix + "list"
	userSearch               = userPrefix + "search"
	userListByGroup          = userPrefix + "list_by_group"
	userIdentify             = userPrefix + "identify"
	generateResetToken       = userPrefix + "generate_reset_token"
	issueToken               = userPrefix + "issue_token"
	refreshToken             = userPrefix + "refresh_token"
	resetSecret              = userPrefix + "reset_secret"
	sendPasswordReset        = userPrefix + "send_password_reset"
	oauthCallback            = userPrefix + "oauth_callback"
	addClientPolicy          = userPrefix + "add_policy"
	deleteUser               = userPrefix + "delete"
	userUpdateUsername       = userPrefix + "update_username"
	userUpdateProfilePicture = userPrefix + "update_profile_picture"
)

var (
	_ events.Event = (*createUserEvent)(nil)
	_ events.Event = (*updateUserEvent)(nil)
	_ events.Event = (*updateProfilePictureEvent)(nil)
	_ events.Event = (*updateUsernameEvent)(nil)
	_ events.Event = (*removeUserEvent)(nil)
	_ events.Event = (*viewUserEvent)(nil)
	_ events.Event = (*viewProfileEvent)(nil)
	_ events.Event = (*listUserEvent)(nil)
	_ events.Event = (*searchUserEvent)(nil)
	_ events.Event = (*identifyUserEvent)(nil)
	_ events.Event = (*generateResetTokenEvent)(nil)
	_ events.Event = (*issueTokenEvent)(nil)
	_ events.Event = (*refreshTokenEvent)(nil)
	_ events.Event = (*resetSecretEvent)(nil)
	_ events.Event = (*sendPasswordResetEvent)(nil)
	_ events.Event = (*oauthCallbackEvent)(nil)
	_ events.Event = (*deleteUserEvent)(nil)
	_ events.Event = (*addUserPolicyEvent)(nil)
)

type createUserEvent struct {
	users.User
	authn.Session
	requestID string
}

func (uce createUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userCreate,
		"id":          uce.ID,
		"status":      uce.Status.String(),
		"created_at":  uce.CreatedAt,
		"token_type":  uce.Type.String(),
		"super_admin": uce.SuperAdmin,
		"request_id":  uce.requestID,
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
	if uce.Credentials.Username != "" {
		val["username"] = uce.Credentials.Username
	}
	if uce.Email != "" {
		val["email"] = uce.Email
	}

	return val, nil
}

type updateUserEvent struct {
	users.User
	operation string
	authn.Session
	requestID string
}

func (uce updateUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userUpdate,
		"updated_at":  uce.UpdatedAt,
		"updated_by":  uce.UpdatedBy,
		"token_type":  uce.Type.String(),
		"super_admin": uce.SuperAdmin,
		"request_id":  uce.requestID,
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
	if uce.Credentials.Username != "" {
		val["username"] = uce.Credentials.Username
	}
	if uce.Email != "" {
		val["email"] = uce.Email
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

type updateUsernameEvent struct {
	users.User
	authn.Session
	requestID string
}

func (une updateUsernameEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userUpdateUsername,
		"updated_at":  une.UpdatedAt,
		"updated_by":  une.UpdatedBy,
		"token_type":  une.Type.String(),
		"super_admin": une.SuperAdmin,
		"request_id":  une.requestID,
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
	if une.Credentials.Username != "" {
		val["username"] = une.Credentials.Username
	}

	return val, nil
}

type updateProfilePictureEvent struct {
	users.User
	authn.Session
	requestID string
}

func (uppe updateProfilePictureEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userUpdateProfilePicture,
		"updated_at":  uppe.UpdatedAt,
		"updated_by":  uppe.UpdatedBy,
		"token_type":  uppe.Type.String(),
		"super_admin": uppe.SuperAdmin,
		"request_id":  uppe.requestID,
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
	authn.Session
	requestID string
}

func (rce removeUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   userRemove,
		"id":          rce.id,
		"status":      rce.status,
		"updated_at":  rce.updatedAt,
		"updated_by":  rce.updatedBy,
		"token_type":  rce.Type.String(),
		"super_admin": rce.SuperAdmin,
		"request_id":  rce.requestID,
	}, nil
}

type viewUserEvent struct {
	users.User
	authn.Session
	requestID string
}

func (vue viewUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userView,
		"id":          vue.ID,
		"token_type":  vue.Type.String(),
		"super_admin": vue.SuperAdmin,
		"request_id":  vue.requestID,
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
	if vue.Email != "" {
		val["email"] = vue.Email
	}
	if vue.Credentials.Username != "" {
		val["email"] = vue.Credentials.Username
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
	authn.Session
	requestID string
}

func (vpe viewProfileEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   profileView,
		"id":          vpe.ID,
		"token_type":  vpe.Type.String(),
		"super_admin": vpe.SuperAdmin,
		"request_id":  vpe.requestID,
	}

	if vpe.FirstName != "" {
		val["first_name"] = vpe.FirstName
	}
	if len(vpe.Tags) > 0 {
		val["tags"] = vpe.Tags
	}
	if vpe.Credentials.Username != "" {
		val["username"] = vpe.Credentials.Username
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
	if vpe.Email != "" {
		val["email"] = vpe.Email
	}

	return val, nil
}

type listUserEvent struct {
	users.Page
	authn.Session
	requestID string
}

func (lue listUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":   userList,
		"total":       lue.Total,
		"offset":      lue.Offset,
		"limit":       lue.Limit,
		"token_type":  lue.Type.String(),
		"super_admin": lue.SuperAdmin,
		"request_id":  lue.requestID,
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
	if lue.Username != "" {
		val["username"] = lue.Username
	}
	if lue.Email != "" {
		val["email"] = lue.Email
	}

	return val, nil
}

type searchUserEvent struct {
	users.Page
	requestID string
}

func (sce searchUserEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  userSearch,
		"total":      sce.Total,
		"offset":     sce.Offset,
		"limit":      sce.Limit,
		"request_id": sce.requestID,
	}
	if sce.Username != "" {
		val["username"] = sce.Username
	}
	if sce.FirstName != "" {
		val["first_name"] = sce.FirstName
	}
	if sce.LastName != "" {
		val["last_name"] = sce.LastName
	}
	if sce.Email != "" {
		val["email"] = sce.Email
	}
	if sce.Id != "" {
		val["id"] = sce.Id
	}

	return val, nil
}

type identifyUserEvent struct {
	userID    string
	requestID string
}

func (ise identifyUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  userIdentify,
		"id":         ise.userID,
		"request_id": ise.requestID,
	}, nil
}

type generateResetTokenEvent struct {
	email     string
	host      string
	requestID string
}

func (grte generateResetTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  generateResetToken,
		"email":      grte.email,
		"host":       grte.host,
		"request_id": grte.requestID,
	}, nil
}

type issueTokenEvent struct {
	username  string
	requestID string
}

func (ite issueTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  issueToken,
		"username":   ite.username,
		"request_id": ite.requestID,
	}, nil
}

type refreshTokenEvent struct {
	requestID string
}

func (rte refreshTokenEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  refreshToken,
		"request_id": rte.requestID,
	}, nil
}

type resetSecretEvent struct {
	requestID string
}

func (rse resetSecretEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  resetSecret,
		"request_id": rse.requestID,
	}, nil
}

type sendPasswordResetEvent struct {
	host      string
	email     string
	user      string
	requestID string
}

func (spre sendPasswordResetEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  sendPasswordReset,
		"host":       spre.host,
		"email":      spre.email,
		"user":       spre.user,
		"request_id": spre.requestID,
	}, nil
}

type oauthCallbackEvent struct {
	userID    string
	requestID string
}

func (oce oauthCallbackEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":  oauthCallback,
		"user_id":    oce.userID,
		"request_id": oce.requestID,
	}, nil
}

type deleteUserEvent struct {
	id string
	authn.Session
	requestID string
}

func (dce deleteUserEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   deleteUser,
		"id":          dce.id,
		"token_type":  dce.Type.String(),
		"super_admin": dce.SuperAdmin,
		"request_id":  dce.requestID,
	}, nil
}

type addUserPolicyEvent struct {
	id   string
	role string
	authn.Session
	requestID string
}

func (acpe addUserPolicyEvent) Encode() (map[string]interface{}, error) {
	return map[string]interface{}{
		"operation":   addClientPolicy,
		"id":          acpe.id,
		"role":        acpe.role,
		"token_type":  acpe.Type.String(),
		"super_admin": acpe.SuperAdmin,
		"request_id":  acpe.requestID,
	}, nil
}
