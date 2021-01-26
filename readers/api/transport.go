// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	contentType = "application/json"
	defLimit    = 10
	defOffset   = 0
	format      = "format"
	defFormat   = "messages"
)

var (
	errInvalidRequest     = errors.New("received invalid request")
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errNotInQuery         = errors.New("parameter missing in the query")
	auth                  mainflux.ThingsServiceClient
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, tc mainflux.ThingsServiceClient, svcName string) http.Handler {
	auth = tc

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()
	mux.Get("/channels/:chanID/messages", kithttp.NewServer(
		listMessagesEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/version", mainflux.Version(svcName))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	chanID := bone.GetValue(r, "chanID")
	if chanID == "" {
		return nil, errInvalidRequest
	}

	if err := authorize(r, chanID); err != nil {
		return nil, err
	}

	offset, err := readUintQuery(r, "offset", defOffset)
	if err != nil {
		return nil, err
	}

	limit, err := readUintQuery(r, "limit", defLimit)
	if err != nil {
		return nil, err
	}

	format, err := readStringQuery(r, "format")
	if err != nil {
		return nil, err
	}
	if format != "" {
		format = defFormat
	}

	subtopic, err := readStringQuery(r, "subtopic")
	if err != nil {
		return nil, err
	}

	publisher, err := readStringQuery(r, "publisher")
	if err != nil {
		return nil, err
	}

	protocol, err := readStringQuery(r, "protocol")
	if err != nil {
		return nil, err
	}

	name, err := readStringQuery(r, "name")
	if err != nil {
		return nil, err
	}

	v, err := readFloatQuery(r, "v")
	if err != nil {
		return nil, err
	}

	vs, err := readStringQuery(r, "vs")
	if err != nil {
		return nil, err
	}

	vd, err := readStringQuery(r, "vd")
	if err != nil {
		return nil, err
	}

	from, err := readFloatQuery(r, "from")
	if err != nil {
		return nil, err
	}

	to, err := readFloatQuery(r, "to")
	if err != nil {
		return nil, err
	}

	req := listMessagesReq{
		chanID: chanID,
		pageMeta: readers.PageMetadata{
			Offset:      offset,
			Limit:       limit,
			Format:      format,
			Subtopic:    subtopic,
			Publisher:   publisher,
			Protocol:    protocol,
			Name:        name,
			Value:       v,
			StringValue: vs,
			DataValue:   vd,
			From:        from,
			To:          to,
		},
	}

	vb, err := readBoolQuery(r, "vb")
	// Check if vb is in the query
	if err != nil && err != errNotInQuery {
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
	case errors.Contains(err, errInvalidRequest):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errUnauthorizedAccess):
		w.WriteHeader(http.StatusForbidden)
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

func authorize(r *http.Request, chanID string) error {
	token := r.Header.Get("Authorization")
	if token == "" {
		return errUnauthorizedAccess
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := auth.CanAccessByKey(ctx, &mainflux.AccessByKeyReq{Token: token, ChanID: chanID})
	if err != nil {
		e, ok := status.FromError(err)
		if ok && e.Code() == codes.PermissionDenied {
			return errUnauthorizedAccess
		}
		return err
	}

	return nil
}

func readUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidRequest
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errInvalidRequest
	}

	return val, nil
}

func readFloatQuery(r *http.Request, key string) (float64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidRequest
	}

	if len(vals) == 0 {
		return 0, nil
	}

	fval := vals[0]
	val, err := strconv.ParseFloat(fval, 64)
	if err != nil {
		return 0, errInvalidRequest
	}

	return val, nil
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidRequest
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
}

func readBoolQuery(r *http.Request, key string) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return false, errInvalidRequest
	}

	if len(vals) == 0 {
		return false, errNotInQuery
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, errInvalidRequest
	}

	return b, nil
}
