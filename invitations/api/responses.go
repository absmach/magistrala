// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/invitations"
)

var (
	_ magistrala.Response = (*sendInvitationRes)(nil)
	_ magistrala.Response = (*viewInvitationRes)(nil)
	_ magistrala.Response = (*listInvitationsRes)(nil)
	_ magistrala.Response = (*acceptInvitationRes)(nil)
	_ magistrala.Response = (*deleteInvitationRes)(nil)
)

type sendInvitationRes struct {
	Message string `json:"message"`
}

func (res sendInvitationRes) Code() int {
	return http.StatusCreated
}

func (res sendInvitationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res sendInvitationRes) Empty() bool {
	return true
}

type viewInvitationRes struct {
	invitations.Invitation `json:",inline"`
}

func (res viewInvitationRes) Code() int {
	return http.StatusOK
}

func (res viewInvitationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewInvitationRes) Empty() bool {
	return false
}

type listInvitationsRes struct {
	invitations.InvitationPage `json:",inline"`
}

func (res listInvitationsRes) Code() int {
	return http.StatusOK
}

func (res listInvitationsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listInvitationsRes) Empty() bool {
	return false
}

type acceptInvitationRes struct{}

func (res acceptInvitationRes) Code() int {
	return http.StatusOK
}

func (res acceptInvitationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res acceptInvitationRes) Empty() bool {
	return true
}

type deleteInvitationRes struct{}

func (res deleteInvitationRes) Code() int {
	return http.StatusNoContent
}

func (res deleteInvitationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteInvitationRes) Empty() bool {
	return true
}
