// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"

	"github.com/mainflux/mainflux"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
)

var (
	_ mainflux.Response = (*viewClientRes)(nil)
	_ mainflux.Response = (*createClientRes)(nil)
	_ mainflux.Response = (*deleteClientRes)(nil)
	_ mainflux.Response = (*clientsPageRes)(nil)
	_ mainflux.Response = (*viewMembersRes)(nil)
	_ mainflux.Response = (*memberPageRes)(nil)
	_ mainflux.Response = (*assignUsersGroupsRes)(nil)
	_ mainflux.Response = (*unassignUsersGroupsRes)(nil)
	_ mainflux.Response = (*connectChannelThingRes)(nil)
	_ mainflux.Response = (*disconnectChannelThingRes)(nil)
)

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Total  uint64 `json:"total,omitempty"`
}

type createClientRes struct {
	mfclients.Client
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
	mfclients.Client
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
	mfclients.Client
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
	mfclients.Client
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

type memberPageRes struct {
	pageRes
	Members []viewMembersRes `json:"things"`
}

func (res memberPageRes) Code() int {
	return http.StatusOK
}

func (res memberPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res memberPageRes) Empty() bool {
	return false
}

type deleteClientRes struct {
	mfclients.Client
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

type assignUsersGroupsRes struct {
}

func (res assignUsersGroupsRes) Code() int {
	return http.StatusOK
}

func (res assignUsersGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUsersGroupsRes) Empty() bool {
	return true
}

type unassignUsersGroupsRes struct {
}

func (res unassignUsersGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUsersGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUsersGroupsRes) Empty() bool {
	return true
}

type assignUsersRes struct {
}

func (res assignUsersRes) Code() int {
	return http.StatusOK
}

func (res assignUsersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUsersRes) Empty() bool {
	return true
}

type unassignUsersRes struct {
}

func (res unassignUsersRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUsersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUsersRes) Empty() bool {
	return true
}

type assignUserGroupsRes struct {
}

func (res assignUserGroupsRes) Code() int {
	return http.StatusOK
}

func (res assignUserGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignUserGroupsRes) Empty() bool {
	return true
}

type unassignUserGroupsRes struct {
}

func (res unassignUserGroupsRes) Code() int {
	return http.StatusNoContent
}

func (res unassignUserGroupsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignUserGroupsRes) Empty() bool {
	return true
}

type connectChannelThingRes struct {
}

func (res connectChannelThingRes) Code() int {
	return http.StatusOK
}

func (res connectChannelThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connectChannelThingRes) Empty() bool {
	return true
}

type disconnectChannelThingRes struct {
}

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
	return http.StatusOK
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
