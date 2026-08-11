// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/policies"
	policymocks "github.com/absmach/magistrala/pkg/policies/mocks"
	"github.com/absmach/magistrala/readers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testDomain  = "domain-1"
	testSubject = "domain-1_user-1"
)

func TestIntersectPublishers(t *testing.T) {
	cases := []struct {
		desc       string
		requested  []string
		authorized map[string]struct{}
		want       []string
		noAccess   bool
	}{
		{
			desc:       "no filter requested returns the full authorized set",
			requested:  nil,
			authorized: set("a", "b", "c"),
			want:       []string{"a", "b", "c"},
		},
		{
			desc:       "no filter requested and nothing authorized denies access",
			requested:  nil,
			authorized: set(),
			noAccess:   true,
		},
		{
			desc:       "a subset of the authorized set is used as given",
			requested:  []string{"a"},
			authorized: set("a", "b", "c"),
			want:       []string{"a"},
		},
		{
			desc:       "ids outside the authorized set are dropped silently",
			requested:  []string{"a", "z"},
			authorized: set("a", "b"),
			want:       []string{"a"},
		},
		{
			desc:       "only unauthorized ids yields an empty result",
			requested:  []string{"z", "y"},
			authorized: set("a", "b"),
			noAccess:   true,
		},
		{
			desc:       "duplicate requested ids are deduplicated",
			requested:  []string{"a", "a"},
			authorized: set("a"),
			want:       []string{"a"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, noAccess, err := intersectPublishers(tc.requested, tc.authorized)
			require.NoError(t, err)
			assert.Equal(t, tc.noAccess, noAccess)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

// TestPublisherAuthorizerFilterDeniesUngrantedPublisher checks that a caller cannot read a
// publisher's messages just by naming it in the query, unless they hold a grant for it.
func TestPublisherAuthorizerFilterDeniesUngrantedPublisher(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: []string{"pub-granted"}}, nil)

	authz := newPublisherAuthorizer(evaluator, lister)

	publishers, noAccess, err := authz.filter(context.Background(), testDomain, testSubject, []string{"pub-ungranted"})
	require.NoError(t, err)
	assert.True(t, noAccess, "caller has no grant on pub-ungranted, so the request must be denied, not unfiltered")
	assert.Empty(t, publishers)
}

func TestPublisherAuthorizerFilterAllowsGrantedPublisher(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: []string{"pub-a", "pub-b"}}, nil)

	authz := newPublisherAuthorizer(evaluator, lister)

	publishers, noAccess, err := authz.filter(context.Background(), testDomain, testSubject, []string{"pub-a"})
	require.NoError(t, err)
	assert.False(t, noAccess)
	assert.Equal(t, []string{"pub-a"}, publishers)
}

func TestPublisherAuthorizerFilterNoGrantsAtAllIsEmptyNotUnfiltered(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: nil}, nil)

	authz := newPublisherAuthorizer(evaluator, lister)

	// No publisher filter supplied at all: an ordinary user with zero grants
	// must see nothing, not the whole unfiltered channel.
	publishers, noAccess, err := authz.filter(context.Background(), testDomain, testSubject, nil)
	require.NoError(t, err)
	assert.True(t, noAccess)
	assert.Empty(t, publishers)
}

func TestPublisherAuthorizerAdminBypassIsCapabilityDriven(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.ObjectType == policies.DomainType && pr.Permission == policies.AdminPermission
	})).Return(nil)
	// No ListAllObjects expectation: an admin bypass must never fall through
	// to the per-publisher lookup, since an admin's grant list is typically
	// empty (they hold a tenant-wide capability instead, not per-object
	// grants) and treating that empty list as "no restriction" would be the
	// same bug, upside down.

	authz := newPublisherAuthorizer(evaluator, lister)

	publishers, noAccess, err := authz.filter(context.Background(), testDomain, testSubject, []string{"any-publisher"})
	require.NoError(t, err)
	assert.False(t, noAccess)
	assert.Equal(t, []string{"any-publisher"}, publishers)
}

func TestPublisherAuthorizerNonAdminWithNoGrantsSeesNothingWhileAdminSeesEverything(t *testing.T) {
	adminEvaluator := policymocks.NewEvaluator(t)
	adminLister := policymocks.NewService(t)
	adminEvaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(nil)
	adminAuthz := newPublisherAuthorizer(adminEvaluator, adminLister)

	admins, adminNoAccess, err := adminAuthz.filter(context.Background(), testDomain, "domain-1_admin", nil)
	require.NoError(t, err)
	assert.False(t, adminNoAccess)
	assert.Empty(t, admins, "an unrestricted admin has no filter to report back, but that must not mean denied")

	userEvaluator := policymocks.NewEvaluator(t)
	userLister := policymocks.NewService(t)
	userEvaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	userLister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{}, nil)
	userAuthz := newPublisherAuthorizer(userEvaluator, userLister)

	_, userNoAccess, err := userAuthz.filter(context.Background(), testDomain, testSubject, nil)
	require.NoError(t, err)
	assert.True(t, userNoAccess)
}

// TestPublisherAuthorizerCacheExpires exercises the cache lifecycle: grant access, query, revoke,
// query again. Access must not disappear before the TTL elapses, and must not survive past it.
func TestPublisherAuthorizerCacheExpires(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: []string{"pub-a"}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: nil}, nil).Once()

	clock := time.Now()
	authz := newPublisherAuthorizer(evaluator, lister)
	authz.ttl = 50 * time.Millisecond
	authz.now = func() time.Time { return clock }

	publishers, noAccess, err := authz.filter(context.Background(), testDomain, testSubject, nil)
	require.NoError(t, err)
	require.False(t, noAccess)
	require.Equal(t, []string{"pub-a"}, publishers)

	// Access was revoked upstream, but within the TTL the cached grant must
	// still be served -- this is what makes it a cache, not an instant
	// pass-through.
	clock = clock.Add(10 * time.Millisecond)
	publishers, noAccess, err = authz.filter(context.Background(), testDomain, testSubject, nil)
	require.NoError(t, err)
	require.False(t, noAccess)
	require.Equal(t, []string{"pub-a"}, publishers)

	// Past the TTL, the revoked grant must take effect -- this is what
	// keeps the cache from becoming "never re-checked".
	clock = clock.Add(100 * time.Millisecond)
	_, noAccess, err = authz.filter(context.Background(), testDomain, testSubject, nil)
	require.NoError(t, err)
	assert.True(t, noAccess)
}

func TestPublisherAuthorizerPropagatesListerError(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{}, errors.New("atom unreachable"))

	authz := newPublisherAuthorizer(evaluator, lister)

	_, _, err := authz.filter(context.Background(), testDomain, testSubject, nil)
	assert.Error(t, err)
}

func TestRequestedPublishers(t *testing.T) {
	cases := []struct {
		desc string
		pm   readers.PageMetadata
		want []string
	}{
		{desc: "neither set", pm: readers.PageMetadata{}, want: nil},
		{desc: "singular only", pm: readers.PageMetadata{Publisher: "a"}, want: []string{"a"}},
		{desc: "plural only", pm: readers.PageMetadata{Publishers: []string{"a", "b"}}, want: []string{"a", "b"}},
		{desc: "both", pm: readers.PageMetadata{Publisher: "c", Publishers: []string{"a", "b"}}, want: []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.want, requestedPublishers(tc.pm))
		})
	}
}

func set(ids ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}
