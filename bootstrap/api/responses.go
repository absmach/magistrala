// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
)

var (
	_ mainflux.Response = (*removeRes)(nil)
	_ mainflux.Response = (*configRes)(nil)
	_ mainflux.Response = (*stateRes)(nil)
	_ mainflux.Response = (*viewRes)(nil)
	_ mainflux.Response = (*listRes)(nil)
)

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}

type configRes struct {
	id      string
	created bool
}

func (res configRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res configRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/things/configs/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res configRes) Empty() bool {
	return true
}

type channelRes struct {
	ID       string      `json:"id"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type viewRes struct {
	MFThing     string          `json:"mainflux_id,omitempty"`
	MFKey       string          `json:"mainflux_key,omitempty"`
	Channels    []channelRes    `json:"mainflux_channels,omitempty"`
	ExternalID  string          `json:"external_id"`
	ExternalKey string          `json:"external_key,omitempty"`
	Content     string          `json:"content,omitempty"`
	Name        string          `json:"name,omitempty"`
	State       bootstrap.State `json:"state"`
}

func (res viewRes) Code() int {
	return http.StatusOK
}

func (res viewRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewRes) Empty() bool {
	return false
}

type listRes struct {
	Total   uint64    `json:"total"`
	Offset  uint64    `json:"offset"`
	Limit   uint64    `json:"limit"`
	Configs []viewRes `json:"configs"`
}

func (res listRes) Code() int {
	return http.StatusOK
}

func (res listRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRes) Empty() bool {
	return false
}

type stateRes struct{}

func (res stateRes) Code() int {
	return http.StatusOK
}

func (res stateRes) Headers() map[string]string {
	return map[string]string{}
}

func (res stateRes) Empty() bool {
	return true
}
