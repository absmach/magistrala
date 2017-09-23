package api

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/cisco/senml"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol string = "HTTP"
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
	if _, err := checkContentType(r); err != nil {
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

	return buildMessages(bone.GetValue(r, "id"), publisher, payload), nil
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

func decodePayload(body io.ReadCloser) (senml.SenML, error) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return senml.SenML{}, errMalformedData
	}
	defer body.Close()

	var payload senml.SenML
	if payload, err = senml.Decode(b, senml.JSON); err != nil {
		return senml.SenML{}, errMalformedData
	}

	return payload, nil
}

func buildMessages(channel, publisher string, data senml.SenML) []writer.Message {
	messages := make([]writer.Message, len(data.Records))

	// NOTE:
	// Due to the deficiencies in cisco's senml library, base value and base
	// sum are set to 0. Once the library is updates, these value will be
	// properly set.
	for k, v := range data.Records {
		messages[k] = writer.Message{
			Channel:     channel,
			Publisher:   publisher,
			Protocol:    protocol,
			BaseName:    v.BaseName,
			BaseTime:    v.BaseTime,
			BaseUnit:    v.BaseUnit,
			BaseValue:   0,
			BaseSum:     0,
			Version:     v.BaseVersion,
			Name:        v.Name,
			Unit:        v.Unit,
			StringValue: v.StringValue,
			DataValue:   v.DataValue,
			Time:        v.Time,
			UpdateTime:  v.UpdateTime,
			Link:        v.Link,
		}
		if v.Value != nil {
			messages[k].Value = *v.Value
		}
		if v.BoolValue != nil {
			messages[k].BoolValue = *v.BoolValue
		}
		if v.Sum != nil {
			messages[k].ValueSum = *v.Sum
		}
	}
	return messages
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
