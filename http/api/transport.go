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
	"net/url"
	"regexp"
	"strings"
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
	errMalformedData     = errors.New("malformed request data")
	errMalformedSubtopic = errors.New("malformed subtopic")
)

var (
	auth              mainflux.ThingsServiceClient
	channelPartRegExp = regexp.MustCompile(`^/channels/([\w\-]+)/messages((/[^/?]+)*)?(\?.*)?$`)
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc mainflux.MessagePublisher, tc mainflux.ThingsServiceClient) http.Handler {
	auth = tc

	r := bone.New()
	r.Post("/channels/:id/messages", handshake(svc))
	r.Post("/channels/:id/messages/*", handshake(svc))

	r.GetFunc("/version", mainflux.Version("http"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func handshake(svc mainflux.MessagePublisher) *kithttp.Server {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	return kithttp.NewServer(
		sendMessageEndpoint(svc),
		decodeRequest,
		encodeResponse,
		opts...,
	)
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	var err error
	subtopic, err = url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
	}
	subtopic = strings.Replace(subtopic, "/", ".", -1)
	// channelParts[2] contains the subtopic parts starting with char /
	subtopic = subtopic[1:]
	return subtopic, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	channelParts := channelPartRegExp.FindStringSubmatch(r.RequestURI)
	if len(channelParts) < 2 {
		return nil, errMalformedData
	}

	chanID := bone.GetValue(r, "id")
	subtopic, err := parseSubtopic(channelParts[2])
	if err != nil {
		return nil, err
	}

	publisher, err := authorize(r, chanID)
	if err != nil {
		return nil, err
	}

	payload, err := decodePayload(r.Body)
	if err != nil {
		return nil, err
	}

	msg := mainflux.RawMessage{
		Publisher:   publisher,
		Protocol:    protocol,
		ContentType: r.Header.Get("Content-Type"),
		Channel:     chanID,
		Subtopic:    subtopic,
		Payload:     payload,
	}

	return msg, nil
}

func authorize(r *http.Request, chanID string) (string, error) {
	apiKey := r.Header.Get("Authorization")

	if apiKey == "" {
		return "", things.ErrUnauthorizedAccess
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
