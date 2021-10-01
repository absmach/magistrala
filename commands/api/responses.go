// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/commands"
)

var _ mainflux.Response = (*createCommandRes)(nil)
var _ mainflux.Response = (*viewCommandRes)(nil)
var _ mainflux.Response = (*listCommandRes)(nil)
var _ mainflux.Response = (*updateCommandRes)(nil)
var _ mainflux.Response = (*removeCommandRes)(nil)

type createCommandRes struct {
	ID      string
	created bool
}

func (res createCommandRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}
func (res createCommandRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/commands/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createCommandRes) Empty() bool {
	return false
}

type viewCommandRes struct {
	ID          string            `json:"id"`
	Owner       string            `json:"-"`
	Name        string            `json:"name,omitempty"`
	Command     string            `json:"command"`
	ChannelID   string            `json:"channel_id"`
	ExecuteTime string            `json:"execution_time"`
	Metadata    commands.Metadata `json:"metadata,omitempty"`
}

func (res viewCommandRes) Code() int {
	return http.StatusOK
}

func (res viewCommandRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewCommandRes) Empty() bool {
	return false
}

type listCommandRes struct {
}

func (res listCommandRes) Code() int {
	return http.StatusOK
}

func (res listCommandRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listCommandRes) Empty() bool {
	return false
}

type updateCommandRes struct {
	Command     string            `json:"command"`
	Name        string            `json:"name,omitempty"`
	ExecuteTime time.Time         `json:"execution_time"`
	Metadata    commands.Metadata `json:"metadata,omitempty"`
}

func (res updateCommandRes) Code() int {
	return http.StatusOK
}

func (res updateCommandRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateCommandRes) Empty() bool {
	return false
}

type removeCommandRes struct{}

func (res removeCommandRes) Code() int {
	return http.StatusOK
}

func (res removeCommandRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeCommandRes) Empty() bool {
	return false
}
