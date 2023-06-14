// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
	"google.golang.org/grpc"
)

var _ policies.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
}

// NewThingsService returns mock implementation of things service.
func NewThingsService(channels map[string]string) policies.ThingsServiceClient {
	return &thingsServiceMock{channels}
}

func (svc thingsServiceMock) AuthorizeByKey(ctx context.Context, in *policies.AuthorizeReq, opts ...grpc.CallOption) (*policies.ClientID, error) {
	token := in.GetSub()
	if token == "invalid" || token == "" {
		return nil, errors.ErrAuthentication
	}

	return &policies.ClientID{Value: token}, nil
}

func (svc thingsServiceMock) Authorize(context.Context, *policies.AuthorizeReq, ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	return &policies.AuthorizeRes{Authorized: true}, nil
}

func (svc thingsServiceMock) Identify(context.Context, *policies.Key, ...grpc.CallOption) (*policies.ClientID, error) {
	panic("not implemented")
}
