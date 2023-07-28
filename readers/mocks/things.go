// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/things/policies"
	"google.golang.org/grpc"
)

var _ policies.AuthServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
}

// NewThingsService returns mock implementation of things service.
func NewThingsService(channels map[string]string) policies.AuthServiceClient {
	return &thingsServiceMock{channels}
}

func (svc thingsServiceMock) Authorize(context.Context, *policies.AuthorizeReq, ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	return &policies.AuthorizeRes{Authorized: true}, nil
}

func (svc thingsServiceMock) Identify(context.Context, *policies.IdentifyReq, ...grpc.CallOption) (*policies.IdentifyRes, error) {
	panic("not implemented")
}
