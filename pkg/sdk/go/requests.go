// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

// updateUserSecretReq is used to update the user secret.
type updateUserSecretReq struct {
	OldSecret string `json:"old_secret,omitempty"`
	NewSecret string `json:"new_secret,omitempty"`
}

type resetPasswordRequestreq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

type resetPasswordReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	ConfPass string `json:"confirm_password"`
}

type updateThingSecretReq struct {
	Secret string `json:"secret,omitempty"`
}

// updateUserEmailReq is used to update the user email.
type updateUserEmailReq struct {
	token string
	id    string
	Email string `json:"email,omitempty"`
}

// UserPasswordReq contains old and new passwords.
type UserPasswordReq struct {
	OldPassword string `json:"old_password,omitempty"`
	Password    string `json:"password,omitempty"`
}

// Connection contains thing and channel ID that are connected.
type Connection struct {
	ThingID   string `json:"thing_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

type UsersRelationRequest struct {
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

type UserGroupsRequest struct {
	UserGroupIDs []string `json:"group_ids"`
}

type UpdateUsernameReq struct {
	id       string
	Username string `json:"username"`
}
