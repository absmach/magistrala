// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/users"
)

// MailSent message response when link is sent.
const MailSent = "Email with reset link is sent"

var (
	_ supermq.Response = (*tokenRes)(nil)
	_ supermq.Response = (*viewUserRes)(nil)
	_ supermq.Response = (*createUserRes)(nil)
	_ supermq.Response = (*changeUserStatusRes)(nil)
	_ supermq.Response = (*usersPageRes)(nil)
	_ supermq.Response = (*viewMembersRes)(nil)
	_ supermq.Response = (*passwResetReqRes)(nil)
	_ supermq.Response = (*passwChangeRes)(nil)
	_ supermq.Response = (*updateUserRes)(nil)
	_ supermq.Response = (*tokenRes)(nil)
	_ supermq.Response = (*deleteUserRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type createUserRes struct {
	users.User
	created bool
}

func (res createUserRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createUserRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/users/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createUserRes) Empty() bool {
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

type updateUserRes struct {
	users.User `json:",inline"`
}

func (res updateUserRes) Code() int {
	return http.StatusOK
}

func (res updateUserRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateUserRes) Empty() bool {
	return false
}

type viewUserRes struct {
	users.User `json:",inline"`
}

func (res viewUserRes) Code() int {
	return http.StatusOK
}

func (res viewUserRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewUserRes) Empty() bool {
	return false
}

type usersPageRes struct {
	pageRes
	Users []viewUserRes `json:"users"`
}

func (res usersPageRes) Code() int {
	return http.StatusOK
}

func (res usersPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res usersPageRes) Empty() bool {
	return false
}

type viewMembersRes struct {
	users.User `json:",inline"`
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

type changeUserStatusRes struct {
	users.User `json:",inline"`
}

func (res changeUserStatusRes) Code() int {
	return http.StatusOK
}

func (res changeUserStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeUserStatusRes) Empty() bool {
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

type deleteUserRes struct {
	deleted bool
}

func (res deleteUserRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusOK
}

func (res deleteUserRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteUserRes) Empty() bool {
	return true
}
