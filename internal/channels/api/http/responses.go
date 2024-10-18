// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/channels"
)

var (
	_ magistrala.Response = (*createChannelRes)(nil)
	_ magistrala.Response = (*viewChannelRes)(nil)
	_ magistrala.Response = (*channelsPageRes)(nil)
	_ magistrala.Response = (*updateChannelRes)(nil)
	_ magistrala.Response = (*deleteChannelRes)(nil)
	_ magistrala.Response = (*connectChannelThingRes)(nil)
	_ magistrala.Response = (*disconnectChannelThingRes)(nil)
	_ magistrala.Response = (*changeChannelStatusRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type createChannelRes struct {
	channels.Channel
	created bool
}

func (res createChannelRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createChannelRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/channels/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createChannelRes) Empty() bool {
	return false
}

type viewChannelRes struct {
	channels.Channel
}

func (res viewChannelRes) Code() int {
	return http.StatusOK
}

func (res viewChannelRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewChannelRes) Empty() bool {
	return false
}

type channelsPageRes struct {
	pageRes
	Channels []viewChannelRes `json:"channels"`
}

func (res channelsPageRes) Code() int {
	return http.StatusOK
}

func (res channelsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res channelsPageRes) Empty() bool {
	return false
}

type changeChannelStatusRes struct {
	channels.Channel
}

func (res changeChannelStatusRes) Code() int {
	return http.StatusOK
}

func (res changeChannelStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeChannelStatusRes) Empty() bool {
	return false
}

type updateChannelRes struct {
	channels.Channel
}

func (res updateChannelRes) Code() int {
	return http.StatusOK
}

func (res updateChannelRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateChannelRes) Empty() bool {
	return false
}

type deleteChannelRes struct{}

func (res deleteChannelRes) Code() int {
	return http.StatusNoContent
}

func (res deleteChannelRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteChannelRes) Empty() bool {
	return true
}

type connectChannelThingRes struct{}

func (res connectChannelThingRes) Code() int {
	return http.StatusCreated
}

func (res connectChannelThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectChannelThingRes) Empty() bool {
	return true
}

type disconnectChannelThingRes struct{}

func (res disconnectChannelThingRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectChannelThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectChannelThingRes) Empty() bool {
	return true
}
