// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	ctSenmlJSON = "application/senml+json"
	ctSenmlCBOR = "application/senml+cbor"
	contentType = "application/json"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()
	r.Post("/channels/{chanID}/messages", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(),
		decodeRequest,
		api.EncodeResponse,
		opts...,
	), "publish").ServeHTTP)

	r.Post("/channels/{chanID}/messages/*", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(),
		decodeRequest,
		api.EncodeResponse,
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

	var req publishReq
	_, pass, ok := r.BasicAuth()
	switch {
	case ok:
		req.token = pass
	case !ok:
		req.token = apiutil.ExtractClientSecret(r)
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity)
	}
	defer r.Body.Close()

	req.msg = &messaging.Message{Payload: payload}

	return req, nil
}
