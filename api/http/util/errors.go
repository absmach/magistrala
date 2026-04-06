// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package util

import "github.com/absmach/magistrala/pkg/errors"

// Errors defined in this file are used by the LoggingErrorEncoder decorator
// to distinguish and log API request validation errors and avoid that service
// errors are logged twice.
var (
	// ErrValidation indicates that an error was returned by the API.
	ErrValidation = errors.NewRequestError("something went wrong with the request")

	// ErrBearerToken indicates missing or invalid bearer user token.
	ErrBearerToken = errors.NewAuthNError("missing or invalid bearer user token")

	// ErrBearerKey indicates missing or invalid bearer entity key.
	ErrBearerKey = errors.NewAuthNError("missing or invalid bearer entity key")

	// ErrMissingID indicates missing entity ID.
	ErrMissingID = errors.NewRequestError("missing entity id")

	// ErrMissingEntityID indicates missing entity ID.
	ErrMissingEntityID = errors.NewRequestError("missing entity id")

	// ErrMissingClientID indicates missing client ID.
	ErrMissingClientID = errors.NewRequestError("missing client id")

	// ErrMissingChannelID indicates missing client ID.
	ErrMissingChannelID = errors.NewRequestError("missing channel id")

	// ErrMissingConnectionType indicates missing connection tpye.
	ErrMissingConnectionType = errors.NewRequestError("missing connection type")

	// ErrMissingParentGroupID indicates missing parent group ID.
	ErrMissingParentGroupID = errors.NewRequestError("missing parent group id")

	// ErrMissingChildrenGroupIDs indicates missing children group IDs.
	ErrMissingChildrenGroupIDs = errors.NewRequestError("missing children group ids")

	// ErrSelfParentingNotAllowed indicates child id is same as parent id.
	ErrSelfParentingNotAllowed = errors.NewRequestError("self parenting not allowed")

	// ErrInvalidChildGroupID indicates invalid child group ID.
	ErrInvalidChildGroupID = errors.NewRequestError("invalid child group id")

	// ErrInvalidAuthKey indicates invalid auth key.
	ErrInvalidAuthKey = errors.New("invalid auth key")

	// ErrInvalidIDFormat indicates an invalid ID format.
	ErrInvalidIDFormat = errors.NewRequestError("invalid id format provided")

	// ErrNameSize indicates that name size exceeds the max.
	ErrNameSize = errors.NewRequestError("invalid name size")

	// ErrEmailSize indicates that email size exceeds the max.
	ErrEmailSize = errors.NewRequestError("invalid email size")

	// ErrInvalidRole indicates that an invalid role.
	ErrInvalidRole = errors.NewRequestError("invalid client role")

	// ErrLimitSize indicates that an invalid limit.
	ErrLimitSize = errors.NewRequestError("invalid limit size")

	// ErrLevel indicates that an invalid level.
	ErrLevel = errors.NewRequestError("invalid level")

	// ErrOffsetSize indicates an invalid offset.
	ErrOffsetSize = errors.NewRequestError("invalid offset size")

	// ErrInvalidOrder indicates an invalid list order.
	ErrInvalidOrder = errors.NewRequestError("invalid list order provided")

	// ErrInvalidDirection indicates an invalid list direction.
	ErrInvalidDirection = errors.NewRequestError("invalid list direction provided")

	// ErrInvalidMemberKind indicates an invalid member kind.
	ErrInvalidMemberKind = errors.NewRequestError("invalid member kind")

	// ErrEmptyList indicates that entity data is empty.
	ErrEmptyList = errors.NewRequestError("empty list provided")

	// ErrMissingRoleName indicates that role name is empty.
	ErrMissingRoleName = errors.NewRequestError("empty role name")

	// ErrMissingRoleID indicates that role id is empty.
	ErrMissingRoleID = errors.NewRequestError("empty role id")

	// ErrMissingRoleOperations indicates that role operations are empty.
	ErrMissingRoleOperations = errors.NewRequestError("empty role operations")

	// ErrMissingRoleMembers indicates that role members are empty.
	ErrMissingRoleMembers = errors.NewRequestError("empty role members")

	// ErrMalformedPolicy indicates that policies are malformed.
	ErrMalformedPolicy = errors.NewRequestError("malformed policy")

	// ErrMissingPolicySub indicates that policies are subject.
	ErrMissingPolicySub = errors.NewRequestError("malformed policy subject")

	// ErrMissingPolicyObj indicates missing policies object.
	ErrMissingPolicyObj = errors.NewRequestError("malformed policy object")

	// ErrMalformedPolicyAct indicates missing policies action.
	ErrMalformedPolicyAct = errors.NewRequestError("malformed policy action")

	// ErrMissingPolicyEntityType indicates missing policies entity type.
	ErrMissingPolicyEntityType = errors.NewRequestError("missing policy entity type")

	// ErrMalformedPolicyPer indicates missing policies relation.
	ErrMalformedPolicyPer = errors.NewRequestError("malformed policy permission")

	// ErrMissingCertData indicates missing cert data (ttl).
	ErrMissingCertData = errors.NewRequestError("missing certificate data")

	// ErrInvalidCertData indicates invalid cert data (ttl).
	ErrInvalidCertData = errors.NewRequestError("invalid certificate data")

	// ErrInvalidTopic indicates an invalid subscription topic.
	ErrInvalidTopic = errors.NewRequestError("invalid Subscription topic")

	// ErrInvalidContact indicates an invalid subscription contract.
	ErrInvalidContact = errors.NewRequestError("invalid Subscription contact")

	// ErrMissingEmail indicates missing email.
	ErrMissingEmail = errors.NewRequestError("missing email")

	// ErrInvalidEmail indicates missing email.
	ErrInvalidEmail = errors.NewRequestError("invalid email")

	// ErrMissingHost indicates missing host.
	ErrMissingHost = errors.NewRequestError("missing host")

	// ErrMissingPass indicates missing password.
	ErrMissingPass = errors.NewRequestError("missing password")

	// ErrMissingConfPass indicates missing conf password.
	ErrMissingConfPass = errors.NewRequestError("missing conf password")

	// ErrInvalidResetPass indicates an invalid reset password.
	ErrInvalidResetPass = errors.NewRequestError("invalid reset password")

	// ErrInvalidComparator indicates an invalid comparator.
	ErrInvalidComparator = errors.NewRequestError("invalid comparator")

	// ErrMissingMemberIDs indicates missing member ids.
	ErrMissingMemberIDs = errors.NewRequestError("missing member ids")

	// ErrMissingMemberType indicates missing group member type.
	ErrMissingMemberType = errors.NewRequestError("missing group member type")

	// ErrMissingMemberKind indicates missing group member kind.
	ErrMissingMemberKind = errors.NewRequestError("missing group member kind")

	// ErrMissingRelation indicates missing relation.
	ErrMissingRelation = errors.NewRequestError("missing relation")

	// ErrInvalidRelation indicates an invalid relation.
	ErrInvalidRelation = errors.NewRequestError("invalid relation")

	// ErrInvalidAPIKey indicates an invalid API key type.
	ErrInvalidAPIKey = errors.NewRequestError("invalid api key type")

	// ErrInvitationState indicates an invalid invitation state.
	ErrInvitationState = errors.NewRequestError("invalid invitation state")

	// ErrMissingIdentity indicates missing entity Identity.
	ErrMissingIdentity = errors.NewRequestError("missing entity identity")

	// ErrMissingSecret indicates missing secret.
	ErrMissingSecret = errors.NewRequestError("missing secret")

	// ErrPasswordFormat indicates weak password.
	ErrPasswordFormat = errors.NewRequestError("password does not meet the requirements")

	// ErrMissingName indicates missing identity name.
	ErrMissingName = errors.NewRequestError("missing identity name")

	// ErrMissingRoute indicates missing route.
	ErrMissingRoute = errors.NewRequestError("missing route")

	// ErrInvalidLevel indicates an invalid group level.
	ErrInvalidLevel = errors.NewRequestError("invalid group level (should be between 0 and 5)")

	// ErrNotFoundParam indicates that the parameter was not found in the query.
	ErrNotFoundParam = errors.NewRequestError("parameter not found in the query")

	// ErrInvalidQueryParams indicates invalid query parameters.
	ErrInvalidQueryParams = errors.NewRequestError("invalid query parameters")

	// ErrInvalidVisibilityType indicates invalid visibility type.
	ErrInvalidVisibilityType = errors.NewRequestError("invalid visibility type")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type.
	ErrUnsupportedContentType = errors.NewMediaTypeError("unsupported content type")

	// ErrRollbackTx indicates failed to rollback transaction.
	ErrRollbackTx = errors.NewRequestError("failed to rollback transaction")

	// ErrInvalidAggregation indicates invalid aggregation value.
	ErrInvalidAggregation = errors.NewRequestError("invalid aggregation value")

	// ErrInvalidInterval indicates invalid interval value.
	ErrInvalidInterval = errors.NewRequestError("invalid interval value")

	// ErrMissingFrom indicates missing from value.
	ErrMissingFrom = errors.NewRequestError("missing from time value")

	// ErrMissingTo indicates missing to value.
	ErrMissingTo = errors.NewRequestError("missing to time value")

	// ErrEmptyMessage indicates empty message.
	ErrEmptyMessage = errors.NewRequestError("empty message")

	// ErrMissingEntityType indicates missing entity type.
	ErrMissingEntityType = errors.NewRequestError("missing entity type")

	// ErrInvalidEntityType indicates invalid entity type.
	ErrInvalidEntityType = errors.NewRequestError("invalid entity type")

	// ErrInvalidTimeFormat indicates invalid time format i.e not unix time.
	ErrInvalidTimeFormat = errors.NewRequestError("invalid time format use unix time")

	// ErrEmptySearchQuery indicates search query should not be empty.
	ErrEmptySearchQuery = errors.NewRequestError("search query must not be empty")

	// ErrLenSearchQuery indicates search query length.
	ErrLenSearchQuery = errors.NewRequestError("search query must be at least 3 characters")

	// ErrMissingDomainID indicates missing domainID.
	ErrMissingDomainID = errors.NewRequestError("missing domainID")

	// ErrMissingUsername indicates missing user name.
	ErrMissingUsername = errors.NewRequestError("missing username")

	// ErrInvalidUsername indicates invalid user name.
	ErrInvalidUsername = errors.NewRequestError("invalid username")

	// ErrMissingFirstName indicates missing first name.
	ErrMissingFirstName = errors.NewRequestError("missing first name")

	// ErrMissingLastName indicates missing last name.
	ErrMissingLastName = errors.NewRequestError("missing last name")

	// ErrInvalidProfilePictureURL indicates that the profile picture url is invalid.
	ErrInvalidProfilePictureURL = errors.NewRequestError("invalid profile picture url")

	ErrMultipleEntitiesFilter = errors.NewRequestError("multiple entities are provided in filter are not supported")

	// ErrMissingDescription indicates missing description.
	ErrMissingDescription = errors.NewRequestError("missing description")

	// ErrUnsupportedTokenType indicates that this type of token is not supported.
	ErrUnsupportedTokenType = errors.NewRequestError("unsupported content token type")

	// ErrMissingUserID indicates missing user ID.
	ErrMissingUserID = errors.NewRequestError("missing user id")

	// ErrMissingPATID indicates missing pat ID.
	ErrMissingPATID = errors.NewRequestError("missing pat id")

	// ErrInvalidNameFormat indicates invalid name format.
	ErrInvalidNameFormat = errors.NewRequestError("invalid name format")

	// ErrInvalidRouteFormat indicates invalid route format.
	ErrInvalidRouteFormat = errors.NewRequestError("invalid route format")

	// ErrMissingUsernameEmail indicates missing user name / email.
	ErrMissingUsernameEmail = errors.NewRequestError("missing username / email")

	// ErrInvalidVerification indicates invalid email verification.
	ErrInvalidVerification = errors.NewRequestError("invalid verification")

	// ErrEmailNotVerified indicates invalid email not verified.
	ErrEmailNotVerified = errors.NewRequestError("email not verified")

	// ErrMalformedRequest indicates malformed request body.
	ErrMalformedRequestBody = errors.NewRequestError("request body is not a valid JSON, expecting a valid JSON")
)
