//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errUnauthorized = status.Error(codes.PermissionDenied, "missing or invalid credentials provided")

var _ mainflux.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct{}

// NewThingsService returns mock implementation of things service
func NewThingsService() mainflux.ThingsServiceClient {
	return thingsServiceMock{}
}

func (svc thingsServiceMock) CanAccess(ctx context.Context, in *mainflux.AccessReq, opts ...grpc.CallOption) (*mainflux.ThingID, error) {
	token := in.GetToken()
	if token == "invalid" {
		return nil, errUnauthorized
	}

	if token == "" {
		return nil, errUnauthorized
	}

	return &mainflux.ThingID{Value: token}, nil
}

func (svc thingsServiceMock) Identify(_ context.Context, _ *mainflux.Token, _ ...grpc.CallOption) (*mainflux.ThingID, error) {
	return nil, nil
}
