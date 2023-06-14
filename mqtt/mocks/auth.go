package mocks

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
	"google.golang.org/grpc"
)

const separator = ":"

type mockClient struct {
	ids   map[string]string   // (password,id) for Identify
	elems map[string][]string // (password:channel, []actions) for Authorize
}

// NewClient returns a new mock Things gRPC client.
func NewClient(ids map[string]string, elems map[string][]string) policies.ThingsServiceClient {
	return mockClient{elems: elems, ids: ids}
}

func (cli mockClient) Authorize(ctx context.Context, ar *policies.AuthorizeReq, opts ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	actions, ok := cli.elems[Key(ar)]
	if !ok {
		return &policies.AuthorizeRes{Authorized: false}, nil
	}
	for _, a := range actions {
		if a == ar.Act {
			return &policies.AuthorizeRes{ThingID: ar.Sub, Authorized: true}, nil
		}
	}
	return &policies.AuthorizeRes{Authorized: false}, nil
}

func (cli mockClient) Identify(ctx context.Context, in *policies.Key, opts ...grpc.CallOption) (*policies.ClientID, error) {
	if id, ok := cli.ids[in.GetValue()]; ok {
		return &policies.ClientID{Value: id}, nil
	}
	return &policies.ClientID{}, errors.ErrAuthentication
}

// Key generates key for internal auth map.
func Key(ar *policies.AuthorizeReq) string {
	return fmt.Sprintf("%s%s%s", ar.Sub, separator, ar.Obj)
}
