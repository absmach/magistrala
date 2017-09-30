package api

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux/manager"
)

const contentType = "application/json; charset=utf-8"

type apiRes interface {
	code() int
	headers() map[string]string
	empty() bool
}

type identityRes struct {
	id string
}

func (res identityRes) headers() map[string]string {
	return map[string]string{
		"X-Client-Id": res.id,
	}
}

func (res identityRes) code() int {
	return http.StatusOK
}

func (res identityRes) empty() bool {
	return true
}

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

func (res tokenRes) code() int {
	return http.StatusCreated
}

func (res tokenRes) headers() map[string]string {
	return map[string]string{}
}

func (res tokenRes) empty() bool {
	return res.Token == ""
}

type removeRes struct{}

func (res removeRes) code() int {
	return http.StatusNoContent
}

func (res removeRes) headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) empty() bool {
	return true
}

type clientRes struct {
	id      string
	created bool
}

func (res clientRes) code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res clientRes) headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprint("/clients/", res.id),
		}
	}

	return map[string]string{}
}

func (res clientRes) empty() bool {
	return true
}

type viewClientRes struct {
	manager.Client
}

func (res viewClientRes) code() int {
	return http.StatusOK
}

func (res viewClientRes) headers() map[string]string {
	return map[string]string{}
}

func (res viewClientRes) empty() bool {
	return false
}

type listClientsRes struct {
	Clients []manager.Client `json:"clients"`
	count   int
}

func (res listClientsRes) code() int {
	return http.StatusOK
}

func (res listClientsRes) headers() map[string]string {
	return map[string]string{
		"X-Count": fmt.Sprintf("%d", res.count),
	}
}

func (res listClientsRes) empty() bool {
	return false
}

type channelRes struct {
	id      string
	created bool
}

func (res channelRes) code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res channelRes) headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprint("/channels/", res.id),
		}
	}

	return map[string]string{}
}

func (res channelRes) empty() bool {
	return true
}

type viewChannelRes struct {
	manager.Channel
}

func (res viewChannelRes) code() int {
	return http.StatusOK
}

func (res viewChannelRes) headers() map[string]string {
	return map[string]string{}
}

func (res viewChannelRes) empty() bool {
	return false
}

type listChannelsRes struct {
	Channels []manager.Channel `json:"channels"`
	count    int
}

func (res listChannelsRes) code() int {
	return http.StatusOK
}

func (res listChannelsRes) headers() map[string]string {
	return map[string]string{
		"X-Count": fmt.Sprintf("%d", res.count),
	}
}

func (res listChannelsRes) empty() bool {
	return false
}

type accessRes struct{}

func (res accessRes) code() int {
	return http.StatusAccepted
}

func (res accessRes) headers() map[string]string {
	return map[string]string{}
}

func (res accessRes) empty() bool {
	return true
}
