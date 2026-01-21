// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/supermq"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/users"
)

// MailSent message response when link is sent.
const MailSent = "Email with reset link is sent"

var (
	_ supermq.Response = (*tokenRes)(nil)
	_ supermq.Response = (*sendVerificationRes)(nil)
	_ supermq.Response = (*verifyEmailRes)(nil)
	_ supermq.Response = (*viewUserRes)(nil)
	_ supermq.Response = (*createUserRes)(nil)
	_ supermq.Response = (*changeUserStatusRes)(nil)
	_ supermq.Response = (*usersPageRes)(nil)
	_ supermq.Response = (*passResetReqRes)(nil)
	_ supermq.Response = (*passChangeRes)(nil)
	_ supermq.Response = (*updateUserRes)(nil)
	_ supermq.Response = (*revokeRes)(nil)
	_ supermq.Response = (*deleteUserRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
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

type revokeRes struct{}

func (res revokeRes) Code() int {
	return http.StatusNoContent
}

func (res revokeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeRes) Empty() bool {
	return true
}

type listRefreshTokensRes struct {
	RefreshTokens []*grpcTokenV1.RefreshToken `json:"refresh_tokens"`
}

func (res listRefreshTokensRes) Code() int {
	return http.StatusOK
}

func (res listRefreshTokensRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRefreshTokensRes) Empty() bool {
	return false
}

type sendVerificationRes struct{}

func (res sendVerificationRes) Code() int {
	return http.StatusOK
}

func (res sendVerificationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res sendVerificationRes) Empty() bool {
	return true
}

type verifyEmailRes struct{}

func (res verifyEmailRes) Code() int {
	return http.StatusOK
}

func (res verifyEmailRes) Headers() map[string]string {
	return map[string]string{}
}

func (res verifyEmailRes) Empty() bool {
	return true
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

type passResetReqRes struct {
	Msg string `json:"msg"`
}

func (res passResetReqRes) Code() int {
	return http.StatusCreated
}

func (res passResetReqRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passResetReqRes) Empty() bool {
	return false
}

type passChangeRes struct{}

func (res passChangeRes) Code() int {
	return http.StatusCreated
}

func (res passChangeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passChangeRes) Empty() bool {
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
