// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/absmach/supermq"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/users"
	"github.com/gofrs/uuid/v5"
)

const (
	OffsetKey = "offset"
	DirKey    = "dir"
	OrderKey  = "order"
	LimitKey  = "limit"

	NameOrder = "name"
	IDOrder   = "id"
	AscDir    = "asc"
	DescDir   = "desc"

	MetadataKey = "metadata"
	NameKey     = "name"
	TagKey      = "tag"
	StatusKey   = "status"

	ClientKey   = "client"
	ChannelKey  = "channel"
	ConnTypeKey = "connection_type"
	GroupKey    = "group"
	DomainKey   = "domain"

	StartLevelKey = "start_level"
	EndLevelKey   = "end_level"
	TreeKey       = "tree"
	ParentKey     = "parent_id"
	LevelKey      = "level"
	RootGroupKey  = "root_group"

	TokenKey   = "token"
	SubjectKey = "subject"
	ObjectKey  = "object"

	ActionKey           = "action"
	ActionsKey          = "actions"
	RoleIDKey           = "role_id"
	RoleNameKey         = "role_name"
	AccessProviderIDKey = "access_provider_id"
	AccessTypeKey       = "access_type"

	UsernameKey  = "username"
	UserKey      = "user"
	EmailKey     = "email"
	FirstNameKey = "first_name"
	LastNameKey  = "last_name"

	DefTotal        = uint64(100)
	DefOffset       = 0
	DefOrder        = "updated_at"
	DefDir          = "asc"
	DefLimit        = 10
	DefLevel        = 0
	DefStartLevel   = 1
	DefEndLevel     = 0
	DefStatus       = "enabled"
	DefClientStatus = clients.Enabled
	DefUserStatus   = users.Enabled
	DefGroupStatus  = groups.Enabled

	// ContentType represents JSON content type.
	ContentType = "application/json"

	// MaxNameSize limits name size to prevent making them too complex.
	MaxLimitSize = 100
	MaxNameSize  = 1024
	MaxIDSize    = 36
)

