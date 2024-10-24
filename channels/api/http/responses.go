// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/channels"
)

var (
	_ magistrala.Response = (*createChannelRes)(nil)
	_ magistrala.Response = (*viewChannelRes)(nil)
	_ magistrala.Response = (*channelsPageRes)(nil)
	_ magistrala.Response = (*updateChannelRes)(nil)
	_ magistrala.Response = (*deleteChannelRes)(nil)
	_ magistrala.Response = (*connectChannelClientsRes)(nil)
	_ magistrala.Response = (*disconnectChannelClientsRes)(nil)
	_ magistrala.Response = (*connectRes)(nil)
	_ magistrala.Response = (*disconnectRes)(nil)
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

type setChannelParentGroupRes struct{}

func (res setChannelParentGroupRes) Code() int {
	return http.StatusAccepted
}

func (res setChannelParentGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res setChannelParentGroupRes) Empty() bool {
	return true
}

type removeChannelParentGroupRes struct{}

func (res removeChannelParentGroupRes) Code() int {
	return http.StatusNoContent
}

func (res removeChannelParentGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeChannelParentGroupRes) Empty() bool {
	return true
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

type connectChannelClientsRes struct{}

func (res connectChannelClientsRes) Code() int {
	return http.StatusCreated
}

func (res connectChannelClientsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectChannelClientsRes) Empty() bool {
	return true
}

type disconnectChannelClientsRes struct{}

func (res disconnectChannelClientsRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectChannelClientsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectChannelClientsRes) Empty() bool {
	return true
}

type connectRes struct{}

func (res connectRes) Code() int {
	return http.StatusCreated
}

func (res connectRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectRes) Empty() bool {
	return true
}

type disconnectRes struct{}

func (res disconnectRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectRes) Empty() bool {
	return true
}
