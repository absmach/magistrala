// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/roles"
)

const (
	domainPrefix         = "domain."
	domainCreate         = domainPrefix + "create"
	domainRetrieve       = domainPrefix + "retrieve"
	domainUpdate         = domainPrefix + "update"
	domainEnable         = domainPrefix + "enable"
	domainDisable        = domainPrefix + "disable"
	domainFreeze         = domainPrefix + "freeze"
	domainList           = domainPrefix + "list"
	domainDelete         = domainPrefix + "delete"
	invitationPrefix     = "invitation."
	invitationSend       = invitationPrefix + "send"
	invitationAccept     = invitationPrefix + "accept"
	invitationReject     = invitationPrefix + "reject"
	invitationList       = invitationPrefix + "list"
	invitationListDomain = invitationPrefix + "list_domain"
	invitationDelete     = invitationPrefix + "delete"
)

var (
	_ events.Event = (*createDomainEvent)(nil)
	_ events.Event = (*retrieveDomainEvent)(nil)
	_ events.Event = (*updateDomainEvent)(nil)
	_ events.Event = (*enableDomainEvent)(nil)
	_ events.Event = (*disableDomainEvent)(nil)
	_ events.Event = (*freezeDomainEvent)(nil)
	_ events.Event = (*listDomainsEvent)(nil)
	_ events.Event = (*deleteDomainEvent)(nil)
	_ events.Event = (*sendInvitationEvent)(nil)
	_ events.Event = (*listInvitationsEvent)(nil)
	_ events.Event = (*listDomainInvitationsEvent)(nil)
	_ events.Event = (*acceptInvitationEvent)(nil)
	_ events.Event = (*rejectInvitationEvent)(nil)
	_ events.Event = (*deleteInvitationEvent)(nil)
)

type createDomainEvent struct {
	domains.Domain
	rolesProvisioned []roles.RoleProvision
	authn.Session
	requestID string
}

func (cde createDomainEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":         domainCreate,
		"id":                cde.ID,
		"route":             cde.Route,
		"status":            cde.Status.String(),
		"created_at":        cde.CreatedAt,
		"created_by":        cde.CreatedBy,
		"roles_provisioned": cde.rolesProvisioned,
		"user_id":           cde.UserID,
		"token_type":        cde.Type.String(),
		"super_admin":       cde.SuperAdmin,
		"request_id":        cde.requestID,
	}

	if cde.Name != "" {
		val["name"] = cde.Name
	}
	if len(cde.Tags) > 0 {
		val["tags"] = cde.Tags
	}
	if cde.Metadata != nil {
		val["metadata"] = cde.Metadata
	}

	return val, nil
}

type retrieveDomainEvent struct {
	domains.Domain
	authn.Session
	requestID string
}

func (rde retrieveDomainEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   domainRetrieve,
		"id":          rde.ID,
		"route":       rde.Route,
		"status":      rde.Status.String(),
		"created_at":  rde.CreatedAt,
		"user_id":     rde.UserID,
		"token_type":  rde.Type.String(),
		"super_admin": rde.SuperAdmin,
		"request_id":  rde.requestID,
	}

	if rde.Name != "" {
		val["name"] = rde.Name
	}
	if len(rde.Tags) > 0 {
		val["tags"] = rde.Tags
	}
	if rde.Metadata != nil {
		val["metadata"] = rde.Metadata
	}

	if !rde.UpdatedAt.IsZero() {
		val["updated_at"] = rde.UpdatedAt
	}
	if rde.UpdatedBy != "" {
		val["updated_by"] = rde.UpdatedBy
	}
	return val, nil
}

type updateDomainEvent struct {
	domain    domains.Domain
	Session   authn.Session
	requestID string
}

