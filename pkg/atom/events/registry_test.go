// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeInvalidator records every key it was asked to invalidate, optionally
// returning err on every call.
type fakeInvalidator struct {
	keys []string
	err  error
}

func (f *fakeInvalidator) Invalidate(_ context.Context, key string) error {
	f.keys = append(f.keys, key)
	return f.err
}

func TestRegistryInvalidateFansOutWithinFamily(t *testing.T) {
	translation := &fakeInvalidator{}
	authorizedSet := &fakeInvalidator{}

	r := NewRegistry()
	r.Register(FamilyTranslation, translation)
	r.Register(FamilyAuthorizedSet, authorizedSet)

	require.NoError(t, r.Invalidate(context.Background(), FamilyTranslation, "domain-1"))

	assert.Equal(t, []string{"domain-1"}, translation.keys)
	assert.Empty(t, authorizedSet.keys, "an invalidator must only hear about its own family")
}

func TestRegistryInvalidateCallsEveryInvalidatorInFamily(t *testing.T) {
	first := &fakeInvalidator{}
	second := &fakeInvalidator{}

	r := NewRegistry()
	r.Register(FamilyAuthorizedSet, first)
	r.Register(FamilyAuthorizedSet, second)

	require.NoError(t, r.Invalidate(context.Background(), FamilyAuthorizedSet, "domain-1"))

	assert.Equal(t, []string{"domain-1"}, first.keys)
	assert.Equal(t, []string{"domain-1"}, second.keys)
}

func TestRegistryInvalidateUnknownFamilyIsNoop(t *testing.T) {
	r := NewRegistry()
	r.Register(FamilyTranslation, &fakeInvalidator{})

	assert.NoError(t, r.Invalidate(context.Background(), "not-a-family", "domain-1"))
}

func TestRegistryInvalidateJoinsErrorsButKeepsGoing(t *testing.T) {
	failing := &fakeInvalidator{err: errors.New("boom")}
	healthy := &fakeInvalidator{}

	r := NewRegistry()
	r.Register(FamilyTranslation, failing)
	r.Register(FamilyTranslation, healthy)

	err := r.Invalidate(context.Background(), FamilyTranslation, "domain-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, failing.err)
	assert.Equal(t, []string{"domain-1"}, healthy.keys, "one invalidator's error must not stop its sibling from running")
}

func TestRegistryRegisterNilInvalidatorIsIgnored(t *testing.T) {
	r := NewRegistry()
	r.Register(FamilyTranslation, nil)

	assert.NoError(t, r.Invalidate(context.Background(), FamilyTranslation, "domain-1"))
}

func TestRegistryOneInvalidatorCanRegisterAgainstSeveralFamilies(t *testing.T) {
	shared := &fakeInvalidator{}

	r := NewRegistry()
	r.Register(FamilyTranslation, shared)
	r.Register(FamilyAuthorizedSet, shared)

	require.NoError(t, r.Invalidate(context.Background(), FamilyTranslation, "domain-1"))
	require.NoError(t, r.Invalidate(context.Background(), FamilyAuthorizedSet, "domain-1"))

	assert.Equal(t, []string{"domain-1", "domain-1"}, shared.keys)
}
