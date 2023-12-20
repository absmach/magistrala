// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"

	"github.com/absmach/magistrala"
	mgclients "github.com/absmach/magistrala/pkg/clients"
)

var (
	_ magistrala.Response = (*viewClientRes)(nil)
	_ magistrala.Response = (*viewClientPermsRes)(nil)
	_ magistrala.Response = (*createClientRes)(nil)
	_ magistrala.Response = (*deleteClientRes)(nil)
	_ magistrala.Response = (*clientsPageRes)(nil)
	_ magistrala.Response = (*viewMembersRes)(nil)
	_ magistrala.Response = (*assignUsersGroupsRes)(nil)
	_ magistrala.Response = (*unassignUsersGroupsRes)(nil)
	_ magistrala.Response = (*connectChannelThingRes)(nil)
	_ magistrala.Response = (*disconnectChannelThingRes)(nil)
	_ magistrala.Response = (*changeClientStatusRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Total  uint64 `json:"total,omitempty"`
}

type createClientRes struct {
	mgclients.Client
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
			"Location": fmt.Sprintf("/things/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createClientRes) Empty() bool {
	return false
}

type updateClientRes struct {
	mgclients.Client
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
	mgclients.Client
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
	pageRes
	Clients []viewClientRes `json:"things"`
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
	mgclients.Client
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

type changeClientStatusRes struct {
	mgclients.Client
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

type assignUsersGroupsRes struct{}

func (res assignUsersGroupsRes) Code() int {
	return http.StatusCreated
}

func (res assignUsersGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUsersGroupsRes) Empty() bool {
	return true
}

type unassignUsersGroupsRes struct{}

func (res unassignUsersGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUsersGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUsersGroupsRes) Empty() bool {
	return true
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

type assignUserGroupsRes struct{}

func (res assignUserGroupsRes) Code() int {
	return http.StatusCreated
}

func (res assignUserGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUserGroupsRes) Empty() bool {
	return true
}

type unassignUserGroupsRes struct{}

func (res unassignUserGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUserGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUserGroupsRes) Empty() bool {
	return true
}

type connectChannelThingRes struct{}

func (res connectChannelThingRes) Code() int {
	return http.StatusCreated
}

func (res connectChannelThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectChannelThingRes) Empty() bool {
	return true
}

type disconnectChannelThingRes struct{}

func (res disconnectChannelThingRes) Code() int {
	return http.StatusNoContent
}

func (res disconnectChannelThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disconnectChannelThingRes) Empty() bool {
	return true
}

type thingShareRes struct{}

func (res thingShareRes) Code() int {
	return http.StatusCreated
}

func (res thingShareRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingShareRes) Empty() bool {
	return true
}

type thingUnshareRes struct{}

func (res thingUnshareRes) Code() int {
	return http.StatusNoContent
}

func (res thingUnshareRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingUnshareRes) Empty() bool {
	return true
}