func (ude updateDomainEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   domainUpdate,
		"id":          ude.domain.ID,
		"route":       ude.domain.Route,
		"status":      ude.domain.Status.String(),
		"created_at":  ude.domain.CreatedAt,
		"created_by":  ude.domain.CreatedBy,
		"updated_at":  ude.domain.UpdatedAt,
		"updated_by":  ude.domain.UpdatedBy,
		"user_id":     ude.Session.UserID,
		"token_type":  ude.Session.Type.String(),
		"super_admin": ude.Session.SuperAdmin,
		"request_id":  ude.requestID,
	}

	if ude.domain.Name != "" {
		val["name"] = ude.domain.Name
	}
	if len(ude.domain.Tags) > 0 {
		val["tags"] = ude.domain.Tags
	}
	if ude.domain.Metadata != nil {
		val["metadata"] = ude.domain.Metadata
	}

	return val, nil
}

type enableDomainEvent struct {
	domainID  string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (cdse enableDomainEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   domainEnable,
		"id":          cdse.domainID,
		"updated_at":  cdse.updatedAt,
		"updated_by":  cdse.updatedBy,
		"user_id":     cdse.UserID,
		"token_type":  cdse.Type.String(),
		"super_admin": cdse.SuperAdmin,
		"request_id":  cdse.requestID,
	}, nil
}

type disableDomainEvent struct {
	domainID  string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (cdse disableDomainEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   domainDisable,
		"id":          cdse.domainID,
		"updated_at":  cdse.updatedAt,
		"updated_by":  cdse.updatedBy,
		"user_id":     cdse.UserID,
		"token_type":  cdse.Type.String(),
		"super_admin": cdse.SuperAdmin,
		"request_id":  cdse.requestID,
	}, nil
}

type freezeDomainEvent struct {
	domainID  string
	updatedAt time.Time
	updatedBy string
	authn.Session
	requestID string
}

func (cdse freezeDomainEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   domainFreeze,
		"id":          cdse.domainID,
		"updated_at":  cdse.updatedAt,
		"updated_by":  cdse.updatedBy,
		"user_id":     cdse.UserID,
		"token_type":  cdse.Type.String(),
		"super_admin": cdse.SuperAdmin,
		"request_id":  cdse.requestID,
	}, nil
}

type listDomainsEvent struct {
	domains.Page
	total      uint64
	userID     string
	tokenType  string
	superAdmin bool
	requestID  string
}

func (lde listDomainsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   domainList,
		"total":       lde.total,
		"offset":      lde.Offset,
		"limit":       lde.Limit,
		"user_id":     lde.userID,
		"token_type":  lde.tokenType,
		"super_admin": lde.superAdmin,
		"request_id":  lde.requestID,
	}

	if lde.Name != "" {
		val["name"] = lde.Name
	}
	if lde.Order != "" {
		val["order"] = lde.Order
	}
	if lde.Dir != "" {
		val["dir"] = lde.Dir
	}
	if lde.Metadata != nil {
		val["metadata"] = lde.Metadata
	}
	if len(lde.Tags.Elements) > 0 {
		val["tag"] = lde.Tags.Elements
	}
	if lde.RoleID != "" {
		val["role_id"] = lde.RoleID
	}
	if lde.RoleName != "" {
		val["role_name"] = lde.RoleName
	}
	if len(lde.Actions) != 0 {
		val["actions"] = lde.Actions
	}
	if lde.Status.String() != "" {
		val["status"] = lde.Status.String()
	}
	if lde.ID != "" {
		val["id"] = lde.ID
	}
	if len(lde.IDs) > 0 {
		val["ids"] = lde.IDs
	}
	if lde.Identity != "" {
		val["identity"] = lde.Identity
	}
	if lde.UserID != "" {
		val["user_id"] = lde.UserID
	}

	return val, nil
}

type deleteDomainEvent struct {
	domainID string
	authn.Session
	requestID string
}

func (dde deleteDomainEvent) Encode() (map[string]any, error) {
	return map[string]any{
		"operation":   domainDelete,
		"id":          dde.domainID,
		"user_id":     dde.UserID,
		"token_type":  dde.Type.String(),
		"super_admin": dde.SuperAdmin,
		"request_id":  dde.requestID,
	}, nil
}

type sendInvitationEvent struct {
	invitation domains.Invitation
	session    authn.Session
	requestID  string
}

