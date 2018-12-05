//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const protocol = "http"

var (
	errMalformedData = errors.New("malformed request data")
	auth             mainflux.ThingsServiceClient
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc mainflux.MessagePublisher, tc mainflux.ThingsServiceClient) http.Handler {
	auth = tc

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()

	r.Post("/channels/:id/messages", kithttp.NewServer(
		sendMessageEndpoint(svc),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/version", mainflux.Version("http"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	publisher, err := authorize(r)
	if err != nil {
		return nil, err
	}

	payload, err := decodePayload(r.Body)
	if err != nil {
		return nil, err
	}

	channel := bone.GetValue(r, "id")
	if channel == "" {
		return nil, errMalformedData
	}

	msg := mainflux.RawMessage{
		Publisher:   publisher,
		Protocol:    protocol,
		ContentType: r.Header.Get("Content-Type"),
		Channel:     channel,
		Payload:     payload,
	}

	return msg, nil
}

func authorize(r *http.Request) (string, error) {
	apiKey := r.Header.Get("Authorization")

	if apiKey == "" {
		return "", things.ErrUnauthorizedAccess
	}

	// extract ID from /channels/:id/messages
	chanID := bone.GetValue(r, "id")
	if chanID == "" {
		return "", errMalformedData
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := auth.CanAccess(ctx, &mainflux.AccessReq{Token: apiKey, ChanID: chanID})
	if err != nil {
		return "", err
	}

	return id.GetValue(), nil
}

func decodePayload(body io.ReadCloser) ([]byte, error) {
	payload, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, errMalformedData
	}
	defer body.Close()

	return payload, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch err {
	case errMalformedData:
		w.WriteHeader(http.StatusBadRequest)
	case things.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	default:
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.PermissionDenied:
				w.WriteHeader(http.StatusForbidden)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}
}
