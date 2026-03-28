// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

const (
	OpCreateDomain permissions.Operation = iota
	OpUpdateDomain
	OpRetrieveDomain
	OpEnableDomain
	OpDisableDomain
	OpFreezeDomain
	OpListDomains
	OpDeleteDomain

	OpSendDomainInvitation
	OpListDomainInvitations
	OpDeleteDomainInvitation

	OpViewInvitation
	OpListInvitations
	OpAcceptInvitation
	OpRejectInvitation

	OpCreateDomainClients
	OpListDomainClients
	OpCreateDomainChannels
	OpListDomainChannels
	OpCreateDomainGroups
	OpListDomainGroups
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	ops := map[permissions.Operation]permissions.OperationDetails{
		OpUpdateDomain: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpRetrieveDomain: {
			Name:               "read",
			PermissionRequired: true,
		},
		OpListDomains: {
			Name:               "list",
			PermissionRequired: true,
		},
		OpEnableDomain: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableDomain: {
			Name:               "disable",
			PermissionRequired: true,
		},
		OpDeleteDomain: {
			Name:               "delete",
			PermissionRequired: true,
		},

		// Permission not required, only Super Admin can freeze the domain
		OpFreezeDomain: {
			Name:               "freeze",
			PermissionRequired: false,
		},

		OpCreateDomain: {
			Name:               "create",
			PermissionRequired: false,
		},

		// Domain Invitation related permissions
		OpSendDomainInvitation: {
			Name:               "send_invitation",
			PermissionRequired: true,
		},

		OpDeleteDomainInvitation: {
			Name:               "delete_invitation",
			PermissionRequired: true,
		},

		OpListDomainInvitations: {
			Name:               "list_domain_invitation",
			PermissionRequired: true,
		},

		// User Invitation related permissions
		OpViewInvitation: {
			Name:               "view_invitation",
			PermissionRequired: false,
		},
		OpListInvitations: {
			Name:               "list_invitation",
			PermissionRequired: false,
		},

		OpAcceptInvitation: {
			Name:               "accept_invitation",
			PermissionRequired: false,
		},

		OpRejectInvitation: {
			Name:               "reject_invitation",
			PermissionRequired: false,
		},

		// Operations related to entities
		OpCreateDomainClients: {
			Name:               "create_clients",
			PermissionRequired: true,
		},

		OpListDomainClients: {
			Name:               "list_clients",
			PermissionRequired: true,
		},

		OpCreateDomainChannels: {
			Name:               "create_channels",
			PermissionRequired: true,
		},

		OpListDomainChannels: {
			Name:               "list_channels",
			PermissionRequired: true,
		},

		OpCreateDomainGroups: {
			Name:               "create_groups",
			PermissionRequired: true,
		},

		OpListDomainGroups: {
			Name:               "list_groups",
			PermissionRequired: true,
		},
	}
	return ops
}
