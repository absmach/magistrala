// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/gofrs/uuid"
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
	IdentityKey      = "identity"
	GroupKey         = "group"
	ActionKey        = "action"
	TagKey           = "tag"
	NameKey          = "name"
	TotalKey         = "total"
	SubjectKey       = "subject"
	ObjectKey        = "object"
	LevelKey         = "level"
	TreeKey          = "tree"
	DirKey           = "dir"
	ListPerms        = "list_perms"
	VisibilityKey    = "visibility"
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
	case errors.Contains(err, svcerr.ErrMalformedEntity):
		err = svcerr.ErrMalformedEntity
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, errors.ErrMalformedEntity):
		err = errors.ErrMalformedEntity
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingID):
		err = apiutil.ErrMissingID
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrEmptyList):
		err = apiutil.ErrEmptyList
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingMemberType):
		err = apiutil.ErrMissingMemberType
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingMemberKind):
		err = apiutil.ErrMissingMemberKind
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrLimitSize):
		err = apiutil.ErrLimitSize
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrBearerKey):
		err = apiutil.ErrBearerKey
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrNameSize):
		err = apiutil.ErrNameSize
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrInvalidStatus):
		err = svcerr.ErrInvalidStatus
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrInvalidStatus):
		err = apiutil.ErrInvalidStatus
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrInvalidIDFormat):
		err = apiutil.ErrInvalidIDFormat
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrInvalidQueryParams):
		err = apiutil.ErrInvalidQueryParams
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingRelation):
		err = apiutil.ErrMissingRelation
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrValidation):
		err = apiutil.ErrValidation
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingIdentity):
		err = apiutil.ErrMissingIdentity
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingSecret):
		err = apiutil.ErrMissingSecret
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingPass):
		err = apiutil.ErrMissingPass
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrMissingConfPass):
		err = apiutil.ErrMissingConfPass
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, apiutil.ErrPasswordFormat):
		err = apiutil.ErrPasswordFormat
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrAuthentication):
		err = svcerr.ErrAuthentication
		w.WriteHeader(http.StatusUnauthorized)

	case errors.Contains(err, apiutil.ErrBearerToken):
		err = apiutil.ErrBearerToken
		w.WriteHeader(http.StatusUnauthorized)

	case errors.Contains(err, svcerr.ErrNotFound):
		err = svcerr.ErrNotFound
		w.WriteHeader(http.StatusNotFound)

	case errors.Contains(err, postgres.ErrMemberAlreadyAssigned):
		err = postgres.ErrMemberAlreadyAssigned
		w.WriteHeader(http.StatusConflict)

	case errors.Contains(err, svcerr.ErrConflict):
		err = svcerr.ErrConflict
		w.WriteHeader(http.StatusConflict)

	case errors.Contains(err, svcerr.ErrAuthorization):
		err = svcerr.ErrAuthorization
		w.WriteHeader(http.StatusForbidden)

	case errors.Contains(err, svcerr.ErrDomainAuthorization):
		err = svcerr.ErrDomainAuthorization
		w.WriteHeader(http.StatusForbidden)

	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		err = apiutil.ErrUnsupportedContentType
		w.WriteHeader(http.StatusUnsupportedMediaType)

	case errors.Contains(err, svcerr.ErrCreateEntity):
		err = svcerr.ErrCreateEntity
		w.WriteHeader(http.StatusInternalServerError)

	case errors.Contains(err, svcerr.ErrUpdateEntity):
		err = svcerr.ErrUpdateEntity
		w.WriteHeader(http.StatusInternalServerError)

	case errors.Contains(err, svcerr.ErrViewEntity):
		err = svcerr.ErrViewEntity
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrRemoveEntity):
		err = svcerr.ErrRemoveEntity
		w.WriteHeader(http.StatusInternalServerError)

	case errors.Contains(err, svcerr.ErrLogin):
		err = svcerr.ErrLogin
		w.WriteHeader(http.StatusUnauthorized)

	case errors.Contains(err, svcerr.ErrInvalidRole):
		err = svcerr.ErrInvalidRole
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrInvalidPolicy):
		err = svcerr.ErrInvalidPolicy
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, svcerr.ErrEnableClient):
		err = mgclients.ErrEnableClient
		w.WriteHeader(http.StatusNotFound)

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
