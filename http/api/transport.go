// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const protocol = "http"

var (
	errMalformedData     = errors.New("malformed request data")
	errMalformedSubtopic = errors.New("malformed subtopic")
)

var channelPartRegExp = regexp.MustCompile(`^/channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, tracer opentracing.Tracer) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()
	r.Post("/channels/:id/messages", kithttp.NewServer(
		kitot.TraceServer(tracer, "publish")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.Post("/channels/:id/messages/*", kithttp.NewServer(
		kitot.TraceServer(tracer, "publish")(sendMessageEndpoint(svc)),
		decodeRequest,
		encodeResponse,
		opts...,
	))

	r.GetFunc("/health", mainflux.Health("http"))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
	}
	subtopic = strings.Replace(subtopic, "/", ".", -1)

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", errMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}

func decodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	channelParts := channelPartRegExp.FindStringSubmatch(r.RequestURI)
	if len(channelParts) < 2 {
		return nil, errMalformedData
	}

	chanID := bone.GetValue(r, "id")
	subtopic, err := parseSubtopic(channelParts[2])
	if err != nil {
		return nil, err
	}
	token := r.Header.Get("Authorization")
	if _, pass, ok := r.BasicAuth(); ok {
		token = pass
	}

	payload, err := decodePayload(r.Body)
	if err != nil {
		return nil, err
	}

	msg := messaging.Message{
		Protocol: protocol,
		Channel:  chanID,
		Subtopic: subtopic,
		Payload:  payload,
		Created:  time.Now().UnixNano(),
	}

	req := publishReq{
		msg:   msg,
		token: token,
	}

	return req, nil
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
	case errMalformedData, errMalformedSubtopic:
		w.WriteHeader(http.StatusBadRequest)
	case errors.ErrAuthentication:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.Unauthenticated:
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}
}
