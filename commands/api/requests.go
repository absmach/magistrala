// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/commands"
)

type apiReq interface {
	validate() error
}

type createCommandReq struct {
	token       string
	Command     string `json:"command"`
	Name        string `josn:"name"`
	ChannelID   string `json:"channel_id"`
	ExecuteTime string `json:"execute_time"`
}

func (req createCommandReq) validate() error {
	if req.Command == "" {
		return commands.ErrMalformedEntity
	}
	return nil
}

type viewCommandReq struct {
	token string
	id    string
}

func (req viewCommandReq) validate() error {
	if req.token == "" {
		return commands.ErrMalformedEntity
	}
	return nil
}

type listCommandReq struct {
	Secret string `json:"secret"`
}

func (req listCommandReq) validate() error {
	if req.Secret == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}

type updateCommandReq struct {
	token       string
	id          string
	Command     string                 `json:"command"`
	Name        string                 `josn:"name"`
	ExecuteTime string                 `json:"execute_time"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateCommandReq) validate() error {
	if req.id == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}

type removeCommandReq struct {
	token string
	id    string
}

func (req removeCommandReq) validate() error {
	if req.id == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}
