//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
)

var (
	_ mainflux.Response = (*identityRes)(nil)
	_ mainflux.Response = (*removeRes)(nil)
	_ mainflux.Response = (*thingRes)(nil)
	_ mainflux.Response = (*viewThingRes)(nil)
	_ mainflux.Response = (*listThingsRes)(nil)
	_ mainflux.Response = (*channelRes)(nil)
	_ mainflux.Response = (*viewChannelRes)(nil)
	_ mainflux.Response = (*listChannelsRes)(nil)
	_ mainflux.Response = (*connectionRes)(nil)
	_ mainflux.Response = (*disconnectionRes)(nil)
)

type identityRes struct {
	id uint64
}

func (res identityRes) Headers() map[string]string {
	return map[string]string{
		"X-thing-id": fmt.Sprint(res.id),
	}
}

func (res identityRes) Code() int {
	return http.StatusOK
}

func (res identityRes) Empty() bool {
	return true
}

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
	id      uint64
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
			"Location": fmt.Sprintf("/things/%d", res.id),
		}
	}

	return map[string]string{}
}

func (res thingRes) Empty() bool {
	return true
}

type viewThingRes struct {
	things.Thing
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

type listThingsRes struct {
	Things []things.Thing `json:"things"`
}

func (res listThingsRes) Code() int {
	return http.StatusOK
}

func (res listThingsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listThingsRes) Empty() bool {
	return false
}

type channelRes struct {
	id      uint64
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
			"Location": fmt.Sprintf("/channels/%d", res.id),
		}
	}

	return map[string]string{}
}

func (res channelRes) Empty() bool {
	return true
}

type viewChannelRes struct {
	things.Channel
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

type listChannelsRes struct {
	Channels []things.Channel `json:"channels"`
}

func (res listChannelsRes) Code() int {
	return http.StatusOK
}

func (res listChannelsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listChannelsRes) Empty() bool {
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
