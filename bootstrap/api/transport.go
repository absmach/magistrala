// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/absmach/magistrala"
	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/pelletier/go-toml/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"gopkg.in/yaml.v3"
)

const (
	contentType     = "application/json"
	yamlContentType = "yaml"
	tomlContentType = "toml"
	byteContentType = "application/octet-stream"
	offsetKey       = "offset"
	limitKey        = "limit"
	defOffset       = 0
	defLimit        = 10
)

var (
	fullMatch    = []string{"status", "external_id", "id"}
	partialMatch = []string{"name"}
	// ErrBootstrap indicates error in getting bootstrap configuration.
	ErrBootstrap = errors.New("failed to read bootstrap configuration")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc bootstrap.Service, authn smqauthn.AuthNMiddleware, reader bootstrap.ConfigReader, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r := chi.NewRouter()

	r.Route("/{domainID}/clients", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(authn.WithOptions(smqauthn.WithDomainCheck(true)).Middleware())
			r.Route("/configs", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addEndpoint(svc),
					decodeAddRequest,
					api.EncodeResponse,
					opts...), "add").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listEndpoint(svc),
					decodeListRequest,
					api.EncodeResponse,
					opts...), "list").ServeHTTP)

				r.Get("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					viewEndpoint(svc),
					decodeEntityRequest,
					api.EncodeResponse,
					opts...), "view").ServeHTTP)

				r.Patch("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					updateEndpoint(svc),
					decodeUpdateRequest,
					api.EncodeResponse,
					opts...), "update").ServeHTTP)

				r.Delete("/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					removeEndpoint(svc),
					decodeEntityRequest,
					api.EncodeResponse,
					opts...), "remove").ServeHTTP)

				r.Patch("/certs/{configID}", otelhttp.NewHandler(kithttp.NewServer(
					updateCertEndpoint(svc),
					decodeUpdateCertRequest,
					api.EncodeResponse,
					opts...), "update_cert").ServeHTTP)

				r.Post("/{configID}/enable", otelhttp.NewHandler(kithttp.NewServer(
					enableConfigEndpoint(svc),
					decodeChangeConfigStatusRequest,
					api.EncodeResponse,
					opts...), "enable_config").ServeHTTP)

				r.Post("/{configID}/disable", otelhttp.NewHandler(kithttp.NewServer(
					disableConfigEndpoint(svc),
					decodeChangeConfigStatusRequest,
					api.EncodeResponse,
					opts...), "disable_config").ServeHTTP)

			})
		})

		// Profile and enrollment binding endpoints.
		r.Route("/bootstrap", func(r chi.Router) {
			r.Use(authn.WithOptions(smqauthn.WithDomainCheck(true)).Middleware())

			r.Route("/profiles", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					createProfileEndpoint(svc),
					decodeCreateProfileRequest,
					api.EncodeResponse,
					opts...), "create_profile").ServeHTTP)

				r.Post("/upload", otelhttp.NewHandler(kithttp.NewServer(
					uploadProfileEndpoint(svc),
					decodeUploadProfileRequest,
					api.EncodeResponse,
					opts...), "upload_profile").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listProfilesEndpoint(svc),
					decodeListProfilesRequest,
					api.EncodeResponse,
					opts...), "list_profiles").ServeHTTP)

				r.Get("/{profileID}", otelhttp.NewHandler(kithttp.NewServer(
					viewProfileEndpoint(svc),
					decodeProfileEntityRequest,
					api.EncodeResponse,
					opts...), "view_profile").ServeHTTP)

				r.Get("/{profileID}/slots", otelhttp.NewHandler(kithttp.NewServer(
					profileSlotsEndpoint(svc),
					decodeProfileEntityRequest,
					api.EncodeResponse,
					opts...), "profile_slots").ServeHTTP)

				r.Post("/{profileID}/render-preview", otelhttp.NewHandler(kithttp.NewServer(
					renderPreviewEndpoint(svc),
					decodeRenderPreviewRequest,
					api.EncodeResponse,
					opts...), "render_preview").ServeHTTP)

				r.Patch("/{profileID}", otelhttp.NewHandler(kithttp.NewServer(
					updateProfileEndpoint(svc),
					decodeUpdateProfileRequest,
					api.EncodeResponse,
					opts...), "update_profile").ServeHTTP)

				r.Delete("/{profileID}", otelhttp.NewHandler(kithttp.NewServer(
					deleteProfileEndpoint(svc),
					decodeDeleteProfileRequest,
					api.EncodeResponse,
					opts...), "delete_profile").ServeHTTP)
			})

			r.Route("/enrollments", func(r chi.Router) {
				r.Patch("/{configID}/profile", otelhttp.NewHandler(kithttp.NewServer(
					assignProfileEndpoint(svc),
					decodeAssignProfileRequest,
					api.EncodeResponse,
					opts...), "assign_profile").ServeHTTP)

				r.Put("/{configID}/bindings", otelhttp.NewHandler(kithttp.NewServer(
					bindResourcesEndpoint(svc),
					decodeBindResourcesRequest,
					api.EncodeResponse,
					opts...), "bind_resources").ServeHTTP)

				r.Get("/{configID}/bindings", otelhttp.NewHandler(kithttp.NewServer(
					listBindingsEndpoint(svc),
					decodeEnrollmentEntityRequest,
					api.EncodeResponse,
					opts...), "list_bindings").ServeHTTP)

				r.Post("/{configID}/bindings/refresh", otelhttp.NewHandler(kithttp.NewServer(
					refreshBindingsEndpoint(svc),
					decodeRefreshBindingsRequest,
					api.EncodeResponse,
					opts...), "refresh_bindings").ServeHTTP)
			})
		})
	})

	r.Route("/clients/bootstrap", func(r chi.Router) {
		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, false),
			decodeBootstrapRequest,
			api.EncodeResponse,
			opts...), "bootstrap").ServeHTTP)
		r.Get("/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, false),
			decodeBootstrapRequest,
			api.EncodeResponse,
			opts...), "bootstrap").ServeHTTP)
		r.Get("/secure/{externalID}", otelhttp.NewHandler(kithttp.NewServer(
			bootstrapEndpoint(svc, reader, true),
			decodeBootstrapRequest,
			encodeSecureRes,
			opts...), "bootstrap_secure").ServeHTTP)
	})

	r.Get("/health", magistrala.Health("bootstrap", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeAddRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := addReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateReq{
		id: chi.URLParam(r, "configID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateCertRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	req := updateCertReq{
		clientID: chi.URLParam(r, "configID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeListRequest(_ context.Context, r *http.Request) (any, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidQueryParams)
	}

	req := listReq{
		filter: parseFilter(q),
		offset: o,
		limit:  l,
	}
	if status, ok := req.filter.FullMatch["status"]; ok {
		parsed, err := bootstrap.ToStatus(status)
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidQueryParams)
		}
		req.filter.FullMatch["status"] = parsed.String()
	}

	return req, nil
}

func decodeBootstrapRequest(_ context.Context, r *http.Request) (any, error) {
	req := bootstrapReq{
		id:  chi.URLParam(r, "externalID"),
		key: apiutil.ExtractClientSecret(r),
	}

	return req, nil
}

func decodeChangeConfigStatusRequest(_ context.Context, r *http.Request) (any, error) {
	return changeConfigStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    chi.URLParam(r, "configID"),
	}, nil
}

