// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package apiutil

import "github.com/absmach/magistrala/pkg/errors"

// Errors defined in this file are used by the LoggingErrorEncoder decorator
// to distinguish and log API request validation errors and avoid that service
// errors are logged twice.
var (
	// ErrValidation indicates that an error was returned by the API.
	ErrValidation = errors.New("something went wrong with the request")

	// ErrBearerToken indicates missing or invalid bearer user token.
	ErrBearerToken = errors.New("missing or invalid bearer user token")

	// ErrBearerKey indicates missing or invalid bearer entity key.
	ErrBearerKey = errors.New("missing or invalid bearer entity key")

	// ErrMissingID indicates missing entity ID.
	ErrMissingID = errors.New("missing entity id")

	// ErrMissingClientID indicates missing client ID.
	ErrMissingClientID = errors.New("missing cient id")

	// ErrMissingChannelID indicates missing client ID.
	ErrMissingChannelID = errors.New("missing channel id")

	// ErrMissingConnectionType indicates missing connection tpye.
	ErrMissingConnectionType = errors.New("missing connection type")

	// ErrMissingParentGroupID indicates missing parent group ID.
	ErrMissingParentGroupID = errors.New("missing parent group id")

	// ErrMissingChildrenGroupIDs indicates missing children group IDs.
	ErrMissingChildrenGroupIDs = errors.New("missing children group ids")

	// ErrSelfParentingNotAllowed indicates child id is same as parent id.
	ErrSelfParentingNotAllowed = errors.New("self parenting not allowed")

	// ErrInvalidChildGroupID indicates invalid child group ID.
	ErrInvalidChildGroupID = errors.New("invalid child group id")

	// ErrInvalidAuthKey indicates invalid auth key.
	ErrInvalidAuthKey = errors.New("invalid auth key")

	// ErrInvalidIDFormat indicates an invalid ID format.
	ErrInvalidIDFormat = errors.New("invalid id format provided")

	// ErrNameSize indicates that name size exceeds the max.
	ErrNameSize = errors.New("invalid name size")

	// ErrEmailSize indicates that email size exceeds the max.
	ErrEmailSize = errors.New("invalid email size")

	// ErrInvalidRole indicates that an invalid role.
	ErrInvalidRole = errors.New("invalid client role")

	// ErrLimitSize indicates that an invalid limit.
	ErrLimitSize = errors.New("invalid limit size")

	// ErrLevel indicates that an invalid level.
	ErrLevel = errors.New("invalid level")

	// ErrOffsetSize indicates an invalid offset.
	ErrOffsetSize = errors.New("invalid offset size")

	// ErrInvalidOrder indicates an invalid list order.
	ErrInvalidOrder = errors.New("invalid list order provided")

	// ErrInvalidDirection indicates an invalid list direction.
	ErrInvalidDirection = errors.New("invalid list direction provided")

	// ErrInvalidMemberKind indicates an invalid member kind.
	ErrInvalidMemberKind = errors.New("invalid member kind")

	// ErrEmptyList indicates that entity data is empty.
	ErrEmptyList = errors.New("empty list provided")

	// ErrMissingRoleName indicates that role name are empty
	ErrMissingRoleName = errors.New("empty role name")

	// ErrMissingRoleOperations indicates that role operations are empty
	ErrMissingRoleOperations = errors.New("empty role operations")

	// ErrMissingRoleMembers indicates that role members are empty
	ErrMissingRoleMembers = errors.New("empty role members")

	// ErrMalformedPolicy indicates that policies are malformed.
	ErrMalformedPolicy = errors.New("malformed policy")

	// ErrMissingPolicySub indicates that policies are subject.
	ErrMissingPolicySub = errors.New("malformed policy subject")

	// ErrMissingPolicyObj indicates missing policies object.
	ErrMissingPolicyObj = errors.New("malformed policy object")

	// ErrMalformedPolicyAct indicates missing policies action.
	ErrMalformedPolicyAct = errors.New("malformed policy action")

	// ErrMissingPolicyEntityType indicates missing policies entity type.
	ErrMissingPolicyEntityType = errors.New("missing policy entity type")

	// ErrMalformedPolicyPer indicates missing policies relation.
	ErrMalformedPolicyPer = errors.New("malformed policy permission")

	// ErrMissingCertData indicates missing cert data (ttl).
	ErrMissingCertData = errors.New("missing certificate data")

	// ErrInvalidCertData indicates invalid cert data (ttl).
	ErrInvalidCertData = errors.New("invalid certificate data")

	// ErrInvalidTopic indicates an invalid subscription topic.
	ErrInvalidTopic = errors.New("invalid Subscription topic")

	// ErrInvalidContact indicates an invalid subscription contract.
	ErrInvalidContact = errors.New("invalid Subscription contact")

	// ErrMissingEmail indicates missing email.
	ErrMissingEmail = errors.New("missing email")

	// ErrInvalidEmail indicates missing email.
	ErrInvalidEmail = errors.New("invalid email")

	// ErrMissingHost indicates missing host.
	ErrMissingHost = errors.New("missing host")

	// ErrMissingPass indicates missing password.
	ErrMissingPass = errors.New("missing password")

	// ErrMissingConfPass indicates missing conf password.
	ErrMissingConfPass = errors.New("missing conf password")

	// ErrInvalidResetPass indicates an invalid reset password.
	ErrInvalidResetPass = errors.New("invalid reset password")

	// ErrInvalidComparator indicates an invalid comparator.
	ErrInvalidComparator = errors.New("invalid comparator")

	// ErrMissingMemberType indicates missing group member type.
	ErrMissingMemberType = errors.New("missing group member type")

	// ErrMissingMemberKind indicates missing group member kind.
	ErrMissingMemberKind = errors.New("missing group member kind")

	// ErrMissingRelation indicates missing relation.
	ErrMissingRelation = errors.New("missing relation")

	// ErrInvalidRelation indicates an invalid relation.
	ErrInvalidRelation = errors.New("invalid relation")

	// ErrInvalidAPIKey indicates an invalid API key type.
	ErrInvalidAPIKey = errors.New("invalid api key type")

	// ErrBootstrapState indicates an invalid bootstrap state.
	ErrBootstrapState = errors.New("invalid bootstrap state")

	// ErrInvitationState indicates an invalid invitation state.
	ErrInvitationState = errors.New("invalid invitation state")

	// ErrMissingIdentity indicates missing entity Identity.
	ErrMissingIdentity = errors.New("missing entity identity")

	// ErrMissingSecret indicates missing secret.
	ErrMissingSecret = errors.New("missing secret")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.New("password does not meet the requirements")

	// ErrMissingName indicates missing identity name.
	ErrMissingName = errors.New("missing identity name")

	// ErrMissingName indicates missing alias.
	ErrMissingAlias = errors.New("missing alias")

	// ErrInvalidLevel indicates an invalid group level.
	ErrInvalidLevel = errors.New("invalid group level (should be between 0 and 5)")

	// ErrNotFoundParam indicates that the parameter was not found in the query.
	ErrNotFoundParam = errors.New("parameter not found in the query")

	// ErrInvalidQueryParams indicates invalid query parameters.
	ErrInvalidQueryParams = errors.New("invalid query parameters")

	// ErrInvalidVisibilityType indicates invalid visibility type.
	ErrInvalidVisibilityType = errors.New("invalid visibility type")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type.
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrRollbackTx indicates failed to rollback transaction.
	ErrRollbackTx = errors.New("failed to rollback transaction")

	// ErrInvalidAggregation indicates invalid aggregation value.
	ErrInvalidAggregation = errors.New("invalid aggregation value")

	// ErrInvalidInterval indicates invalid interval value.
	ErrInvalidInterval = errors.New("invalid interval value")

	// ErrMissingFrom indicates missing from value.
	ErrMissingFrom = errors.New("missing from time value")

	// ErrMissingTo indicates missing to value.
	ErrMissingTo = errors.New("missing to time value")

	// ErrEmptyMessage indicates empty message.
	ErrEmptyMessage = errors.New("empty message")

	// ErrMissingEntityType indicates missing entity type.
	ErrMissingEntityType = errors.New("missing entity type")

	// ErrInvalidEntityType indicates invalid entity type.
	ErrInvalidEntityType = errors.New("invalid entity type")

	// ErrInvalidTimeFormat indicates invalid time format i.e not unix time.
	ErrInvalidTimeFormat = errors.New("invalid time format use unix time")

	// ErrEmptySearchQuery indicates search query should not be empty.
	ErrEmptySearchQuery = errors.New("search query must not be empty")

	// ErrLenSearchQuery indicates search query length.
	ErrLenSearchQuery = errors.New("search query must be at least 3 characters")

	// ErrMissingDomainID indicates missing domainID.
	ErrMissingDomainID = errors.New("missing domainID")

	// ErrMissingUsername indicates missing user name.
	ErrMissingUsername = errors.New("missing username")

	// ErrInvalidUsername indicates missing user name.
	ErrInvalidUsername = errors.New("invalid username")

	// ErrMissingFirstName indicates missing first name.
	ErrMissingFirstName = errors.New("missing first name")

	// ErrMissingLastName indicates missing last name.
	ErrMissingLastName = errors.New("missing last name")

	// ErrInvalidProfilePictureURL indicates that the profile picture url is invalid.
	ErrInvalidProfilePictureURL = errors.New("invalid profile picture url")

	ErrMultipleEntitiesFilter = errors.New("multiple entities are provided in filter are not supported")
)
