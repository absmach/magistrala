// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/users"
)

type apiReq interface {
	validate() error
}

type userReq struct {
	user users.User
}

func (req userReq) validate() errors.Error {
	return req.user.Validate()
}

type viewUserInfoReq struct {
	token string
}

func (req viewUserInfoReq) validate() errors.Error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type updateUserReq struct {
	token    string
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() errors.Error {
	if req.token == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}

type passwResetReq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

func (req passwResetReq) validate() errors.Error {
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

func (req resetTokenReq) validate() errors.Error {
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

func (req passwChangeReq) validate() errors.Error {
	if req.Token == "" {
		return users.ErrUnauthorizedAccess
	}
	if req.Password == "" {
		return users.ErrMalformedEntity
	}
	if req.OldPassword == "" {
		return users.ErrUnauthorizedAccess
	}
	return nil
}
