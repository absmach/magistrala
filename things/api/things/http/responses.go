// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	created  bool
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
			"Location":           fmt.Sprintf("/things/%s", res.ID),
			"Warning-Deprecated": "This endpoint will be depreciated in v1.0.0. It will be replaced with the bulk endpoint currently found at /things/bulk.",
		}
	}

	return map[string]string{}
}

func (res thingRes) Empty() bool {
	return true
}

type thingsRes struct {
	Things  []thingRes `json:"things"`
	created bool
}

func (res thingsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res thingsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsRes) Empty() bool {
	return false
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
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	created  bool
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
			"Location":           fmt.Sprintf("/channels/%s", res.ID),
			"Warning-Deprecated": "This endpoint will be depreciated in v1.0.0. It will be replaced with the bulk endpoint currently found at /channels/bulk.",
		}
	}

	return map[string]string{}
}

func (res channelRes) Empty() bool {
	return true
}

type channelsRes struct {
	Channels []channelRes `json:"channels"`
	created  bool
}

func (res channelsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res channelsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res channelsRes) Empty() bool {
	return false
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
	return map[string]string{
		"Warning-Deprecated": "This endpoint will be depreciated in v1.0.0. It will be replaced with the bulk endpoint found at /connect.",
	}
}

func (res connectionRes) Empty() bool {
	return true
}

type createConnectionsRes struct{}

func (res createConnectionsRes) Code() int {
	return http.StatusOK
}

func (res createConnectionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createConnectionsRes) Empty() bool {
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
	Order  string `json:"order"`
	Dir    string `json:"direction"`
}

type errorRes struct {
	Err string `json:"error"`
}