func (sie sendInvitationEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":       invitationSend,
		"invitee_user_id": sie.invitation.InviteeUserID,
		"domain_id":       sie.invitation.DomainID,
		"invited_by":      sie.session.UserID,
		"role_id":         sie.invitation.RoleID,
		"token_type":      sie.session.Type.String(),
		"super_admin":     sie.session.SuperAdmin,
		"request_id":      sie.requestID,
	}

	if sie.invitation.DomainName != "" {
		val["domain_name"] = sie.invitation.DomainName
	}
	if sie.invitation.RoleName != "" {
		val["role_name"] = sie.invitation.RoleName
	}

	return val, nil
}

type listInvitationsEvent struct {
	domains.InvitationPageMeta
	session   authn.Session
	requestID string
}

func (lie listInvitationsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  invitationList,
		"offset":     lie.Offset,
		"limit":      lie.Limit,
		"user_id":    lie.session.UserID,
		"token_type": lie.session.Type.String(),
		"request_id": lie.requestID,
	}

	if lie.InvitedBy != "" {
		val["invited_by"] = lie.InvitedBy
	}
	if lie.InviteeUserID != "" {
		val["invitee_user_id"] = lie.InviteeUserID
	}
	if lie.DomainID != "" {
		val["domain_id"] = lie.DomainID
	}
	if lie.RoleID != "" {
		val["role_id"] = lie.RoleID
	}
	if lie.State.String() != domains.UnknownState {
		val["state"] = lie.State.String()
	}

	return val, nil
}

type listDomainInvitationsEvent struct {
	domains.InvitationPageMeta
	session   authn.Session
	requestID string
}

func (lie listDomainInvitationsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":   invitationListDomain,
		"offset":      lie.Offset,
		"limit":       lie.Limit,
		"domain_id":   lie.session.DomainID,
		"token_type":  lie.session.Type.String(),
		"super_admin": lie.session.SuperAdmin,
		"request_id":  lie.requestID,
	}

	if lie.InvitedBy != "" {
		val["invited_by"] = lie.InvitedBy
	}
	if lie.InviteeUserID != "" {
		val["invitee_user_id"] = lie.InviteeUserID
	}
	if lie.RoleID != "" {
		val["role_id"] = lie.RoleID
	}
	if lie.State.String() != domains.UnknownState {
		val["state"] = lie.State.String()
	}

	return val, nil
}

type acceptInvitationEvent struct {
	invitation domains.Invitation
	session    authn.Session
	requestID  string
}

func (aie acceptInvitationEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":       invitationAccept,
		"domain_id":       aie.invitation.DomainID,
		"invitee_user_id": aie.session.UserID,
		"invited_by":      aie.invitation.InvitedBy,
		"role_id":         aie.invitation.RoleID,
		"token_type":      aie.session.Type.String(),
		"super_admin":     aie.session.SuperAdmin,
		"request_id":      aie.requestID,
	}

	if aie.invitation.DomainName != "" {
		val["domain_name"] = aie.invitation.DomainName
	}
	if aie.invitation.RoleName != "" {
		val["role_name"] = aie.invitation.RoleName
	}

	return val, nil
}

type rejectInvitationEvent struct {
	invitation domains.Invitation
	session    authn.Session
	requestID  string
}

func (rie rejectInvitationEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":       invitationReject,
		"domain_id":       rie.invitation.DomainID,
		"invitee_user_id": rie.session.UserID,
		"invited_by":      rie.invitation.InvitedBy,
		"role_id":         rie.invitation.RoleID,
		"token_type":      rie.session.Type.String(),
		"super_admin":     rie.session.SuperAdmin,
		"request_id":      rie.requestID,
	}

	if rie.invitation.DomainName != "" {
		val["domain_name"] = rie.invitation.DomainName
	}
	if rie.invitation.RoleName != "" {
		val["role_name"] = rie.invitation.RoleName
	}

	return val, nil
}

type deleteInvitationEvent struct {
	inviteeUserID string
	domainID      string
	session       authn.Session
	requestID     string
}

func (die deleteInvitationEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":       invitationDelete,
		"invitee_user_id": die.inviteeUserID,
		"domain_id":       die.domainID,
		"token_type":      die.session.Type.String(),
		"super_admin":     die.session.SuperAdmin,
		"request_id":      die.requestID,
	}

	return val, nil
}
