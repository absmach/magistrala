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

type updateClientSecretReq struct {
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

// Connection contains clients and channel IDs that are connected.
type Connection struct {
	ClientIDs  []string `json:"client_ids,omitempty"`
	ChannelIDs []string `json:"channel_ids,omitempty"`
	Types      []string `json:"types,omitempty"`
}

type UsersRelationRequest struct {
	Relation string   `json:"relation"`
	UserIDs  []string `json:"user_ids"`
}

type UserGroupsRequest struct {
	UserGroupIDs []string `json:"group_ids"`
}

type RoleReq struct {
	RoleName        string   `json:"role_name"`
	OptionalActions []string `json:"optional_actions"`
	OptionalMembers []string `json:"optional_members"`
}

type UpdateUsernameReq struct {
	id       string
	Username string `json:"username"`
}

type parentGroupReq struct {
	ParentGroupID string `json:"parent_group_id"`
}

type updateRoleNameReq struct {
	Name string `json:"name"`
}
type roleActionsReq struct {
	Actions []string `json:"actions"`
}

type roleMembersReq struct {
	Members []string `json:"members"`
}

type groupParentReq struct {
	ParentID string `json:"parent_id"`
}

type childrenGroupsReq struct {
	ChildrenIDs []string `json:"children_ids"`
}
