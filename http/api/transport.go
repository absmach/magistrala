// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	protocol    = "http"
	ctSenmlJSON = "application/senml+json"
	ctSenmlCBOR = "application/senml+cbor"
	contentType = "application/json"
)

var errMalformedSubtopic = errors.New("malformed subtopic")

var channelPartRegExp = regexp.MustCompile(`^/channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	r := bone.New()
	r.Post("/channels/:chanID/messages", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(svc),
		decodeRequest,
		encodeResponse,
		opts...,
	), "publish"))

	r.Post("/channels/:chanID/messages/*", otelhttp.NewHandler(kithttp.NewServer(
		sendMessageEndpoint(svc),
		decodeRequest,
		encodeResponse,
		opts...,
	), "publish"))

	r.GetFunc("/health", mainflux.Health("http", instanceID))
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errors.Wrap(apiutil.ErrValidation, errMalformedSubtopic)
	}
	subtopic = strings.ReplaceAll(subtopic, "/", ".")

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", errors.Wrap(apiutil.ErrValidation, errMalformedSubtopic)
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	ct := r.Header.Get("Content-Type")
	if ct != ctSenmlJSON && ct != contentType && ct != ctSenmlCBOR {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	channelParts := channelPartRegExp.FindStringSubmatch(r.RequestURI)
	if len(channelParts) < 2 {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity)
	}

	subtopic, err := parseSubtopic(channelParts[2])
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	var token string
	_, pass, ok := r.BasicAuth()
	switch {
	case ok:
		token = pass
	case !ok:
		token = apiutil.ExtractThingKey(r)
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity)
	}
	defer r.Body.Close()

	req := publishReq{
		msg: &messaging.Message{
			Protocol: protocol,
			Channel:  bone.GetValue(r, "chanID"),
			Subtopic: subtopic,
			Payload:  payload,
			Created:  time.Now().UnixNano(),
		},
		token: token,
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerKey),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, apiutil.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errors.Contains(err, errMalformedSubtopic),
		errors.Contains(err, errors.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)

	default:
		switch e, ok := status.FromError(err); {
		case ok:
			switch e.Code() {
			case codes.Unauthenticated:
				w.WriteHeader(http.StatusUnauthorized)
			case codes.PermissionDenied:
				w.WriteHeader(http.StatusForbidden)
			case codes.Internal:
				w.WriteHeader(http.StatusInternalServerError)
			case codes.NotFound:
				err = errors.ErrNotFound
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)

		errMsg := errorVal.Msg()
		if errorVal.Err() != nil {
			errMsg = fmt.Sprintf("%s : %s", errMsg, errorVal.Err().Msg())
		}

		if err := json.NewEncoder(w).Encode(apiutil.ErrorRes{Err: errMsg}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
