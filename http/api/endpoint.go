package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
)

func sendMessageEndpoint(svc mainflux.MessagePublisher) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		msg := request.(mainflux.RawMessage)
		err := svc.Publish(msg)
		return nil, err
	}
}
