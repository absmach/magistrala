package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		messages := request.([]writer.Message)
		svc.Send(messages)
		return nil, nil
	}
}
