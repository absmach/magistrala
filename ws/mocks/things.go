//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.ThingsServiceClient = (*thingsClient)(nil)

// ServiceErrToken is used to simulate internal server error.
const ServiceErrToken = "unavailable"

type thingsClient struct {
	things map[string]string
}

// NewThingsClient returns mock implementation of things service client.
func NewThingsClient(data map[string]string) mainflux.ThingsServiceClient {
	return &thingsClient{data}
}

func (tc thingsClient) CanAccess(ctx context.Context, req *mainflux.AccessReq, opts ...grpc.CallOption) (*mainflux.ThingID, error) {
	key := req.GetToken()

	// Since there is no appropriate way to simulate internal server error,
	// we had to use this obscure approach. ErrorToken simulates gRPC
	// call which returns internal server error.
	if key == ServiceErrToken {
		return nil, status.Error(codes.Internal, "internal server error")
	}
	if key == "" {
		return nil, things.ErrUnauthorizedAccess
	}

	id, ok := tc.things[key]
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "invalid credentials provided")
	}

	return &mainflux.ThingID{Value: id}, nil
}

func (tc thingsClient) CanAccessByID(context.Context, *mainflux.AccessByIDReq, ...grpc.CallOption) (*empty.Empty, error) {
	panic("not implemented")
}

func (tc thingsClient) Identify(context.Context, *mainflux.Token, ...grpc.CallOption) (*mainflux.ThingID, error) {
	panic("not implemented")
}
