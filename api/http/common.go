// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"regexp"
	"strings"

	"github.com/absmach/magistrala"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/users"
	"github.com/gofrs/uuid/v5"
)

const (
	OffsetKey = "offset"
	DirKey    = "dir"
	OrderKey  = "order"
	LimitKey  = "limit"
	OnlyTotal = "only_total"

	NameOrder      = "name"
	IDOrder        = "id"
	AscDir         = "asc"
	DescDir        = "desc"
	UpdatedAtOrder = "updated_at"
	CreatedAtOrder = "created_at"

	MetadataKey = "metadata"
	NameKey     = "name"
	TagKey      = "tag"
	TagsKey     = "tags"
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

	RolesKey            = "roles"
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
	DefDir          = "desc"
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
	routeRegExp       = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,35}$`)
	userNameRegExp    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,31}$`)
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

func ValidateEmail(email string) (err error) {
	if _, err := mail.ParseAddress(email); err != nil {
		return apiutil.ErrInvalidEmail
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

// ValidateRoute validates route format.
func ValidateRoute(route string) error {
	if !routeRegExp.MatchString(route) {
		return apiutil.ErrInvalidRouteFormat
	}

	if strings.Contains(route, "__") || strings.Contains(route, "--") {
		return errors.Wrap(apiutil.ErrInvalidRouteFormat, errUnreadableName)
	}

	return nil
}

// ValidateUserName validates user name format.
func ValidateUserName(name string) error {
	if !userNameRegExp.MatchString(name) {
		return apiutil.ErrInvalidUsername
	}

	if strings.Contains(name, "__") || strings.Contains(name, "--") {
		return errors.Wrap(apiutil.ErrInvalidUsername, errUnreadableName)
	}

	return nil
}

// EncodeResponse encodes successful response.
func EncodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
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
	w.Header().Set("Content-Type", ContentType)
	if sdkErr, ok := err.(errors.SDKError); ok {
		w.WriteHeader(sdkErr.StatusCode())
		if err := json.NewEncoder(w).Encode(sdkErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	switch retErr := err.(type) {
	case *errors.RequestError:
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.AuthNError:
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.AuthZError:
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.MediaTypeError:
		w.WriteHeader(http.StatusUnsupportedMediaType)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.ServiceError:
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.NotFoundError:
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(retErr); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	case *errors.InternalError:
		w.WriteHeader(http.StatusInternalServerError)
		return
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