var (
	nameRegExp        = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{34}[a-z0-9]$`)
	errUnreadableName = errors.New("name containing double underscores or double dashes not allowed")
)

// ValidateUUID validates UUID format.
func ValidateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return apiutil.ErrInvalidIDFormat
	}

	return nil
}

// ValidateName validates name format.
func ValidateName(id string) error {
	if !nameRegExp.MatchString(id) {
		return apiutil.ErrInvalidNameFormat
	}
	// Names containing double underscores or double dashes are invalid due to similarity concerns.
	if strings.Contains(id, "__") || strings.Contains(id, "--") {
		return errors.Wrap(apiutil.ErrInvalidNameFormat, errUnreadableName)
	}

	return nil
}

// EncodeResponse encodes successful response.
func EncodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(supermq.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", ContentType)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

// EncodeError encodes an error response.
func EncodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	w.Header().Set("Content-Type", ContentType)
	switch {
	case errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, svcerr.ErrDomainAuthorization),
		errors.Contains(err, svcerr.ErrUnauthorizedPAT):
		err = unwrap(err)
		w.WriteHeader(http.StatusForbidden)

	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken),
		errors.Contains(err, svcerr.ErrLogin),
		errors.Contains(err, apiutil.ErrUnsupportedTokenType):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMalformedPolicy),
		errors.Contains(err, apiutil.ErrMissingSecret),
		errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrMissingName),
		errors.Contains(err, apiutil.ErrMissingAlias),
		errors.Contains(err, apiutil.ErrMissingEmail),
		errors.Contains(err, apiutil.ErrInvalidEmail),
		errors.Contains(err, apiutil.ErrMissingHost),
		errors.Contains(err, apiutil.ErrInvalidResetPass),
		errors.Contains(err, apiutil.ErrEmptyList),
		errors.Contains(err, apiutil.ErrMissingMemberKind),
		errors.Contains(err, apiutil.ErrMissingMemberType),
		errors.Contains(err, apiutil.ErrLimitSize),
		errors.Contains(err, apiutil.ErrBearerKey),
		errors.Contains(err, svcerr.ErrInvalidStatus),
		errors.Contains(err, apiutil.ErrNameSize),
		errors.Contains(err, apiutil.ErrInvalidIDFormat),
		errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, apiutil.ErrMissingRelation),
		errors.Contains(err, apiutil.ErrValidation),
		errors.Contains(err, apiutil.ErrMissingPass),
		errors.Contains(err, apiutil.ErrMissingConfPass),
		errors.Contains(err, apiutil.ErrPasswordFormat),
		errors.Contains(err, svcerr.ErrInvalidRole),
		errors.Contains(err, svcerr.ErrInvalidPolicy),
		errors.Contains(err, apiutil.ErrInvitationState),
		errors.Contains(err, apiutil.ErrInvalidAPIKey),
		errors.Contains(err, svcerr.ErrViewEntity),
		errors.Contains(err, apiutil.ErrMissingCertData),
		errors.Contains(err, apiutil.ErrInvalidContact),
		errors.Contains(err, apiutil.ErrInvalidTopic),
		errors.Contains(err, apiutil.ErrInvalidCertData),
		errors.Contains(err, apiutil.ErrEmptyMessage),
		errors.Contains(err, apiutil.ErrInvalidLevel),
		errors.Contains(err, apiutil.ErrInvalidDirection),
		errors.Contains(err, apiutil.ErrInvalidEntityType),
		errors.Contains(err, apiutil.ErrMissingEntityType),
		errors.Contains(err, apiutil.ErrInvalidTimeFormat),
		errors.Contains(err, svcerr.ErrSearch),
		errors.Contains(err, apiutil.ErrEmptySearchQuery),
		errors.Contains(err, apiutil.ErrLenSearchQuery),
		errors.Contains(err, apiutil.ErrMissingDomainID),
		errors.Contains(err, certs.ErrFailedReadFromPKI),
		errors.Contains(err, apiutil.ErrMissingUserID),
		errors.Contains(err, apiutil.ErrMissingPATID),
		errors.Contains(err, apiutil.ErrMissingUsername),
		errors.Contains(err, apiutil.ErrMissingFirstName),
		errors.Contains(err, apiutil.ErrMissingLastName),
		errors.Contains(err, apiutil.ErrInvalidUsername),
		errors.Contains(err, apiutil.ErrMissingIdentity),
		errors.Contains(err, apiutil.ErrInvalidProfilePictureURL),
		errors.Contains(err, apiutil.ErrSelfParentingNotAllowed),
		errors.Contains(err, apiutil.ErrMissingChildrenGroupIDs),
		errors.Contains(err, apiutil.ErrMissingParentGroupID),
		errors.Contains(err, apiutil.ErrMissingConnectionType),
		errors.Contains(err, apiutil.ErrMissingRoleName),
		errors.Contains(err, apiutil.ErrMissingRoleID),
		errors.Contains(err, apiutil.ErrMissingPolicyEntityType),
		errors.Contains(err, apiutil.ErrMissingRoleMembers),
		errors.Contains(err, apiutil.ErrMissingDescription),
		errors.Contains(err, apiutil.ErrMissingEntityID):
		err = unwrap(err)
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrCreateEntity),
		errors.Contains(err, svcerr.ErrUpdateEntity),
		errors.Contains(err, svcerr.ErrRemoveEntity),
		errors.Contains(err, svcerr.ErrEnableClient),
		errors.Contains(err, svcerr.ErrEnableUser),
		errors.Contains(err, svcerr.ErrDisableUser):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnprocessableEntity)

	case errors.Contains(err, svcerr.ErrNotFound):
		err = unwrap(err)
		w.WriteHeader(http.StatusNotFound)

	case errors.Contains(err, errors.ErrStatusAlreadyAssigned),
		errors.Contains(err, svcerr.ErrInvitationAlreadyRejected),
		errors.Contains(err, svcerr.ErrInvitationAlreadyAccepted),
		errors.Contains(err, svcerr.ErrConflict):
		err = unwrap(err)
		w.WriteHeader(http.StatusConflict)

	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnsupportedMediaType)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func unwrap(err error) error {
	wrapper, err := errors.Unwrap(err)
	if wrapper != nil {
		return wrapper
	}
	return err
}
