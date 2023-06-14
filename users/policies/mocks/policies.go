// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/users/policies"
	"github.com/stretchr/testify/mock"
)

type Repository struct {
	mock.Mock
}

func (m *Repository) Delete(ctx context.Context, p policies.Policy) error {
	ret := m.Called(ctx, p)

	return ret.Error(0)
}

func (m *Repository) RetrieveAll(ctx context.Context, pm policies.Page) (policies.PolicyPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(policies.PolicyPage), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, p policies.Policy) error {
	ret := m.Called(ctx, p)

	return ret.Error(0)
}

func (m *Repository) Update(ctx context.Context, p policies.Policy) error {
	ret := m.Called(ctx, p)

	return ret.Error(0)
}

func (m *Repository) EvaluateUserAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	ret := m.Called(ctx, ar)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) EvaluateGroupAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	ret := m.Called(ctx, ar)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) CheckAdmin(ctx context.Context, id string) error {
	ret := m.Called(ctx, id)

	return ret.Error(0)
}
