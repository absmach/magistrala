// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/gofrs/uuid/v5"
)

const (
	MemberKindKey    = "member_kind"
	PermissionKey    = "permission"
	RelationKey      = "relation"
	StatusKey        = "status"
	OffsetKey        = "offset"
	OrderKey         = "order"
	LimitKey         = "limit"
	MetadataKey      = "metadata"
	ParentKey        = "parent_id"
	OwnerKey         = "owner_id"
	ClientKey        = "client"
	UsernameKey      = "username"
	NameKey          = "name"
	GroupKey         = "group"
	ActionKey        = "action"
	TagKey           = "tag"
	FirstNameKey     = "first_name"
	LastNameKey      = "last_name"
	TotalKey         = "total"
	SubjectKey       = "subject"
	ObjectKey        = "object"
	LevelKey         = "level"
	TreeKey          = "tree"
	DirKey           = "dir"
	ListPerms        = "list_perms"
	VisibilityKey    = "visibility"
	EmailKey         = "email"
	SharedByKey      = "shared_by"
	TokenKey         = "token"
	DefPermission    = "view"
	DefTotal         = uint64(100)
	DefOffset        = 0
	DefOrder         = "updated_at"
	DefDir           = "asc"
	DefLimit         = 10
	DefLevel         = 0
	DefStatus        = "enabled"
	DefClientStatus  = mgclients.Enabled
	DefGroupStatus   = mgclients.Enabled
	DefListPerms     = false
	SharedVisibility = "shared"
	MyVisibility     = "mine"
	AllVisibility    = "all"
	// ContentType represents JSON content type.
	ContentType = "application/json"

	// MaxNameSize limits name size to prevent making them too complex.
	MaxLimitSize = 100
	MaxNameSize  = 1024
	NameOrder    = "name"
	IDOrder      = "id"
	AscDir       = "asc"
	DescDir      = "desc"
)

// ValidateUUID validates UUID format.
func ValidateUUID(extID string) (err error) {
	id, err := uuid.FromString(extID)
	if id.String() != extID || err != nil {
		return apiutil.ErrInvalidIDFormat
	}

	return nil
}

// EncodeResponse encodes successful response.
func EncodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(magistrala.Response); ok {
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
		errors.Contains(err, bootstrap.ErrExternalKey),
		errors.Contains(err, bootstrap.ErrExternalKeySecure):
		err = unwrap(err)
		w.WriteHeader(http.StatusForbidden)

	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken),
		errors.Contains(err, svcerr.ErrLogin):
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
		errors.Contains(err, apiutil.ErrBootstrapState),
		errors.Contains(err, apiutil.ErrMissingCertData),
		errors.Contains(err, apiutil.ErrInvalidContact),
		errors.Contains(err, apiutil.ErrInvalidTopic),
		errors.Contains(err, bootstrap.ErrAddBootstrap),
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
		errors.Contains(err, apiutil.ErrMissingUsername):
		err = unwrap(err)
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrCreateEntity),
		errors.Contains(err, svcerr.ErrUpdateEntity),
		errors.Contains(err, svcerr.ErrRemoveEntity),
		errors.Contains(err, svcerr.ErrEnableClient):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnprocessableEntity)

	case errors.Contains(err, svcerr.ErrNotFound),
		errors.Contains(err, bootstrap.ErrBootstrap):
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
