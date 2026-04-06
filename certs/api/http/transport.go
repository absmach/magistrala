// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/certs"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/crypto/ocsp"
)

const (
	offsetKey       = "offset"
	limitKey        = "limit"
	entityKey       = "entity_id"
	ocspStatusParam = "force_status"
	entityIDParam   = "entityID"
	ttl             = "ttl"
	defOffset       = 0
	defLimit        = 10
)

func authMiddleware(expectedSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := apiutil.ExtractBearerToken(r)
			if token == "" {
				EncodeError(r.Context(), apiutil.ErrBearerToken, w)
				return
			}

			if token != expectedSecret {
				EncodeError(r.Context(), errors.Wrap(certs.ErrMalformedEntity, errors.New("invalid authentication token")), w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc certs.Service, authn smqauthn.AuthNMiddleware, logger *slog.Logger, instanceID string, secret string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(loggingErrorEncoder(logger, EncodeError)),
	}

	mux := chi.NewRouter()

	mux.Route("/{domainID}", func(r chi.Router) {
		r.Route("/certs", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(authn.Middleware())
				r.Post("/issue/{entityID}", otelhttp.NewHandler(kithttp.NewServer(
					issueCertEndpoint(svc),
					decodeIssueCert,
					api.EncodeResponse,
					opts...,
				), "issue_cert").ServeHTTP)
				r.Patch("/{id}/renew", otelhttp.NewHandler(kithttp.NewServer(
					renewCertEndpoint(svc),
					decodeView,
					api.EncodeResponse,
					opts...,
				), "renew_cert").ServeHTTP)
				r.Patch("/{id}/revoke", otelhttp.NewHandler(kithttp.NewServer(
					revokeCertEndpoint(svc),
					decodeView,
					api.EncodeResponse,
					opts...,
				), "revoke_cert").ServeHTTP)
				r.Delete("/{entityID}/delete", otelhttp.NewHandler(kithttp.NewServer(
					deleteCertEndpoint(svc),
					decodeDelete,
					api.EncodeResponse,
					opts...,
				), "delete_cert").ServeHTTP)
				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listCertsEndpoint(svc),
					decodeListCerts,
					api.EncodeResponse,
					opts...,
				), "list_certs").ServeHTTP)
				r.Get("/{id}", otelhttp.NewHandler(kithttp.NewServer(
					viewCertEndpoint(svc),
					decodeView,
					api.EncodeResponse,
					opts...,
				), "view_cert").ServeHTTP)
				r.Route("/csrs", func(r chi.Router) {
					r.Post("/{entityID}", otelhttp.NewHandler(kithttp.NewServer(
						issueFromCSREndpoint(svc),
						decodeIssueFromCSR,
						api.EncodeResponse,
						opts...,
					), "issue_from_csr").ServeHTTP)
				})
			})
		})
	})

	mux.Route("/certs", func(r chi.Router) {
		r.Post("/ocsp", otelhttp.NewHandler(kithttp.NewServer(
			ocspEndpoint(svc),
			decodeOCSPRequest,
			encodeOSCPResponse,
			opts...,
		), "ocsp").ServeHTTP)
		r.Get("/crl", otelhttp.NewHandler(kithttp.NewServer(
			generateCRLEndpoint(svc),
			decodeCRL,
			api.EncodeResponse,
			opts...,
		), "generate_crl").ServeHTTP)
		r.Get("/view-ca", otelhttp.NewHandler(kithttp.NewServer(
			viewCAEndpoint(svc),
			decodeViewCA,
			api.EncodeResponse,
			opts...,
		), "view_ca").ServeHTTP)
		r.Get("/download-ca", otelhttp.NewHandler(kithttp.NewServer(
			downloadCAEndpoint(svc),
			decodeDownloadCA,
			encodeCADownloadResponse,
			opts...,
		), "download_ca").ServeHTTP)
	})

	mux.Group(func(r chi.Router) {
		r.Use(authMiddleware(secret))
		r.Post("/certs/csrs/{entityID}", otelhttp.NewHandler(kithttp.NewServer(
			issueFromCSRInternalEndpoint(svc),
			decodeIssueFromCSRInternal,
			api.EncodeResponse,
			opts...,
		), "issue_from_csr_internal").ServeHTTP)
	})

	mux.Get("/health", certs.Health("certs", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeView(_ context.Context, r *http.Request) (any, error) {
	req := viewReq{
		id: chi.URLParam(r, "id"),
	}
	return req, nil
}

func decodeDelete(_ context.Context, r *http.Request) (any, error) {
	req := deleteReq{
		entityID: chi.URLParam(r, "entityID"),
	}
	return req, nil
}

func decodeCRL(_ context.Context, r *http.Request) (any, error) {
	req := crlReq{}
	return req, nil
}

func decodeDownloadCA(_ context.Context, r *http.Request) (any, error) {
	req := downloadReq{}
	return req, nil
}

func decodeViewCA(_ context.Context, r *http.Request) (any, error) {
	req := downloadReq{}
	return req, nil
}

func decodeOCSPRequest(_ context.Context, r *http.Request) (any, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(certs.ErrMalformedEntity, err)
	}
	defer r.Body.Close()

	req, err := ocsp.ParseRequest(body)
	if err != nil {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			return decodeJsonOCSPRequest(body)
		}
		return nil, fmt.Errorf("invalid OCSP request: %w", err)
	}

	request := ocspReq{
		req:         req,
		StatusParam: strings.TrimSpace(r.URL.Query().Get(ocspStatusParam)),
	}

	return request, nil
}

