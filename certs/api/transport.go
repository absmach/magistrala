// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
	offsetKey   = "offset"
	limitKey    = "limit"
	certKey     = "cert_id"
	thingKey    = "thing_id"
	nameKey     = "name"
	serialKey   = "serial"
	statusKey   = "status"

	defStatus = "all"
	defOffset = 0
	defLimit  = 10
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, encodeError)),
	}

	r := bone.New()

	r.Post("/certs", kithttp.NewServer(
		issueCert(svc),
		decodeCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/certs/:certID", kithttp.NewServer(
		viewCert(svc),
		decodeViewRevokeRenewRemoveCerts,
		encodeResponse,
		opts...,
	))

	r.Post("/certs/:certID/revoke", kithttp.NewServer(
		revokeCert(svc),
		decodeViewRevokeRenewRemoveCerts,
		encodeResponse,
		opts...,
	))

	r.Post("/certs/:certID/renew", kithttp.NewServer(
		renewCert(svc),
		decodeViewRevokeRenewRemoveCerts,
		encodeResponse,
		opts...,
	))

	r.Delete("/certs/:certID", kithttp.NewServer(
		removeCert(svc),
		decodeViewRevokeRenewRemoveCerts,
		encodeResponse,
		opts...,
	))

	r.Get("/certs", kithttp.NewServer(
		listCerts(svc),
		decodeListCerts,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:thingID/revoke", kithttp.NewServer(
		revokeThingCerts(svc),
		decodeRevokeRenewRemoveThing,
		encodeResponse,
		opts...,
	))

	r.Post("/things/:thingID/renew", kithttp.NewServer(
		renewThingCerts(svc),
		decodeRevokeRenewRemoveThing,
		encodeResponse,
		opts...,
	))

	r.Delete("/things/:thingID", kithttp.NewServer(
		removeThingCerts(svc),
		decodeRevokeRenewRemoveThing,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("certs"))

	return r
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

func decodeListCerts(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}
	o, err := apiutil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	certID, err := apiutil.ReadStringQuery(r, certKey, "")
	if err != nil {
		return nil, err
	}

	thingID, err := apiutil.ReadStringQuery(r, thingKey, "")
	if err != nil {
		return nil, err
	}

	serial, err := apiutil.ReadStringQuery(r, serialKey, "")
	if err != nil {
		return nil, err
	}

	name, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, err
	}

	status, err := apiutil.ReadStringQuery(r, statusKey, defStatus)
	if err != nil {
		return nil, err
	}

	req := listReq{
		token:   apiutil.ExtractBearerToken(r),
		certID:  certID,
		thingID: thingID,
		serial:  serial,
		status:  status,
		name:    name,
		limit:   l,
		offset:  o,
	}
	return req, nil
}

func decodeCerts(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.ErrUnsupportedContentType
	}

	req := addCertsReq{token: apiutil.ExtractBearerToken(r)}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeViewRevokeRenewRemoveCerts(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewRevokeRenewRemoveReq{
		token:  apiutil.ExtractBearerToken(r),
		certID: bone.GetValue(r, "certID"),
	}

	return req, nil
}

func decodeRevokeRenewRemoveThing(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := apiutil.ReadIntQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	req := revokeRenewRemoveThingIDReq{
		token:   apiutil.ExtractBearerToken(r),
		thingID: bone.GetValue(r, "thingID"),
		limit:   l,
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch {

	case err == apiutil.ErrBearerToken:
		w.WriteHeader(http.StatusUnauthorized)

	case err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingCertData,
		err == apiutil.ErrLimitSize,
		err == apiutil.ErrOffsetSize,
		err == apiutil.ErrLimitSize:
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
		err = errors.ErrNotFound

	case errors.Contains(err, errors.ErrAuthentication):
		w.WriteHeader(http.StatusUnauthorized)
		err = errors.ErrAuthentication

	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
		err = errors.ErrUnsupportedContentType

	case errors.Contains(err, errors.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
		err = errors.ErrMalformedEntity

	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		err = errors.ErrConflict

	case errors.Contains(err, errors.ErrCreateEntity):
		w.WriteHeader(http.StatusInternalServerError)
		err = errors.ErrCreateEntity

	case errors.Contains(err, errors.ErrViewEntity):
		w.WriteHeader(http.StatusInternalServerError)
		err = errors.ErrViewEntity

	case errors.Contains(err, errors.ErrUpdateEntity):
		w.WriteHeader(http.StatusInternalServerError)
		err = errors.ErrUpdateEntity

	case errors.Contains(err, errors.ErrRemoveEntity):
		w.WriteHeader(http.StatusInternalServerError)
		err = errors.ErrRemoveEntity

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errorVal.Msg()}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
