package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/clients"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const contentType = "application/json"

var (
	errUnsupportedContentType = errors.New("unsupported content type")
	errInvalidQueryParams     = errors.New("invalid query params")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc clients.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/clients", kithttp.NewServer(
		addClientEndpoint(svc),
		decodeClientCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/clients/:id", kithttp.NewServer(
		updateClientEndpoint(svc),
		decodeClientUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/clients/:id", kithttp.NewServer(
		removeClientEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/clients/:id", kithttp.NewServer(
		viewClientEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/clients", kithttp.NewServer(
		listClientsEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Post("/channels", kithttp.NewServer(
		createChannelEndpoint(svc),
		decodeChannelCreation,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:id", kithttp.NewServer(
		updateChannelEndpoint(svc),
		decodeChannelUpdate,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:id", kithttp.NewServer(
		removeChannelEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels/:id", kithttp.NewServer(
		viewChannelEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Get("/channels", kithttp.NewServer(
		listChannelsEndpoint(svc),
		decodeList,
		encodeResponse,
		opts...,
	))

	r.Put("/channels/:chanId/clients/:clientId", kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.Delete("/channels/:chanId/clients/:clientId", kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeConnection,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("clients"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeClientCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var client clients.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		return nil, err
	}

	req := addClientReq{
		key:    r.Header.Get("Authorization"),
		client: client,
	}

	return req, nil
}

func decodeClientUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var client clients.Client
	if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
		return nil, err
	}

	req := updateClientReq{
		key:    r.Header.Get("Authorization"),
		id:     bone.GetValue(r, "id"),
		client: client,
	}

	return req, nil
}

func decodeChannelCreation(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var channel clients.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		return nil, err
	}

	req := createChannelReq{
		key:     r.Header.Get("Authorization"),
		channel: channel,
	}

	return req, nil
}

func decodeChannelUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if r.Header.Get("Content-Type") != contentType {
		return nil, errUnsupportedContentType
	}

	var channel clients.Channel
	if err := json.NewDecoder(r.Body).Decode(&channel); err != nil {
		return nil, err
	}

	req := updateChannelReq{
		key:     r.Header.Get("Authorization"),
		id:      bone.GetValue(r, "id"),
		channel: channel,
	}

	return req, nil
}

func decodeView(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewResourceReq{
		key: r.Header.Get("Authorization"),
		id:  bone.GetValue(r, "id"),
	}

	return req, nil
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, errInvalidQueryParams
	}
	offset := 0
	limit := 10

	off, lmt := q["offset"], q["limit"]

	if len(off) > 1 || len(lmt) > 1 {
		return nil, errInvalidQueryParams
	}

	if len(off) == 1 {
		offset, err = strconv.Atoi(off[0])
		if err != nil {
			return nil, errInvalidQueryParams
		}
	}

	if len(lmt) == 1 {
		limit, err = strconv.Atoi(lmt[0])
		if err != nil {
			return nil, errInvalidQueryParams
		}
	}
	req := listResourcesReq{
		key:    r.Header.Get("Authorization"),
		offset: offset,
		limit:  limit,
	}

	return req, nil
}

func decodeConnection(_ context.Context, r *http.Request) (interface{}, error) {
	req := connectionReq{
		key:      r.Header.Get("Authorization"),
		chanID:   bone.GetValue(r, "chanId"),
		clientID: bone.GetValue(r, "clientId"),
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
	w.Header().Set("Content-Type", contentType)

	switch err {
	case clients.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case clients.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case clients.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case clients.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	case errUnsupportedContentType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errInvalidQueryParams:
		w.WriteHeader(http.StatusBadRequest)
	case io.ErrUnexpectedEOF:
		w.WriteHeader(http.StatusBadRequest)
	case io.EOF:
		w.WriteHeader(http.StatusBadRequest)
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
