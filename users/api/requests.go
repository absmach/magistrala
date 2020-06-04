// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/users"
)

const minPassLen = 8

type apiReq interface {
	validate() error
}

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate()
}

type viewUserReq struct {
	token string
}

func (req viewUserReq) validate() error {
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
	if len(req.Password) < minPassLen {
		return users.ErrMalformedEntity
	}
	if req.OldPassword == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}
