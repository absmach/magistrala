// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	context "context"

	auth "github.com/absmach/magistrala/auth"
	"github.com/stretchr/testify/mock"
)

var (
	_ auth.Authz       = (*Authz)(nil)
	_ auth.PolicyAgent = (*PolicyAgent)(nil)
)

type Authz struct {
	mock.Mock
}

func (m *Authz) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	ret := m.Called(ctx, prs)

	return ret.Error(0)
}

func (m *Authz) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *Authz) Authorize(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *Authz) CountObjects(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(int), ret.Error(1)
}

func (m *Authz) CountSubjects(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(int), ret.Error(1)
}

func (m *Authz) DeletePolicies(ctx context.Context, prs []auth.PolicyReq) error {
	ret := m.Called(ctx, prs)

	return ret.Error(0)
}

func (m *Authz) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *Authz) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(auth.PolicyPage), ret.Error(1)
}

func (m *Authz) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(auth.PolicyPage), ret.Error(1)
}

func (m *Authz) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (auth.PolicyPage, error) {
	ret := m.Called(ctx, pr, nextPageToken, limit)

	return ret.Get(0).(auth.PolicyPage), ret.Error(1)
}

func (m *Authz) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (auth.PolicyPage, error) {
	ret := m.Called(ctx, pr, nextPageToken, limit)

	return ret.Get(0).(auth.PolicyPage), ret.Error(1)
}

type PolicyAgent struct {
	mock.Mock
}

func (m *PolicyAgent) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	ret := m.Called(ctx, prs)

	return ret.Error(0)
}

func (m *PolicyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *PolicyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *PolicyAgent) DeletePolicies(ctx context.Context, pr []auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *PolicyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	ret := m.Called(ctx, pr)

	return ret.Error(0)
}

func (m *PolicyAgent) RetrieveAllObjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).([]auth.PolicyRes), ret.Error(1)
}

func (m *PolicyAgent) RetrieveAllObjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(int), ret.Error(1)
}

func (m *PolicyAgent) RetrieveAllSubjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).([]auth.PolicyRes), ret.Error(1)
}

func (m *PolicyAgent) RetrieveAllSubjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	ret := m.Called(ctx, pr)

	return ret.Get(0).(int), ret.Error(1)
}

func (m *PolicyAgent) RetrieveObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
	ret := m.Called(ctx, pr, nextPageToken, limit)

	return ret.Get(0).([]auth.PolicyRes), ret.String(1), ret.Error(2)
}

func (m *PolicyAgent) RetrieveSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
	ret := m.Called(ctx, pr, nextPageToken, limit)

	return ret.Get(0).([]auth.PolicyRes), ret.String(1), ret.Error(2)
}
