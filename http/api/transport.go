package api

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/asaskevich/govalidator"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/clients"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const protocol string = "http"

var (
	errMalformedData = errors.New("malformed SenML data")
	errNotFound      = errors.New("non-existent entity")
	auth             mainflux.ClientsServiceClient
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc mainflux.MessagePublisher, cc mainflux.ClientsServiceClient) http.Handler {
	auth = cc

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

	msg := mainflux.RawMessage{
		Publisher:   publisher,
		Protocol:    protocol,
		ContentType: r.Header.Get("Content-Type"),
		Channel:     bone.GetValue(r, "id"),
		Payload:     payload,
	}

	return msg, nil
}

func authorize(r *http.Request) (string, error) {
	apiKey := r.Header.Get("Authorization")

	if apiKey == "" {
		return "", clients.ErrUnauthorizedAccess
	}

	// extract ID from /channels/:id/messages
	c := bone.GetValue(r, "id")
	if !govalidator.IsUUID(c) {
		return "", errNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := auth.CanAccess(ctx, &mainflux.AccessReq{Token: apiKey, ChanID: c})
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
	case errNotFound:
		w.WriteHeader(http.StatusNotFound)
	case clients.ErrUnauthorizedAccess:
		w.WriteHeader(http.StatusUnauthorized)
	case clients.ErrAccessForbidden:
		w.WriteHeader(http.StatusForbidden)
	case errNotFound:
		w.WriteHeader(http.StatusNotFound)
	default:
		if e, ok := status.FromError(err); ok {
			switch e.Code() {
			case codes.Unauthenticated:
				w.WriteHeader(http.StatusUnauthorized)
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
