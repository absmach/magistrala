// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users"
)

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate()
}

type createUserReq struct {
	user  users.User
	token string
}

func (req createUserReq) validate() error {
	return req.user.Validate()
}

type viewUserReq struct {
	token  string
	userID string
}

func (req viewUserReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
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
		return errors.ErrAuthentication
	}
	return nil
}

type updateUserReq struct {
	token    string
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}
	return nil
}

type passwResetReq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

func (req passwResetReq) validate() error {
	if req.Email == "" || req.Host == "" {
		return errors.ErrMalformedEntity
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
		return errors.ErrMalformedEntity
	}
	if req.Token == "" {
		return users.ErrMissingResetToken
	}
	if req.Password != req.ConfPass {
		return errors.ErrMalformedEntity
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
		return errors.ErrAuthentication
	}
	if req.OldPassword == "" {
		return errors.ErrMalformedEntity
	}
	return nil
}

type listMemberGroupReq struct {
	token    string
	offset   uint64
	limit    uint64
	metadata users.Metadata
	groupID  string
}

func (req listMemberGroupReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthentication
	}

	if req.groupID == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}