func decodeJsonOCSPRequest(body []byte) (any, error) {
	var simple ocspReq
	if err := json.Unmarshal(body, &simple); err != nil {
		return nil, fmt.Errorf("invalid JSON OCSP request: %w", err)
	}

	request := ocspReq{
		SerialNumber: simple.SerialNumber,
		Certificate:  simple.Certificate,
	}

	return request, nil
}

func decodeIssueCert(_ context.Context, r *http.Request) (any, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	req := issueCertReq{
		entityID: chi.URLParam(r, entityIDParam),
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, errors.Wrap(ErrInvalidRequest, err)
	}

	return req, nil
}

func decodeListCerts(_ context.Context, r *http.Request) (any, error) {
	o, err := readNumQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readNumQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	entity, err := readStringQuery(r, entityKey, "")
	if err != nil {
		return nil, err
	}

	req := listCertsReq{
		pm: certs.PageMetadata{
			Offset:   o,
			Limit:    l,
			EntityID: entity,
		},
	}
	return req, nil
}

func decodeIssueFromCSR(_ context.Context, r *http.Request) (any, error) {
	t, err := readStringQuery(r, ttl, "")
	if err != nil {
		return nil, err
	}

	req := IssueFromCSRReq{
		entityID: chi.URLParam(r, "entityID"),
		ttl:      t,
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidRequest, errors.New("failed to read request body"))
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		return nil, errors.Wrap(ErrInvalidRequest, errors.New("failed to decode JSON"))
	}

	return req, nil
}

func decodeIssueFromCSRInternal(_ context.Context, r *http.Request) (any, error) {
	t, err := readStringQuery(r, ttl, "")
	if err != nil {
		return nil, err
	}

	req := IssueFromCSRInternalReq{
		entityID: chi.URLParam(r, "entityID"),
		ttl:      t,
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(ErrInvalidRequest, errors.New("failed to read request body"))
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		return nil, errors.Wrap(ErrInvalidRequest, errors.New("failed to decode JSON"))
	}

	return req, nil
}

func encodeOSCPResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	res := response.(ocspRawRes)

	w.Header().Set("Content-Type", OCSPType)
	_, err := w.Write(res.Data)
	return err
}

func encodeCADownloadResponse(_ context.Context, w http.ResponseWriter, response any) error {
	resp := response.(fileDownloadRes)
	var buffer bytes.Buffer
	zw := zip.NewWriter(&buffer)

	f, err := zw.Create("ca.crt")
	if err != nil {
		return err
	}

	if _, err = f.Write(resp.Certificate); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", resp.Filename))
	w.Header().Set("Content-Type", resp.ContentType)

	_, err = w.Write(buffer.Bytes())

	return err
}

// loggingErrorEncoder is a go-kit error encoder logging decorator.
func loggingErrorEncoder(logger *slog.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		if errors.Contains(err, ErrValidation) {
			logger.Error(err.Error())
		}
		enc(ctx, err, w)
	}
}

// readStringQuery reads the value of string http query parameters for a given key.
func readStringQuery(r *http.Request, key, def string) (string, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return "", ErrInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	return vals[0], nil
}

// readNumQuery returns a numeric value.
func readNumQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := r.URL.Query()[key]
	if len(vals) > 1 {
		return 0, ErrInvalidQueryParams
	}
	if len(vals) == 0 {
		return def, nil
	}
	val := vals[0]

	v, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, errors.Wrap(ErrInvalidQueryParams, err)
	}
	return v, nil
}
