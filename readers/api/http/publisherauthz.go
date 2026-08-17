// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"sync"
	"time"

	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/readers"
)

// publisherAuthzCacheTTL is kept short since it backs a security control, not just a performance cache.
const publisherAuthzCacheTTL = 30 * time.Second

// publisherGrant is what a (domain, subject) pair may read: either every publisher, or a bounded set of IDs.
type publisherGrant struct {
	unrestricted bool
	publishers   map[string]struct{}
	expiresAt    time.Time
}

// publisherAuthorizer resolves which publishers a caller may read on a channel and intersects that with the request's publisher filter.
type publisherAuthorizer struct {
	evaluator policies.Evaluator
	lister    policies.Service
	ttl       time.Duration
	now       func() time.Time

	mu    sync.Mutex
	cache map[string]publisherGrant
}

func newPublisherAuthorizer(evaluator policies.Evaluator, lister policies.Service) *publisherAuthorizer {
	return &publisherAuthorizer{
		evaluator: evaluator,
		lister:    lister,
		ttl:       publisherAuthzCacheTTL,
		now:       time.Now,
		cache:     make(map[string]publisherGrant),
	}
}

// filter returns the intersection of the requested publishers and what the caller is authorized to read.
// noAccess means the caller may read nothing here; treat that as zero rows, never as an unfiltered read.
func (a *publisherAuthorizer) filter(ctx context.Context, domain, subject string, requested []string) (publishers []string, noAccess bool, err error) {
	grant, err := a.resolve(ctx, domain, subject)
	if err != nil {
		return nil, false, err
	}
	if grant.unrestricted {
		return requested, false, nil
	}
	return intersectPublishers(requested, grant.publishers)
}

// intersectPublishers widens an empty request to the full authorized set and narrows a non-empty
// one to its authorized part; an empty result either way is reported as noAccess.
func intersectPublishers(requested []string, authorized map[string]struct{}) (publishers []string, noAccess bool, err error) {
	if len(requested) == 0 {
		if len(authorized) == 0 {
			return nil, true, nil
		}
		out := make([]string, 0, len(authorized))
		for id := range authorized {
			out = append(out, id)
		}
		return out, false, nil
	}

	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(requested))
	for _, id := range requested {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := authorized[id]; ok {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil, true, nil
	}
	return out, false, nil
}

func (a *publisherAuthorizer) resolve(ctx context.Context, domain, subject string) (publisherGrant, error) {
	key := domain + "\x00" + subject

	a.mu.Lock()
	grant, ok := a.cache[key]
	a.mu.Unlock()
	if ok && a.now().Before(grant.expiresAt) {
		return grant, nil
	}

	grant, err := a.load(ctx, domain, subject)
	if err != nil {
		return publisherGrant{}, err
	}
	grant.expiresAt = a.now().Add(a.ttl)

	a.mu.Lock()
	a.cache[key] = grant
	a.mu.Unlock()

	return grant, nil
}

func (a *publisherAuthorizer) load(ctx context.Context, domain, subject string) (publisherGrant, error) {
	if a.isDomainAdmin(ctx, domain, subject) {
		return publisherGrant{unrestricted: true}, nil
	}

	page, err := a.lister.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subject,
		Domain:      domain,
		ObjectType:  policies.ClientType,
		Permission:  policies.ViewPermission,
	})
	if err != nil {
		return publisherGrant{}, err
	}

	publishers := make(map[string]struct{}, len(page.Policies))
	for _, id := range page.Policies {
		publishers[id] = struct{}{}
	}
	return publisherGrant{publishers: publishers}, nil
}

// isDomainAdmin checks a tenant-wide admin capability rather than inferring it from a role name.
// CheckPolicy reports denial and request failure alike as a non-nil error, so both count as "not
// admin" here; a real Atom outage still surfaces as an error from the per-publisher lookup below.
func (a *publisherAuthorizer) isDomainAdmin(ctx context.Context, domain, subject string) bool {
	err := a.evaluator.CheckPolicy(ctx, policies.Policy{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subject,
		Domain:      domain,
		ObjectType:  policies.DomainType,
		Object:      domain,
		Permission:  policies.AdminPermission,
	})
	return err == nil
}

// requestedPublishers collects the publisher IDs the caller asked to filter
// by, from either the singular or plural query representation.
func requestedPublishers(pm readers.PageMetadata) []string {
	if pm.Publisher == "" {
		return pm.Publishers
	}
	requested := make([]string, 0, len(pm.Publishers)+1)
	requested = append(requested, pm.Publishers...)
	requested = append(requested, pm.Publisher)
	return requested
}
