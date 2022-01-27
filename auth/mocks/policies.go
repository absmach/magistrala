// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
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
	return errors.ErrAuthorization
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

func (pa *policyAgentMock) RetrievePolicies(ctx context.Context, pr auth.PolicyReq) ([]*acl.RelationTuple, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	tuple := []*acl.RelationTuple{}
	for _, ss := range ssList {
		if ss.Relation == pr.Relation {
			tuple = append(tuple, &acl.RelationTuple{Object: ss.Object, Relation: ss.Relation})
		}
	}
	return tuple, nil
}
