// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/users"
)

const (
	maxNameSize = 1024
)

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate()
}

type viewUserReq struct {
	token  string
	userID string
}

func (req viewUserReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type listUsersReq struct {
	token    string
	offset   uint64
	limit    uint64
	email    string
	metadata users.Metadata
}

func (req listUsersReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type updateUserReq struct {
	token    string
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type passwResetReq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

func (req passwResetReq) validate() error {
	if req.Email == "" || req.Host == "" {
		return users.ErrMalformedEntity
	}
	return nil
}

type resetTokenReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	ConfPass string `json:"confirm_password"`
}

func (req resetTokenReq) validate() error {
	if req.Password == "" || req.ConfPass == "" {
		return users.ErrMalformedEntity
	}
	if req.Token == "" {
		return users.ErrMissingResetToken
	}
	if req.Password != req.ConfPass {
		return users.ErrMalformedEntity
	}
	return nil
}

type passwChangeReq struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	OldPassword string `json:"old_password"`
}

func (req passwChangeReq) validate() error {
	if req.Token == "" {
		return users.ErrUnauthorizedAccess
	}
	if req.OldPassword == "" {
		return users.ErrMalformedEntity
	}
	return nil
}

type createGroupReq struct {
	token       string
	Name        string                 `json:"name,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req createGroupReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	if len(req.Name) > maxNameSize || req.Name == "" {
		return users.ErrMalformedEntity
	}
	return nil
}

type updateGroupReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	if req.id == "" {
		return users.ErrMalformedEntity
	}
	if req.Name == "" || len(req.Name) > maxNameSize {
		return users.ErrMalformedEntity
	}

	return nil
}

type listUserGroupReq struct {
	token    string
	offset   uint64
	limit    uint64
	metadata users.Metadata
	name     string
	groupID  string
	userID   string
}

func (req listUserGroupReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type userGroupReq struct {
	token   string
	groupID string
	userID  string
}

func (req userGroupReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	if req.groupID == "" {
		return users.ErrMalformedEntity
	}
	if req.userID == "" {
		return users.ErrMalformedEntity
	}
	return nil
}

type groupReq struct {
	token   string
	groupID string
	name    string
}

func (req groupReq) validate() error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	if req.groupID == "" && req.name == "" {
		return users.ErrMalformedEntity
	}
	return nil
}
