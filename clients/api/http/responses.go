// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/clients"
)

var (
	_ magistrala.Response = (*createClientRes)(nil)
	_ magistrala.Response = (*viewClientRes)(nil)
	_ magistrala.Response = (*viewClientPermsRes)(nil)
	_ magistrala.Response = (*clientsPageRes)(nil)
	_ magistrala.Response = (*changeClientStatusRes)(nil)
	_ magistrala.Response = (*deleteClientRes)(nil)
)

type clientsPageMetaRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type createClientRes struct {
	clients.Client
	created bool
}

func (res createClientRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createClientRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/clients/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createClientRes) Empty() bool {
	return false
}

type updateClientRes struct {
	clients.Client
}

func (res updateClientRes) Code() int {
	return http.StatusOK
}

func (res updateClientRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateClientRes) Empty() bool {
	return false
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

type viewClientPermsRes struct {
	Permissions []string `json:"permissions"`
}

func (res viewClientPermsRes) Code() int {
	return http.StatusOK
}

func (res viewClientPermsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewClientPermsRes) Empty() bool {
	return false
}

type clientsPageRes struct {
	clientsPageMetaRes
	Clients []viewClientRes `json:"clients"`
}

func (res clientsPageRes) Code() int {
	return http.StatusOK
}

func (res clientsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res clientsPageRes) Empty() bool {
	return false
}

type changeClientStatusRes struct {
	clients.Client
}

func (res changeClientStatusRes) Code() int {
	return http.StatusOK
}

func (res changeClientStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeClientStatusRes) Empty() bool {
	return false
}

type setParentGroupRes struct{}

func (res setParentGroupRes) Code() int {
	return http.StatusAccepted
}

func (res setParentGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res setParentGroupRes) Empty() bool {
	return true
}

type removeParentGroupRes struct{}

func (res removeParentGroupRes) Code() int {
	return http.StatusNoContent
}

func (res removeParentGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeParentGroupRes) Empty() bool {
	return true
}

type deleteClientRes struct{}

func (res deleteClientRes) Code() int {
	return http.StatusNoContent
}

func (res deleteClientRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteClientRes) Empty() bool {
	return true
}
