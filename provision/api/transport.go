package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/provision"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"
)

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errUnauthorized           = errors.New("missing or invalid credentials provided")
	errMalformedEntity        = errors.New("malformed entity")
	errConflict               = errors.New("entity already exists")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc provision.Service) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/mapping", kithttp.NewServer(
		doProvision(svc),
		decodeThingCreation,
		encodeResponse,
		opts...,
	))

	r.Handle("/metrics", promhttp.Handler())

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

func decodeThingCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	req := addThingReq{token: r.Header.Get("Authorization")}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case io.EOF, errMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case errConflict:
		w.WriteHeader(http.StatusConflict)
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
