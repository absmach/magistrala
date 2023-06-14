// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ policies.ThingsServiceClient = (*thingsClient)(nil)

// ServiceErrToken is used to simulate internal server error.
const ServiceErrToken = "unavailable"

type thingsClient struct {
	things map[string]string
}

// NewThingsClient returns mock implementation of things service client.
func NewThingsClient(data map[string]string) policies.ThingsServiceClient {
	return &thingsClient{data}
}

func (tc thingsClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, opts ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	secret := req.GetSub()

	// Since there is no appropriate way to simulate internal server error,
	// we had to use this obscure approach. ErrorToken simulates gRPC
	// call which returns internal server error.
	if secret == ServiceErrToken {
		return &policies.AuthorizeRes{ThingID: "", Authorized: false}, status.Error(codes.Internal, "internal server error")
	}

	if secret == "" {
		return &policies.AuthorizeRes{ThingID: "", Authorized: false}, errors.ErrAuthentication
	}

	id, ok := tc.things[secret]
	if !ok {
		return &policies.AuthorizeRes{ThingID: "", Authorized: false}, status.Error(codes.Unauthenticated, "invalid credentials provided")
	}
	return &policies.AuthorizeRes{ThingID: id, Authorized: true}, nil
}

func (tc thingsClient) Identify(ctx context.Context, req *policies.Key, opts ...grpc.CallOption) (*policies.ClientID, error) {
	panic("not implemented")
}
