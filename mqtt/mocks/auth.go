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
func NewClient(ids map[string]string, elems map[string][]string) policies.AuthServiceClient {
	return mockClient{elems: elems, ids: ids}
}

func (cli mockClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, opts ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	actions, ok := cli.elems[Key(req)]
	if !ok {
		return &policies.AuthorizeRes{Authorized: false}, nil
	}
	for _, a := range actions {
		if a == req.GetAction() {
			return &policies.AuthorizeRes{ThingID: req.GetSubject(), Authorized: true}, nil
		}
	}
	return &policies.AuthorizeRes{Authorized: false}, nil
}

func (cli mockClient) Identify(ctx context.Context, req *policies.IdentifyReq, opts ...grpc.CallOption) (*policies.IdentifyRes, error) {
	if id, ok := cli.ids[req.GetSecret()]; ok {
		return &policies.IdentifyRes{Id: id}, nil
	}
	return &policies.IdentifyRes{}, errors.ErrAuthentication
}

// Key generates key for internal auth map.
func Key(ar *policies.AuthorizeReq) string {
	return fmt.Sprintf("%s%s%s", ar.GetSubject(), separator, ar.GetObject())
}
