// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	mainflux "github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

var _ mainflux.AuthzServiceClient = (*Service)(nil)

type Service struct {
	mock.Mock
}

func (m *Service) Authorize(ctx context.Context, in *mainflux.AuthorizeReq, opts ...grpc.CallOption) (*mainflux.AuthorizeRes, error) {
	ret := m.Called(ctx, in)
	if in.GetSubject() == WrongID || in.GetSubject() == "" {
		return &mainflux.AuthorizeRes{}, errors.ErrAuthorization
	}
	if in.GetObject() == WrongID || in.GetObject() == "" {
		return &mainflux.AuthorizeRes{}, errors.ErrAuthorization
	}

	return ret.Get(0).(*mainflux.AuthorizeRes), ret.Error(1)
}
