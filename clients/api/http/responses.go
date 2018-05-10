package http

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"

	"github.com/mainflux/mainflux/clients"
)

var (
	_ mainflux.Response = (*identityRes)(nil)
	_ mainflux.Response = (*removeRes)(nil)
	_ mainflux.Response = (*clientRes)(nil)
	_ mainflux.Response = (*viewClientRes)(nil)
	_ mainflux.Response = (*listClientsRes)(nil)
	_ mainflux.Response = (*channelRes)(nil)
	_ mainflux.Response = (*viewChannelRes)(nil)
	_ mainflux.Response = (*listChannelsRes)(nil)
	_ mainflux.Response = (*connectionRes)(nil)
	_ mainflux.Response = (*disconnectionRes)(nil)
)

type identityRes struct {
	id string
}

func (res identityRes) Headers() map[string]string {
	return map[string]string{
		"X-client-id": res.id,
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

type clientRes struct {
	id      string
	created bool
}

func (res clientRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res clientRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprint("/clients/", res.id),
		}
	}

	return map[string]string{}
}

func (res clientRes) Empty() bool {
	return true
}

type viewClientRes struct {
	clients.Client
}

func (res viewClientRes) Code() int {
	return http.StatusOK
}

func (res viewClientRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewClientRes) Empty() bool {
	return false
}

type listClientsRes struct {
	Clients []clients.Client `json:"clients"`
}

func (res listClientsRes) Code() int {
	return http.StatusOK
}

func (res listClientsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listClientsRes) Empty() bool {
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
			"Location": fmt.Sprint("/channels/", res.id),
		}
	}

	return map[string]string{}
}

func (res channelRes) Empty() bool {
	return true
}

type viewChannelRes struct {
	clients.Channel
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
	Channels []clients.Channel `json:"channels"`
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
