// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	upolicies "github.com/mainflux/mainflux/users/policies"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType    = "application/json"
	offsetKey      = "offset"
	limitKey       = "limit"
	formatKey      = "format"
	subtopicKey    = "subtopic"
	publisherKey   = "publisher"
	protocolKey    = "protocol"
	nameKey        = "name"
	valueKey       = "v"
	stringValueKey = "vs"
	dataValueKey   = "vd"
	boolValueKey   = "vb"
	comparatorKey  = "comparator"
	fromKey        = "from"
	toKey          = "to"
	defLimit       = 10
	defOffset      = 0
	defFormat      = "messages"
)

var (
	errThingAccess = errors.New("thing has no permission")
	errUserAccess  = errors.New("user has no permission")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, tc tpolicies.AuthServiceClient, ac upolicies.AuthServiceClient, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()
	mux.Get("/channels/:chanID/messages", kithttp.NewServer(
		listMessagesEndpoint(svc, tc, ac),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/health", mainflux.Health(svcName, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	limit, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	format, err := apiutil.ReadStringQuery(r, formatKey, defFormat)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	name, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	v, err := apiutil.ReadFloatQuery(r, valueKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	comparator, err := apiutil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vs, err := apiutil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vd, err := apiutil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return nil, err
	}

	from, err := apiutil.ReadFloatQuery(r, fromKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	to, err := apiutil.ReadFloatQuery(r, toKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listMessagesReq{
		chanID: bone.GetValue(r, "chanID"),
		token:  apiutil.ExtractBearerToken(r),
		key:    apiutil.ExtractThingKey(r),
		pageMeta: readers.PageMetadata{
			Offset:      offset,
			Limit:       limit,
			Format:      format,
			Subtopic:    subtopic,
			Publisher:   publisher,
			Protocol:    protocol,
			Name:        name,
			Value:       v,
			Comparator:  comparator,
			StringValue: vs,
			DataValue:   vd,
			BoolValue:   vb,
			From:        from,
			To:          to,
		},
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, nil):
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrLimitSize),
		errors.Contains(err, apiutil.ErrOffsetSize),
		errors.Contains(err, apiutil.ErrInvalidComparator):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, readers.ErrReadMessages):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)

		errMsg := errorVal.Msg()
		if errorVal.Err() != nil {
			errMsg = fmt.Sprintf("%s : %s", errMsg, errorVal.Err().Msg())
		}

		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errMsg}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func authorize(ctx context.Context, req listMessagesReq, tc tpolicies.AuthServiceClient, ac upolicies.AuthServiceClient) (err error) {
	switch {
	case req.token != "":
		user, err := ac.Identify(ctx, &upolicies.IdentifyReq{Token: req.token})
		if err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		if _, err = tc.Authorize(ctx, &tpolicies.AuthorizeReq{Subject: user.GetId(), Object: req.chanID, Action: tpolicies.ReadAction, EntityType: tpolicies.GroupEntityType}); err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		return nil
	default:
		if _, err := tc.Authorize(ctx, &tpolicies.AuthorizeReq{Subject: req.key, Object: req.chanID, Action: tpolicies.ReadAction, EntityType: tpolicies.GroupEntityType}); err != nil {
			return errors.Wrap(errThingAccess, err)
		}
		return nil
	}
}
