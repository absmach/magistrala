package api

import (
	"context"
	"encoding/json"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/manager"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc manager.Service) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/users", kithttp.NewServer(
		registrationEndpoint(svc),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	r.Post("/tokens", kithttp.NewServer(
		loginEndpoint(svc),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	r.Post("/identity", kithttp.NewServer(
		identityEndpoint(svc),
		decodeIdentity,
		encodeResponse,
		opts...,
	))

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

	r.Get("/channels/:id/messages", kithttp.NewServer(
		canAccessEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/:id/messages", kithttp.NewServer(
		canAccessEndpoint(svc),
		decodeView,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version())
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeIdentity(_ context.Context, r *http.Request) (interface{}, error) {
	req := identityReq{
		key: r.Header.Get("Authorization"),
	}

	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	var user manager.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err
	}

	return userReq{user}, nil
}

func decodeClientCreation(_ context.Context, r *http.Request) (interface{}, error) {
	var client manager.Client
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
	var client manager.Client
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
	var channel manager.Channel
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
	var channel manager.Channel
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
	req := listResourcesReq{
		key:    r.Header.Get("Authorization"),
		size:   10,
		offset: 0,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(apiRes); ok {
		for k, v := range ar.headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.code())

		if ar.empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", contentType)

	switch err {
	case manager.ErrMalformedEntity:
		w.WriteHeader(http.StatusBadRequest)
	case manager.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	case manager.ErrNotFound:
		w.WriteHeader(http.StatusNotFound)
	case manager.ErrConflict:
		w.WriteHeader(http.StatusConflict)
	default:
		if _, ok := err.(*json.SyntaxError); ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}
}
