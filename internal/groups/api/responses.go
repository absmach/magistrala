// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/groups"
)

var (
	_ magistrala.Response = (*createGroupRes)(nil)
	_ magistrala.Response = (*groupPageRes)(nil)
	_ magistrala.Response = (*changeStatusRes)(nil)
	_ magistrala.Response = (*viewGroupRes)(nil)
	_ magistrala.Response = (*updateGroupRes)(nil)
	_ magistrala.Response = (*assignRes)(nil)
	_ magistrala.Response = (*unassignRes)(nil)
)

type viewGroupRes struct {
	groups.Group `json:",inline"`
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

type viewGroupPermsRes struct {
	Permissions []string `json:"permissions"`
}

func (res viewGroupPermsRes) Code() int {
	return http.StatusOK
}

func (res viewGroupPermsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewGroupPermsRes) Empty() bool {
	return false
}

type createGroupRes struct {
	groups.Group `json:",inline"`
	created      bool
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
	groups.Group `json:",inline"`
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
	groups.Group `json:",inline"`
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

type assignRes struct {
	assigned bool
}

func (res assignRes) Code() int {
	if res.assigned {
		return http.StatusCreated
	}

	return http.StatusBadRequest
}

func (res assignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignRes) Empty() bool {
	return true
}

type unassignRes struct {
	unassigned bool
}

func (res unassignRes) Code() int {
	if res.unassigned {
		return http.StatusCreated
	}

	return http.StatusBadRequest
}

func (res unassignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignRes) Empty() bool {
	return true
}

type listMembersRes struct {
	pageRes
	Members []groups.Member `json:"members"`
}

func (res listMembersRes) Code() int {
	return http.StatusOK
}

func (res listMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMembersRes) Empty() bool {
	return false
}

type deleteGroupRes struct {
	deleted bool
}

func (res deleteGroupRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusBadRequest
}

func (res deleteGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteGroupRes) Empty() bool {
	return true
}
