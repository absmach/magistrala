// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"
)

var _ mainflux.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
}

// NewThingsService returns mock implementation of things service
func NewThingsService(channels map[string]string) mainflux.ThingsServiceClient {
	return &thingsServiceMock{channels}
}

func (svc thingsServiceMock) CanAccessByKey(ctx context.Context, in *mainflux.AccessByKeyReq, opts ...grpc.CallOption) (*mainflux.ThingID, error) {
	token := in.GetToken()
	if token == "invalid" {
		return nil, errors.ErrAuthentication
	}

	if token == "" {
		return nil, errors.ErrAuthentication
	}

	if token == "token" {
		return nil, errors.ErrAuthorization
	}

	return &mainflux.ThingID{Value: token}, nil
}

func (svc thingsServiceMock) CanAccessByID(context.Context, *mainflux.AccessByIDReq, ...grpc.CallOption) (*empty.Empty, error) {
	panic("not implemented")
}

func (svc thingsServiceMock) IsChannelOwner(ctx context.Context, in *mainflux.ChannelOwnerReq, opts ...grpc.CallOption) (*empty.Empty, error) {
	if id, ok := svc.channels[in.GetOwner()]; ok {
		if id == in.ChanID {
			return nil, nil
		}
	}
	return nil, errors.ErrAuthorization
}

func (svc thingsServiceMock) Identify(context.Context, *mainflux.Token, ...grpc.CallOption) (*mainflux.ThingID, error) {
	panic("not implemented")
}
