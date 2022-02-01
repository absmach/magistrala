package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/provision"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc provision.Service) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/mapping", kithttp.NewServer(
		doProvision(svc),
		decodeProvisionRequest,
		encodeResponse,
		opts...,
	))

	r.Get("/mapping", kithttp.NewServer(
		getMapping(svc),
		decodeMappingRequest,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())
	r.GetFunc("/health", mainflux.Health("provision"))

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

func decodeProvisionRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.ErrUnsupportedContentType
	}

	req := provisionReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func decodeMappingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errors.ErrUnsupportedContentType
	}

	req := mappingReq{token: r.Header.Get("Authorization")}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case errors.ErrUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case io.EOF, errors.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case errors.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	case errors.ErrAuthentication:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		switch err.(type) {
		case *json.SyntaxError:
			w.WriteHeader(http.StatusBadRequest)
		case *json.UnmarshalTypeError:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