func decodeEntityRequest(_ context.Context, r *http.Request) (any, error) {
	req := entityReq{
		id: chi.URLParam(r, "configID"),
	}

	return req, nil
}

func encodeSecureRes(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", byteContentType)
	w.WriteHeader(http.StatusOK)
	if b, ok := response.([]byte); ok {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func parseFilter(values url.Values) bootstrap.Filter {
	ret := bootstrap.Filter{
		FullMatch:    make(map[string]string),
		PartialMatch: make(map[string]string),
	}
	for k := range values {
		if contains(fullMatch, k) {
			ret.FullMatch[k] = values.Get(k)
		}
		if contains(partialMatch, k) {
			ret.PartialMatch[k] = strings.ToLower(values.Get(k))
		}
	}

	return ret
}

func contains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

func decodeCreateProfileRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	var req createProfileReq
	if err := json.NewDecoder(r.Body).Decode(&req.Profile); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeUploadProfileRequest(_ context.Context, r *http.Request) (any, error) {
	contentType := r.Header.Get("Content-Type")
	var req uploadProfileReq

	switch {
	case strings.Contains(contentType, "json"):
		if err := json.NewDecoder(r.Body).Decode(&req.Profile); err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
		}
	case strings.Contains(contentType, yamlContentType):
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
		}
		if err := decodeYAMLProfile(body, &req.Profile); err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
		}
	case strings.Contains(contentType, tomlContentType):
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
		}
		if err := decodeTOMLProfile(body, &req.Profile); err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
		}
	default:
		return nil, apiutil.ErrUnsupportedContentType
	}

	return req, nil
}

func decodeYAMLProfile(body []byte, profile *bootstrap.Profile) error {
	var raw map[string]any
	if err := yaml.Unmarshal(body, &raw); err != nil {
		return err
	}
	return decodeProfileMap(raw, profile)
}

func decodeTOMLProfile(body []byte, profile *bootstrap.Profile) error {
	var raw map[string]any
	if err := toml.Unmarshal(body, &raw); err != nil {
		return err
	}
	return decodeProfileMap(raw, profile)
}

func decodeProfileMap(raw map[string]any, profile *bootstrap.Profile) error {
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, profile)
}

func decodeListProfilesRequest(_ context.Context, r *http.Request) (any, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	return listProfilesReq{offset: o, limit: l}, nil
}

func decodeProfileEntityRequest(_ context.Context, r *http.Request) (any, error) {
	return viewProfileReq{profileID: chi.URLParam(r, "profileID")}, nil
}

func decodeDeleteProfileRequest(_ context.Context, r *http.Request) (any, error) {
	return deleteProfileReq{profileID: chi.URLParam(r, "profileID")}, nil
}

func decodeUpdateProfileRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	req := updateProfileReq{profileID: chi.URLParam(r, "profileID")}
	if err := json.NewDecoder(r.Body).Decode(&req.Profile); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeRenderPreviewRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	req := renderPreviewReq{profileID: chi.URLParam(r, "profileID")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeAssignProfileRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	req := assignProfileReq{configID: chi.URLParam(r, "configID")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeBindResourcesRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}
	req := bindResourcesReq{
		token:    apiutil.ExtractBearerToken(r),
		configID: chi.URLParam(r, "configID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeEnrollmentEntityRequest(_ context.Context, r *http.Request) (any, error) {
	return listBindingsReq{configID: chi.URLParam(r, "configID")}, nil
}

func decodeRefreshBindingsRequest(_ context.Context, r *http.Request) (any, error) {
	return refreshBindingsReq{
		token:    apiutil.ExtractBearerToken(r),
		configID: chi.URLParam(r, "configID"),
	}, nil
}
