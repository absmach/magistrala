// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
)

var (
	_ mainflux.Response = (*viewMembershipRes)(nil)
	_ mainflux.Response = (*membershipPageRes)(nil)
	_ mainflux.Response = (*createGroupRes)(nil)
	_ mainflux.Response = (*groupPageRes)(nil)
	_ mainflux.Response = (*changeStatusRes)(nil)
	_ mainflux.Response = (*viewGroupRes)(nil)
	_ mainflux.Response = (*updateGroupRes)(nil)
)

type viewMembershipRes struct {
	mfgroups.Group `json:",inline"`
}

func (res viewMembershipRes) Code() int {
	return http.StatusOK
}

func (res viewMembershipRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMembershipRes) Empty() bool {
	return false
}

type membershipPageRes struct {
	pageRes
	Memberships []viewMembershipRes `json:"memberships"`
}

func (res membershipPageRes) Code() int {
	return http.StatusOK
}

func (res membershipPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res membershipPageRes) Empty() bool {
	return false
}

type viewGroupRes struct {
	mfgroups.Group `json:",inline"`
}

func (res viewGroupRes) Code() int {
	return http.StatusOK
}

func (res viewGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewGroupRes) Empty() bool {
	return false
}

type createGroupRes struct {
	mfgroups.Group `json:",inline"`
	created        bool
}

func (res createGroupRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createGroupRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/groups/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createGroupRes) Empty() bool {
	return false
}

type groupPageRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total,omitempty"`
	Level  uint64 `json:"level,omitempty"`
}

func (res groupPageRes) Code() int {
	return http.StatusOK
}

func (res groupPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupPageRes) Empty() bool {
	return false
}

type updateGroupRes struct {
	mfgroups.Group `json:",inline"`
}

func (res updateGroupRes) Code() int {
	return http.StatusOK
}

func (res updateGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateGroupRes) Empty() bool {
	return false
}

type changeStatusRes struct {
	mfgroups.Group `json:",inline"`
}

func (res changeStatusRes) Code() int {
	return http.StatusOK
}

func (res changeStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res changeStatusRes) Empty() bool {
	return false
}
