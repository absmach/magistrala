package api

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol string = "http"
	ctJson   string = "application/senml+json"
)

var (
	errMalformedData      error = errors.New("malformed SenML data")
	errUnknownType        error = errors.New("unknown content type")
	errUnauthorizedAccess error = errors.New("missing or invalid credentials provided")
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc adapter.Service) http.Handler {
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

	r.GetFunc("/version", mainflux.Version())
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func decodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	ct, err := checkContentType(r)
	if err != nil {
		return nil, err
	}

	publisher, err := authorize(r)
	if err != nil {
		return nil, err
	}

	payload, err := decodePayload(r.Body)
	if err != nil {
		return nil, err
	}

	channel := bone.GetValue(r, "id")

	msg := writer.RawMessage{
		Publisher:   publisher,
		Protocol:    protocol,
		ContentType: ct,
		Channel:     channel,
		Payload:     payload,
	}

	return msg, nil
}

// TODO: contact an auth provider
func authorize(r *http.Request) (string, error) {
	var apiKey string
	if apiKey = r.Header.Get("Authorization"); apiKey == "" {
		return "", errUnauthorizedAccess
	}

	return apiKey, nil
}

func checkContentType(r *http.Request) (string, error) {
	ct := r.Header.Get("Content-Type")

	if ct != ctJson {
		return "", errUnknownType
	}

	return ct, nil
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
	case errUnknownType:
		w.WriteHeader(http.StatusUnsupportedMediaType)
	case errUnauthorizedAccess:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
