// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
)

var (
	_ magistrala.Response = (*removeRes)(nil)
	_ magistrala.Response = (*configRes)(nil)
	_ magistrala.Response = (*stateRes)(nil)
	_ magistrala.Response = (*viewRes)(nil)
	_ magistrala.Response = (*listRes)(nil)
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
			"Location": fmt.Sprintf("/clients/configs/%s", res.id),
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
	ClientID     string          `json:"client_id,omitempty"`
	CLientSecret string          `json:"client,_secretomitempty"`
	Channels     []channelRes    `json:"channels,omitempty"`
	ExternalID   string          `json:"external_id"`
	ExternalKey  string          `json:"external_key,omitempty"`
	Content      string          `json:"content,omitempty"`
	Name         string          `json:"name,omitempty"`
	State        bootstrap.State `json:"state"`
	ClientCert   string          `json:"client_cert,omitempty"`
	CACert       string          `json:"ca_cert,omitempty"`
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

type updateConfigRes struct {
	ClientID   string `json:"client_id,omitempty"`
	CACert     string `json:"ca_cert,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
}

func (res updateConfigRes) Code() int {
	return http.StatusOK
}

func (res updateConfigRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateConfigRes) Empty() bool {
	return false
}
