// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	context "context"

	auth "github.com/absmach/magistrala/auth"
	"github.com/stretchr/testify/mock"
)

var _ auth.KeyRepository = (*Keys)(nil)

type Keys struct {
	mock.Mock
}

func (m *Keys) Save(ctx context.Context, key auth.Key) (string, error) {
	ret := m.Called(ctx, key)

	return ret.String(0), ret.Error(1)
}

func (m *Keys) Retrieve(ctx context.Context, issuer, id string) (auth.Key, error) {
	ret := m.Called(ctx, issuer, id)

	return ret.Get(0).(auth.Key), ret.Error(1)
}

func (m *Keys) Remove(ctx context.Context, issuer, id string) error {
	ret := m.Called(ctx, issuer, id)

	return ret.Error(0)
}
