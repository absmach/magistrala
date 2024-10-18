// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/roles"
)

var (
	_ magistrala.Response = (*createRoleRes)(nil)
	_ magistrala.Response = (*listRolesRes)(nil)
	_ magistrala.Response = (*viewRoleRes)(nil)
	_ magistrala.Response = (*updateRoleRes)(nil)
	_ magistrala.Response = (*deleteRoleRes)(nil)
	_ magistrala.Response = (*listAvailableActionsRes)(nil)
	_ magistrala.Response = (*addRoleActionsRes)(nil)
	_ magistrala.Response = (*listRoleActionsRes)(nil)
	_ magistrala.Response = (*deleteRoleActionsRes)(nil)
	_ magistrala.Response = (*deleteAllRoleActionsRes)(nil)
	_ magistrala.Response = (*addRoleMembersRes)(nil)
	_ magistrala.Response = (*listRoleMembersRes)(nil)
	_ magistrala.Response = (*deleteRoleMembersRes)(nil)
	_ magistrala.Response = (*deleteAllRoleMemberRes)(nil)
)

type createRoleRes struct {
	roles.Role
}

func (res createRoleRes) Code() int {
	return http.StatusCreated
}

func (res createRoleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createRoleRes) Empty() bool {
	return false
}

type listRolesRes struct {
	roles.RolePage
}

func (res listRolesRes) Code() int {
	return http.StatusOK
}

func (res listRolesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRolesRes) Empty() bool {
	return false
}

type viewRoleRes struct {
	roles.Role
}

func (res viewRoleRes) Code() int {
	return http.StatusOK
}

func (res viewRoleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewRoleRes) Empty() bool {
	return false
}

type updateRoleRes struct {
	roles.Role
}

func (res updateRoleRes) Code() int {
	return http.StatusOK
}

func (res updateRoleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateRoleRes) Empty() bool {
	return false
}

type deleteRoleRes struct {
}

func (res deleteRoleRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRoleRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRoleRes) Empty() bool {
	return true
}

type listAvailableActionsRes struct {
	AvailableActions []string `json:"available_actions"`
}

func (res listAvailableActionsRes) Code() int {
	return http.StatusOK
}

func (res listAvailableActionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listAvailableActionsRes) Empty() bool {
	return false
}

type addRoleActionsRes struct {
	Actions []string `json:"actions"`
}

func (res addRoleActionsRes) Code() int {
	return http.StatusOK
}

func (res addRoleActionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addRoleActionsRes) Empty() bool {
	return false
}

type listRoleActionsRes struct {
	Actions []string `json:"actions"`
}

func (res listRoleActionsRes) Code() int {
	return http.StatusOK
}

func (res listRoleActionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRoleActionsRes) Empty() bool {
	return false
}

type deleteRoleActionsRes struct{}

func (res deleteRoleActionsRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRoleActionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRoleActionsRes) Empty() bool {
	return true
}

type deleteAllRoleActionsRes struct{}

func (res deleteAllRoleActionsRes) Code() int {
	return http.StatusNoContent
}

func (res deleteAllRoleActionsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteAllRoleActionsRes) Empty() bool {
	return true
}

type addRoleMembersRes struct {
	Members []string `json:"members"`
}

func (res addRoleMembersRes) Code() int {
	return http.StatusOK
}

func (res addRoleMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res addRoleMembersRes) Empty() bool {
	return false
}

type listRoleMembersRes struct {
	roles.MembersPage
}

func (res listRoleMembersRes) Code() int {
	return http.StatusOK
}

func (res listRoleMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRoleMembersRes) Empty() bool {
	return false
}

type deleteRoleMembersRes struct{}

func (res deleteRoleMembersRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRoleMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRoleMembersRes) Empty() bool {
	return true
}

type deleteAllRoleMemberRes struct{}

func (res deleteAllRoleMemberRes) Code() int {
	return http.StatusNoContent
}

func (res deleteAllRoleMemberRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteAllRoleMemberRes) Empty() bool {
	return true
}
