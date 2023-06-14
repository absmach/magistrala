// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/things/policies"
	"github.com/stretchr/testify/mock"
)

type Repository struct {
	mock.Mock
}

func (m *Repository) Delete(ctx context.Context, p policies.Policy) error {
	ret := m.Called(ctx, p)

	return ret.Error(0)
}

func (m *Repository) Retrieve(ctx context.Context, pm policies.Page) (policies.PolicyPage, error) {
	ret := m.Called(ctx, pm)

	return ret.Get(0).(policies.PolicyPage), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, p policies.Policy) (policies.Policy, error) {
	ret := m.Called(ctx, p)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) Update(ctx context.Context, p policies.Policy) (policies.Policy, error) {
	ret := m.Called(ctx, p)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) EvaluateMessagingAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	ret := m.Called(ctx, ar)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) EvaluateThingAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	ret := m.Called(ctx, ar)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}

func (m *Repository) EvaluateGroupAccess(ctx context.Context, ar policies.AccessRequest) (policies.Policy, error) {
	ret := m.Called(ctx, ar)

	return ret.Get(0).(policies.Policy), ret.Error(1)
}
