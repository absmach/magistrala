// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/auth"
)

type MockSubjectSet struct {
	Object   string
	Relation string
}

type policyAgentMock struct {
	mu sync.Mutex
	// authzDb stores 'subject' as a key, and subject policies as a value.
	authzDB map[string][]MockSubjectSet
}

// NewKetoMock returns a mock service for Keto.
// This mock is not implemented yet.
func NewKetoMock(db map[string][]MockSubjectSet) auth.PolicyAgent {
	return &policyAgentMock{authzDB: db}
}

func (pa *policyAgentMock) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	for _, ss := range ssList {
		if ss.Object == pr.Object && ss.Relation == pr.Relation {
			return nil
		}
	}
	return auth.ErrAuthorization
}

func (pa *policyAgentMock) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.authzDB[pr.Subject] = append(pa.authzDB[pr.Subject], MockSubjectSet{Object: pr.Object, Relation: pr.Relation})
	return nil
}

func (pa *policyAgentMock) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	for k, ss := range ssList {
		if ss.Object == pr.Object && ss.Relation == pr.Relation {
			ssList[k] = MockSubjectSet{}
		}
	}
	return nil
}
