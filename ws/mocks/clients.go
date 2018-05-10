package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/clients"

	"google.golang.org/grpc"
)

var _ mainflux.ClientsServiceClient = (*clientsClient)(nil)

type clientsClient struct {
	clients map[string]string
}

// NewClientsClient returns mock implementation of clients service client.
func NewClientsClient(data map[string]string) mainflux.ClientsServiceClient {
	return &clientsClient{data}
}

func (client clientsClient) CanAccess(ctx context.Context, req *mainflux.AccessReq, opts ...grpc.CallOption) (*mainflux.Identity, error) {
	key := req.GetToken()
	if key == "" {
		return nil, clients.ErrUnauthorizedAccess
	}

	id, ok := client.clients[key]
	if !ok {
		return nil, clients.ErrUnauthorizedAccess
	}

	return &mainflux.Identity{Value: id}, nil
}
