// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/absmach/magistrala"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

const WrongID = "wrongID"

var _ magistrala.AuthzServiceClient = (*Service)(nil)

type Service struct {
	mock.Mock
}

func (m *Service) Authorize(ctx context.Context, in *magistrala.AuthorizeReq, opts ...grpc.CallOption) (*magistrala.AuthorizeRes, error) {
	ret := m.Called(ctx, in)
	if in.GetSubject() == WrongID || in.GetSubject() == "" {
		return &magistrala.AuthorizeRes{}, svcerr.ErrAuthorization
	}
	if in.GetObject() == WrongID || in.GetObject() == "" {
		return &magistrala.AuthorizeRes{}, svcerr.ErrAuthorization
	}

	return ret.Get(0).(*magistrala.AuthorizeRes), ret.Error(1)
}
