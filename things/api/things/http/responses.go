//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*removeRes)(nil)
	_ mainflux.Response = (*thingRes)(nil)
	_ mainflux.Response = (*viewThingRes)(nil)
	_ mainflux.Response = (*thingsPageRes)(nil)
	_ mainflux.Response = (*channelRes)(nil)
	_ mainflux.Response = (*viewChannelRes)(nil)
	_ mainflux.Response = (*channelsPageRes)(nil)
	_ mainflux.Response = (*connectionRes)(nil)
	_ mainflux.Response = (*disconnectionRes)(nil)
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

type thingRes struct {
	id      string
	created bool
}

func (res thingRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res thingRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/things/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res thingRes) Empty() bool {
	return true
}

type viewThingRes struct {
	ID       string                 `json:"id"`
	Owner    string                 `json:"-"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (res viewThingRes) Code() int {
	return http.StatusOK
}

func (res viewThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewThingRes) Empty() bool {
	return false
}

type thingsPageRes struct {
	pageRes
	Things []viewThingRes `json:"things"`
}

func (res thingsPageRes) Code() int {
	return http.StatusOK
}

func (res thingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsPageRes) Empty() bool {
	return false
}

type channelRes struct {
	id      string
	created bool
}

func (res channelRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res channelRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/channels/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res channelRes) Empty() bool {
	return true
}

type viewChannelRes struct {
	ID       string                 `json:"id"`
	Owner    string                 `json:"-"`
	Name     string                 `json:"name,omitempty"`
	Things   []viewThingRes         `json:"connected,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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

type connectionRes struct{}

func (res connectionRes) Code() int {
	return http.StatusOK
}

func (res connectionRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectionRes) Empty() bool {
	return true
}

type disconnectionRes struct{}

func (res disconnectionRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectionRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectionRes) Empty() bool {
	return true
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}
