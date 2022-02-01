package policies

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go"
)

const contentType = "application/json"

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux.Post("/policies", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_policy_bulk")(createPolicyEndpoint(svc)),
		decodePoliciesRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/policies", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_policies")(deletePoliciesEndpoint(svc)),
		decodePoliciesRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodePoliciesRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req policiesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.token = r.Header.Get("Authorization")
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
	case errors.Contains(err, errors.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, auth.ErrMemberAlreadyAssigned):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, io.EOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, io.ErrUnexpectedEOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	errorVal, ok := err.(errors.Error)
	if ok {
		if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
