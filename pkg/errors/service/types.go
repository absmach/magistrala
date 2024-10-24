// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/absmach/magistrala/pkg/errors"

// Wrapper for Service errors.
var (
	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = errors.New("failed to perform authentication over the entity")

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = errors.New("failed to perform authorization over the entity")

	// ErrDomainAuthorization indicates failure occurred while authorizing the domain.
	ErrDomainAuthorization = errors.New("failed to perform authorization over the domain")

	// ErrLogin indicates wrong login credentials.
	ErrLogin = errors.New("invalid user id or secret")

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = errors.New("failed to create entity")

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = errors.New("failed to remove entity")

	// ErrViewEntity indicates error in viewing entity or entities.
	ErrViewEntity = errors.New("view entity failed")

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = errors.New("update entity failed")

	// ErrInvalidStatus indicates an invalid status.
	ErrInvalidStatus = errors.New("invalid status")

	// ErrInvalidRole indicates that an invalid role.
	ErrInvalidRole = errors.New("invalid client role")

	// ErrInvalidPolicy indicates that an invalid policy.
	ErrInvalidPolicy = errors.New("invalid policy")

	// ErrEnableClient indicates error in enabling client.
	ErrEnableClient = errors.New("failed to enable client")

	// ErrDisableClient indicates error in disabling client.
	ErrDisableClient = errors.New("failed to disable client")

	// ErrAddPolicies indicates error in adding policies.
	ErrAddPolicies = errors.New("failed to add policies")

	// ErrDeletePolicies indicates error in removing policies.
	ErrDeletePolicies = errors.New("failed to remove policies")

	// ErrSearch indicates error in searching clients.
	ErrSearch = errors.New("failed to search clients")

	// ErrInvitationAlreadyRejected indicates that the invitation is already rejected.
	ErrInvitationAlreadyRejected = errors.New("invitation already rejected")

	// ErrInvitationAlreadyAccepted indicates that the invitation is already accepted.
	ErrInvitationAlreadyAccepted = errors.New("invitation already accepted")

	// ErrParentGroupAuthorization indicates failure occurred while authorizing the parent group.
	ErrParentGroupAuthorization = errors.New("failed to authorize parent group")

	// ErrMissingNames indicates that the user's names are missing.
	ErrMissingNames = errors.New("missing user names")
)
