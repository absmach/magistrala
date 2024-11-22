// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/groups"
)

var (
	_ magistrala.Response = (*createGroupRes)(nil)
	_ magistrala.Response = (*groupPageRes)(nil)
	_ magistrala.Response = (*changeStatusRes)(nil)
	_ magistrala.Response = (*viewGroupRes)(nil)
	_ magistrala.Response = (*updateGroupRes)(nil)
	_ magistrala.Response = (*retrieveGroupHierarchyRes)(nil)
	_ magistrala.Response = (*addParentGroupRes)(nil)
	_ magistrala.Response = (*removeParentGroupRes)(nil)
	_ magistrala.Response = (*addChildrenGroupsRes)(nil)
	_ magistrala.Response = (*removeChildrenGroupsRes)(nil)
	_ magistrala.Response = (*removeAllChildrenGroupsRes)(nil)
	_ magistrala.Response = (*listChildrenGroupsRes)(nil)
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
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
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

type retrieveGroupHierarchyRes struct {
	Level     uint64         `json:"level"`
	Direction int64          `json:"direction"`
	Groups    []viewGroupRes `json:"groups"`
}

func (res retrieveGroupHierarchyRes) Code() int {
	return http.StatusOK
}

func (res retrieveGroupHierarchyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res retrieveGroupHierarchyRes) Empty() bool {
	return false
}

type addParentGroupRes struct{}

func (res addParentGroupRes) Code() int {
	return http.StatusNoContent
}

func (res addParentGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addParentGroupRes) Empty() bool {
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

type addChildrenGroupsRes struct{}

func (res addChildrenGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res addChildrenGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addChildrenGroupsRes) Empty() bool {
	return true
}

type removeChildrenGroupsRes struct{}

func (res removeChildrenGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res removeChildrenGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeChildrenGroupsRes) Empty() bool {
	return true
}

type removeAllChildrenGroupsRes struct{}

func (res removeAllChildrenGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res removeAllChildrenGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeAllChildrenGroupsRes) Empty() bool {
	return true
}

type listChildrenGroupsRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

func (res listChildrenGroupsRes) Code() int {
	return http.StatusOK
}

func (res listChildrenGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listChildrenGroupsRes) Empty() bool {
	return false
}
