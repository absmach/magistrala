// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	mgclients "github.com/absmach/magistrala/pkg/clients"
)

// MailSent message response when link is sent.
const MailSent = "Email with reset link is sent"

var (
	_ magistrala.Response = (*tokenRes)(nil)
	_ magistrala.Response = (*viewClientRes)(nil)
	_ magistrala.Response = (*createClientRes)(nil)
	_ magistrala.Response = (*deleteClientRes)(nil)
	_ magistrala.Response = (*clientsPageRes)(nil)
	_ magistrala.Response = (*viewMembersRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Total  uint64 `json:"total,omitempty"`
}

type createClientRes struct {
	mgclients.Client `json:",inline"`
	created          bool
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
			"Location": fmt.Sprintf("/users/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createClientRes) Empty() bool {
	return false
}

type tokenRes struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccessType   string `json:"access_type,omitempty"`
}

func (res tokenRes) Code() int {
	return http.StatusCreated
}

func (res tokenRes) Headers() map[string]string {
	return map[string]string{}
}

func (res tokenRes) Empty() bool {
	return res.AccessToken == "" || res.RefreshToken == ""
}

type updateClientRes struct {
	mgclients.Client `json:",inline"`
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
	mgclients.Client `json:",inline"`
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

type clientsPageRes struct {
	pageRes
	Clients []viewClientRes `json:"users"`
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

type viewMembersRes struct {
	mgclients.Client `json:",inline"`
}

func (res viewMembersRes) Code() int {
	return http.StatusOK
}

func (res viewMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMembersRes) Empty() bool {
	return false
}

type deleteClientRes struct {
	mgclients.Client `json:",inline"`
}

func (res deleteClientRes) Code() int {
	return http.StatusOK
}

func (res deleteClientRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteClientRes) Empty() bool {
	return false
}

type passwResetReqRes struct {
	Msg string `json:"msg"`
}

func (res passwResetReqRes) Code() int {
	return http.StatusCreated
}

func (res passwResetReqRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passwResetReqRes) Empty() bool {
	return false
}

type passwChangeRes struct{}

func (res passwChangeRes) Code() int {
	return http.StatusCreated
}

func (res passwChangeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passwChangeRes) Empty() bool {
	return false
}

type assignUsersRes struct{}

func (res assignUsersRes) Code() int {
	return http.StatusCreated
}

func (res assignUsersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUsersRes) Empty() bool {
	return true
}

type unassignUsersRes struct{}

func (res unassignUsersRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUsersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUsersRes) Empty() bool {
	return true
}
