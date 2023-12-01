// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ctSenmlJSON = "application/senml+json"
	ctSenmlCBOR = "application/senml+cbor"
	contentType = "application/json"
)

var errMalformedSubtopic = errors.New("malformed subtopic")

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := chi.NewRouter()
	r.Post("/channels/{chanID}/messages", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(),
		decodeRequest,
		encodeResponse,
		opts...,
	), "publish").ServeHTTP)

	r.Post("/channels/{chanID}/messages/*", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(),
		decodeRequest,
		encodeResponse,
		opts...,
	), "publish").ServeHTTP)

	r.Get("/health", magistrala.Health("http", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	ct := r.Header.Get("Content-Type")
	if ct != ctSenmlJSON && ct != contentType && ct != ctSenmlCBOR {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	return nil, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerKey),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, svcerr.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errMalformedSubtopic),
		errors.Contains(err, svcerr.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)

	default:
		switch e, ok := status.FromError(err); {
		case ok:
			switch e.Code() {
			case codes.Unauthenticated:
				w.WriteHeader(http.StatusUnauthorized)
			case codes.PermissionDenied:
				w.WriteHeader(http.StatusForbidden)
			case codes.Internal:
				w.WriteHeader(http.StatusInternalServerError)
			case codes.NotFound:
				err = svcerr.ErrNotFound
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
