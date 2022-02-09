// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/httputil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType      = "application/json"
	offsetKey        = "offset"
	limitKey         = "limit"
	formatKey        = "format"
	subtopicKey      = "subtopic"
	publisherKey     = "publisher"
	protocolKey      = "protocol"
	nameKey          = "name"
	valueKey         = "v"
	stringValueKey   = "vs"
	dataValueKey     = "vd"
	comparatorKey    = "comparator"
	fromKey          = "from"
	toKey            = "to"
	defLimit         = 10
	defOffset        = 0
	defFormat        = "messages"
	thingTokenPrefix = "Thing "
	userTokenPrefix  = "Bearer "
)

var (
	errThingAccess = errors.New("thing has no permission")
	errUserAccess  = errors.New("user has no permission")
	thingsAuth     mainflux.ThingsServiceClient
	usersAuth      mainflux.AuthServiceClient
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient, svcName string) http.Handler {
	thingsAuth = tc
	usersAuth = ac

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

	mux.GetFunc("/health", mainflux.Health(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(ctx context.Context, r *http.Request) (interface{}, error) {
	offset, err := httputil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	limit, err := httputil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	format, err := httputil.ReadStringQuery(r, formatKey, defFormat)
	if err != nil {
		return nil, err
	}

	subtopic, err := httputil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, err
	}

	publisher, err := httputil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, err
	}

	protocol, err := httputil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, err
	}

	name, err := httputil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	v, err := httputil.ReadFloatQuery(r, valueKey, 0)
	if err != nil {
		return nil, err
	}

	comparator, err := httputil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return nil, err
	}

	vs, err := httputil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return nil, err
	}

	vd, err := httputil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return nil, err
	}

	from, err := httputil.ReadFloatQuery(r, fromKey, 0)
	if err != nil {
		return nil, err
	}

	to, err := httputil.ReadFloatQuery(r, toKey, 0)
	if err != nil {
		return nil, err
	}

	req := listMessagesReq{
		chanID: bone.GetValue(r, "chanID"),
		token:  r.Header.Get("Authorization"),
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
			From:        from,
			To:          to,
		},
	}

	vb, err := readBoolValueQuery(r, "vb")
	if err != nil && err != errors.ErrNotFoundParam {
		return nil, err
	}
	if err == nil {
		req.pageMeta.BoolValue = vb
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
	switch {
	case errors.Contains(err, nil):
	case errors.Contains(err, errors.ErrInvalidQueryParams),
		errors.Contains(err, errors.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication):
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	errorVal, ok := err.(errors.Error)
	if ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func authorize(ctx context.Context, req listMessagesReq, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient) (err error) {
	switch {
	case strings.HasPrefix(req.token, userTokenPrefix):
		token := strings.TrimPrefix(req.token, userTokenPrefix)
		user, err := usersAuth.Identify(ctx, &mainflux.Token{Value: token})
		if err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		if _, err = thingsAuth.IsChannelOwner(ctx, &mainflux.ChannelOwnerReq{Owner: user.Email, ChanID: req.chanID}); err != nil {
			e, ok := status.FromError(err)
			if ok && e.Code() == codes.PermissionDenied {
				return errors.Wrap(errUserAccess, err)
			}
			return err
		}
		return nil
	default:
		token := strings.TrimPrefix(req.token, thingTokenPrefix)
		if _, err := thingsAuth.CanAccessByKey(ctx, &mainflux.AccessByKeyReq{Token: token, ChanID: req.chanID}); err != nil {
			return errors.Wrap(errThingAccess, err)
		}
		return nil
	}
}

func readBoolValueQuery(r *http.Request, key string) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return false, errors.ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return false, errors.ErrNotFoundParam
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, errors.ErrInvalidQueryParams
	}

	return b, nil
}
