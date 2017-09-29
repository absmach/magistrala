package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		msg := request.(writer.RawMessage)
		svc.Send(msg)
		return nil, nil
	}
}
