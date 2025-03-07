// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/domains"
)

var (
	_ supermq.Response = (*createDomainRes)(nil)
	_ supermq.Response = (*retrieveDomainRes)(nil)
	_ supermq.Response = (*listDomainsRes)(nil)
	_ supermq.Response = (*enableDomainRes)(nil)
	_ supermq.Response = (*disableDomainRes)(nil)
	_ supermq.Response = (*freezeDomainRes)(nil)
	_ supermq.Response = (*sendInvitationRes)(nil)
	_ supermq.Response = (*viewInvitationRes)(nil)
	_ supermq.Response = (*listInvitationsRes)(nil)
	_ supermq.Response = (*acceptInvitationRes)(nil)
	_ supermq.Response = (*rejectInvitationRes)(nil)
	_ supermq.Response = (*deleteInvitationRes)(nil)
)

type createDomainRes struct {
	domains.Domain
}

func (res createDomainRes) Code() int {
	return http.StatusCreated
}

func (res createDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createDomainRes) Empty() bool {
	return false
}

type retrieveDomainRes struct {
	domains.Domain
}

func (res retrieveDomainRes) Code() int {
	return http.StatusOK
}

func (res retrieveDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res retrieveDomainRes) Empty() bool {
	return false
}

type updateDomainRes struct {
	domains.Domain
}

func (res updateDomainRes) Code() int {
	return http.StatusOK
}

func (res updateDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateDomainRes) Empty() bool {
	return false
}

type listDomainsRes struct {
	domains.DomainsPage
}

func (res listDomainsRes) Code() int {
	return http.StatusOK
}

func (res listDomainsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listDomainsRes) Empty() bool {
	return false
}

type enableDomainRes struct{}

func (res enableDomainRes) Code() int {
	return http.StatusOK
}

func (res enableDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res enableDomainRes) Empty() bool {
	return true
}

type disableDomainRes struct{}

func (res disableDomainRes) Code() int {
	return http.StatusOK
}

func (res disableDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res disableDomainRes) Empty() bool {
	return true
}

type freezeDomainRes struct{}

func (res freezeDomainRes) Code() int {
	return http.StatusOK
}

func (res freezeDomainRes) Headers() map[string]string {
	return map[string]string{}
}

func (res freezeDomainRes) Empty() bool {
	return true
}

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
	domains.Invitation `json:",inline"`
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
	domains.InvitationPage `json:",inline"`
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
	return http.StatusNoContent
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

type rejectInvitationRes struct{}

func (res rejectInvitationRes) Code() int {
	return http.StatusNoContent
}

func (res rejectInvitationRes) Headers() map[string]string {
	return map[string]string{}
}

func (res rejectInvitationRes) Empty() bool {
	return true
}
