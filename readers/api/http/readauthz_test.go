// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/atom"
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

// The PRD's worked example: three meters on one channel, a customer granted
// meters 1 and 3. Each device has two identities on a message row — its platform
// UUID in the publisher column when it publishes for itself, and its external
// serial in device_id when a gateway relays for it.
const (
	meter1UUID = "uuid-meter-1"
	meter2UUID = "uuid-meter-2"
	meter3UUID = "uuid-meter-3"

	meter1Serial = "Meter.1-01:X"
	meter2Serial = "Meter.2-02:Y"
	meter3Serial = "Meter.3-03:Z"

	gatewayUUID   = "uuid-gateway-1"
	gatewaySerial = "Gateway.1-01:X"
)

// fakeResolver stands in for Atom's UUID → device info lookup.
type fakeResolver struct {
	serials map[string]string
	// gateways marks ids flagged attributes.is_gateway (spec §8 A12).
	gateways map[string]struct{}
	// unresolvable ids are omitted from the result entirely, as a deleted or
	// unreadable entity would be — distinct from an id merely absent from
	// serials, which still resolves with an empty external id.
	unresolvable map[string]struct{}
	err          error
	calls        int
	lastIDs      []string
}

func (f *fakeResolver) EntityDeviceInfo(_ context.Context, ids []string) (map[string]atom.DeviceInfo, error) {
	f.calls++
	f.lastIDs = append([]string{}, ids...)
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]atom.DeviceInfo, len(ids))
	for _, id := range ids {
		if _, unresolvable := f.unresolvable[id]; unresolvable {
			continue
		}
		_, isGateway := f.gateways[id]
		out[id] = atom.DeviceInfo{ExternalID: f.serials[id], IsGateway: isGateway}
	}
	return out, nil
}

func allMeters() *fakeResolver {
	return &fakeResolver{serials: map[string]string{
		meter1UUID: meter1Serial,
		meter2UUID: meter2Serial,
		meter3UUID: meter3Serial,
		"pub-a":    "serial-pub-a",
		"pub-b":    "serial-pub-b",
		"pub-c":    "serial-pub-c",
	}}
}

// customerAuthorizer is a non-admin granted exactly the given devices.
func customerAuthorizer(t *testing.T, resolver deviceInfoResolver, granted ...string) *readAuthorizer {
	t.Helper()

	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: granted}, nil)

	return newReadAuthorizer(evaluator, lister, resolver)
}

// Criterion 1: a user granted read on meters 1 and 3, querying a channel
// carrying all three, receives data for 1 and 3 only. Both identities of each
// granted device are in scope, and meter 2's are in neither.
func TestResolveBoundsScopeToGrantedDevices(t *testing.T) {
	authz := customerAuthorizer(t, allMeters(), meter1UUID, meter3UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.False(t, got.noAccess)
	require.NotNil(t, got.scope)

	assert.Equal(t, []string{meter1UUID, meter3UUID}, got.scope.PublisherIDs)
	assert.Equal(t, []string{meter1Serial, meter3Serial}, got.scope.DeviceIDs)
	assert.NotContains(t, got.scope.PublisherIDs, meter2UUID)
	assert.NotContains(t, got.scope.DeviceIDs, meter2Serial)

	// Neither granted device is flagged is_gateway, so both get R1's
	// unconditional self-publish trust.
	assert.Equal(t, []string{meter1UUID, meter3UUID}, got.scope.SelfPublisherIDs)

	// Asking for nothing leaves the caller's own filters unset, so the scope
	// alone bounds the query.
	assert.Nil(t, got.publishers)
	assert.Nil(t, got.deviceIDs)
}

// A regression test for R1: a held device flagged attributes.is_gateway
// (spec §8 A12) must still land in PublisherIDs and DeviceIDs — it can be
// relayed for and read like any device — but must not land in
// SelfPublisherIDs, since it is capable of relaying for someone else and
// publisher-only matching would readmit every device it has ever relayed for
// (finding 02). A non-gateway device in the same grant is unaffected.
func TestResolveExcludesGatewaysFromSelfPublisherIDs(t *testing.T) {
	resolver := &fakeResolver{
		serials:  map[string]string{meter1UUID: meter1Serial, gatewayUUID: gatewaySerial},
		gateways: map[string]struct{}{gatewayUUID: {}},
	}
	authz := customerAuthorizer(t, resolver, meter1UUID, gatewayUUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, got.scope)

	assert.ElementsMatch(t, []string{meter1UUID, gatewayUUID}, got.scope.PublisherIDs)
	assert.ElementsMatch(t, []string{meter1Serial, gatewaySerial}, got.scope.DeviceIDs)
	assert.Equal(t, []string{meter1UUID}, got.scope.SelfPublisherIDs, "the gateway-flagged device must not get unconditional publisher trust")
}

// A regression test for R1: an id the resolver could not confirm either way
// — a permission failure, a transient error on that one entity — must be
// excluded from SelfPublisherIDs, the safe direction (see the load() doc
// comment). It must still land in PublisherIDs, so the existing device_id =
// ” leg keeps working for it.
func TestResolveTreatsUnresolvedDeviceInfoAsGatewayForSafety(t *testing.T) {
	resolver := &fakeResolver{
		serials:      map[string]string{meter1UUID: meter1Serial},
		unresolvable: map[string]struct{}{meter3UUID: {}},
	}
	authz := customerAuthorizer(t, resolver, meter1UUID, meter3UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, got.scope)

	assert.ElementsMatch(t, []string{meter1UUID, meter3UUID}, got.scope.PublisherIDs)
	assert.Equal(t, []string{meter1UUID}, got.scope.SelfPublisherIDs, "an unresolved device must not get unconditional publisher trust")
}

// Criterion 2: the same user requesting meter 2 receives empty, not meter 2's
// data. Denied, never silently widened back to the full authorized set.
func TestResolveUnauthorizedDeviceIsDenied(t *testing.T) {
	authz := customerAuthorizer(t, allMeters(), meter1UUID, meter3UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, []string{meter2Serial})
	require.NoError(t, err, "an unauthorized id must be dropped, not raised as an error that reveals it")
	assert.True(t, got.noAccess)
	assert.Nil(t, got.scope)
	assert.Empty(t, got.deviceIDs)
}

// Criteria 3 and 7: requesting meters 1 and 2 returns meter 1 only, and the
// unauthorized id is dropped silently rather than reported.
func TestResolveMixedRequestKeepsOnlyAuthorized(t *testing.T) {
	authz := customerAuthorizer(t, allMeters(), meter1UUID, meter3UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, []string{meter1Serial, meter2Serial})
	require.NoError(t, err)
	require.False(t, got.noAccess)
	assert.Equal(t, []string{meter1Serial}, got.deviceIDs)
	assert.NotContains(t, got.deviceIDs, meter2Serial)
}

// Criterion 4, the empty-set inversion. A user with no device grants must
// receive nothing. Reading an empty authorized set as "no filter" would give the
// caller with the fewest permissions the most data.
func TestResolveNoGrantsIsEmptyNotEverything(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: nil}, nil)

	authz := newReadAuthorizer(evaluator, lister, allMeters())

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.True(t, got.noAccess, "zero authorized devices must mean zero rows, never an unfiltered read")
	assert.Nil(t, got.scope)
}

// The same inversion one layer down: even if a caller-supplied empty scope did
// reach the query, it must exclude every row rather than vanish from the WHERE
// clause the way an empty convenience filter does.
func TestEmptyDeviceScopeMatchesNothing(t *testing.T) {
	scope := &readers.DeviceScope{}

	assert.True(t, scope.Empty())
	assert.Equal(t, []string{}, scope.Publishers(), "must bind an empty array, not NULL")
	assert.Equal(t, []string{}, scope.Devices(), "must bind an empty array, not NULL")

	var unset *readers.DeviceScope
	assert.False(t, unset.Empty(), "an absent scope is unbounded, which is not the same as bounded-to-nothing")
}

// Criterion 5: a domain admin receives all three, and is not narrowed at all.
func TestResolveDomainAdminIsUnrestricted(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.ObjectType == policies.DomainType && pr.Permission == policies.AdminPermission
	})).Return(nil)
	// No ListAllObjects expectation: the bypass must never fall through to the
	// per-device lookup, whose empty result for an admin would otherwise have to
	// be interpreted — and interpreting it is the bug in criterion 4, upside down.

	authz := newReadAuthorizer(evaluator, lister, allMeters())

	got, err := authz.resolve(context.Background(), testDomain, "domain-1_admin", nil, nil)
	require.NoError(t, err)
	assert.False(t, got.noAccess)
	assert.Nil(t, got.scope, "an admin's query is unbounded")
}

// The admin bypass is capability-driven, not inferred from the size of a grant
// list. An admin with zero per-device grants and a non-admin with many must both
// come out right — the two cases are indistinguishable from the grant list alone.
func TestAdminBypassIsCapabilityDrivenNotGrantCount(t *testing.T) {
	adminEvaluator := policymocks.NewEvaluator(t)
	adminLister := policymocks.NewService(t)
	adminEvaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(nil)
	admin := newReadAuthorizer(adminEvaluator, adminLister, allMeters())

	adminScope, err := admin.resolve(context.Background(), testDomain, "domain-1_admin", nil, nil)
	require.NoError(t, err)
	assert.False(t, adminScope.noAccess, "an admin with no per-device grants still reads everything")
	assert.Nil(t, adminScope.scope)

	granted := make([]string, 0, 500)
	serialsByUUID := make(map[string]string, 500)
	for i := 0; i < 500; i++ {
		id := fmt.Sprintf("uuid-%d", i)
		granted = append(granted, id)
		serialsByUUID[id] = fmt.Sprintf("serial-%d", i)
	}
	user := customerAuthorizer(t, &fakeResolver{serials: serialsByUUID}, granted...)

	userScope, err := user.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.False(t, userScope.noAccess)
	require.NotNil(t, userScope.scope, "a large grant set is still a bounded one")
	assert.Len(t, userScope.scope.PublisherIDs, 500)
	assert.Len(t, userScope.scope.DeviceIDs, 500)
}

// Criterion 6: publishers get the same treatment, on the UUID projection.
func TestResolvePublishersBehaveTheSame(t *testing.T) {
	cases := []struct {
		desc      string
		requested []string
		want      []string
		noAccess  bool
	}{
		{desc: "nothing requested leaves the filter unset", requested: nil, want: nil},
		{desc: "a granted publisher is used as given", requested: []string{meter1UUID}, want: []string{meter1UUID}},
		{desc: "an ungranted publisher is dropped", requested: []string{meter1UUID, meter2UUID}, want: []string{meter1UUID}},
		{desc: "only ungranted publishers is denied", requested: []string{meter2UUID}, noAccess: true},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authz := customerAuthorizer(t, allMeters(), meter1UUID, meter3UUID)

			got, err := authz.resolve(context.Background(), testDomain, testSubject, tc.requested, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.noAccess, got.noAccess)
			assert.Equal(t, tc.want, got.publishers)
		})
	}
}

// Criterion 10: the UUID → external_id translation is what makes a granted
// device's gateway-relayed data reachable. Without it the device half of the
// scope is empty and the customer sees an authorization failure that is really a
// missing conversion.
func TestResolveTranslatesUUIDsToSerials(t *testing.T) {
	resolver := allMeters()
	authz := customerAuthorizer(t, resolver, meter1UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, got.scope)

	assert.Equal(t, []string{meter1UUID}, resolver.lastIDs, "the authorized UUIDs are what gets translated")
	assert.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)
	assert.NotEqual(t, got.scope.PublisherIDs, got.scope.DeviceIDs, "serials are not UUIDs")
}

// A granted device that the resolver has no external id for still
// contributes its publisher identity — EntityExternalIDs documents a
// missing key as "no external identity", not "does not exist", so this is
// the ordinary case of a device that was never given a serial, not evidence
// it was deleted. Dropping the device entirely, as an earlier version of
// this code did, made every self-published row from such a device
// unreadable by its own authorized owner: the device is only absent from
// the serial (device_id) projection, the leg gateway-relayed rows are
// matched against.
func TestResolveKeepsPublisherIdentityForUnresolvedEntityIDs(t *testing.T) {
	authz := customerAuthorizer(t, &fakeResolver{serials: map[string]string{meter1UUID: meter1Serial}}, meter1UUID, meter3UUID)

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, got.scope)

	assert.Equal(t, []string{meter1UUID, meter3UUID}, got.scope.PublisherIDs)
	assert.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)
	assert.NotContains(t, got.scope.DeviceIDs, "")
}

// A missing resolver must fail loudly. Skipping the translation silently would
// present as "this customer has no access" for every gateway-relayed device.
func TestResolveWithoutResolverIsAnError(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil)

	authz := newReadAuthorizer(evaluator, lister, nil)

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	assert.ErrorIs(t, err, errNoDeviceInfoResolver)
}

func TestResolvePropagatesResolverError(t *testing.T) {
	authz := customerAuthorizer(t, &fakeResolver{err: errors.New("atom unreachable")}, meter1UUID)

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	assert.Error(t, err)
}

func TestResolvePropagatesListerError(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{}, errors.New("atom unreachable"))

	authz := newReadAuthorizer(evaluator, lister, allMeters())

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	assert.Error(t, err)
}

// Criterion 12: grant, query, revoke, query again. The TTL is the revocation
// SLA, so access must survive within it and must not survive past it.
func TestReadAuthorizerCacheExpires(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: nil}, nil).Once()

	clock := time.Now()
	resolver := allMeters()
	authz := newReadAuthorizer(evaluator, lister, resolver)
	authz.ttl = 50 * time.Millisecond
	authz.now = func() time.Time { return clock }

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.False(t, got.noAccess)
	require.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)

	// Within the TTL the cached grant is served, translation included — that is
	// what makes it a cache rather than a pass-through.
	clock = clock.Add(10 * time.Millisecond)
	got, err = authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.False(t, got.noAccess)
	require.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)
	assert.Equal(t, 1, resolver.calls, "the translation is cached alongside the authorized set")

	// Past the TTL the revocation takes effect.
	clock = clock.Add(100 * time.Millisecond)
	got, err = authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.True(t, got.noAccess, "a revoked grant must not outlive the TTL")
}

func TestReadAuthorizerCacheIsKeyedPerSubjectAndDomain(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == testSubject
	})).Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == "domain-1_user-2"
	})).Return(policies.PolicyPage{Policies: []string{meter3UUID}}, nil).Once()

	authz := newReadAuthorizer(evaluator, lister, allMeters())

	first, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	second, err := authz.resolve(context.Background(), testDomain, "domain-1_user-2", nil, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{meter1Serial}, first.scope.DeviceIDs)
	assert.Equal(t, []string{meter3Serial}, second.scope.DeviceIDs, "one subject's grant must not be served to another")
}

// TestInvalidateDeviceInfoDoesNotForceAuthorizedSetRecompute is a regression
// test for N3: entity.create/entity.update map only to
// atomevents.FamilyTranslation, so they must invalidate the device-info
// cache without touching the authorized-set cache — otherwise a burst of
// such events during a bulk import forces every concurrent reader in the
// domain back through ListAllObjects for no reason a translation-only event
// actually requires. ListAllObjects is mocked .Once(): a second call here
// would fail the mock's own expectations, not just an assertion.
func TestInvalidateDeviceInfoDoesNotForceAuthorizedSetRecompute(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()

	authz := newReadAuthorizer(evaluator, lister, allMeters())
	authz.ttl = time.Hour

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)

	require.NoError(t, authz.invalidateDeviceInfo(context.Background(), testDomain))

	got, err = authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs, "authorized set must still be served from its own cache, untouched by a translation-only invalidation")
}

// TestInvalidateDeviceInfoForcesRetranslationOnNextLoad proves the
// translation cache is actually cleared, not merely left unconsulted: two
// subjects share a device, so the second subject's load reuses whatever
// invalidateDeviceInfo left behind in the shared per-domain translation
// cache.
func TestInvalidateDeviceInfoForcesRetranslationOnNextLoad(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == testSubject
	})).Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Subject == "domain-1_user-2"
	})).Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()

	resolver := allMeters()
	authz := newReadAuthorizer(evaluator, lister, resolver)

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, resolver.calls)

	require.NoError(t, authz.invalidateDeviceInfo(context.Background(), testDomain))

	_, err = authz.resolve(context.Background(), testDomain, "domain-1_user-2", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, resolver.calls, "invalidateDeviceInfo must clear the shared translation cache, not just go unused")
}

// Criterion 4: an Atom event invalidating this domain must take effect well
// within the TTL, not wait for it — that is the whole point of MG-14.
func TestReadAuthorizerInvalidateForcesRecompute(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		Return(policies.PolicyPage{Policies: nil}, nil).Once()

	authz := newReadAuthorizer(evaluator, lister, allMeters())
	authz.ttl = time.Hour // long enough that only Invalidate, never the TTL, can explain the second result

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []string{meter1Serial}, got.scope.DeviceIDs)

	require.NoError(t, authz.Invalidate(context.Background(), testDomain))

	got, err = authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.True(t, got.noAccess, "a revoked grant must not survive an Invalidate for its domain")
}

// Criterion 8: an event for one tenant must not disturb another's cache.
func TestReadAuthorizerInvalidateIsScopedToItsDomain(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)

	otherDomain := "domain-2"
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Domain == testDomain
	})).Return(policies.PolicyPage{Policies: []string{meter1UUID}}, nil).Once()
	lister.EXPECT().ListAllObjects(mock.Anything, mock.MatchedBy(func(pr policies.Policy) bool {
		return pr.Domain == otherDomain
	})).Return(policies.PolicyPage{Policies: []string{meter3UUID}}, nil).Once()

	authz := newReadAuthorizer(evaluator, lister, allMeters())
	authz.ttl = time.Hour

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	_, err = authz.resolve(context.Background(), otherDomain, testSubject, nil, nil)
	require.NoError(t, err)

	require.NoError(t, authz.Invalidate(context.Background(), testDomain))

	// otherDomain's entry must still be cached: a third ListAllObjects call
	// here would fail the mock's expectations (each is set up Once()).
	got, err := authz.resolve(context.Background(), otherDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{meter3Serial}, got.scope.DeviceIDs)
}

// Criterion 6: at-least-once delivery means the same invalidation can arrive
// twice; a cache delete is naturally idempotent, so this must never error.
func TestReadAuthorizerInvalidateIsIdempotent(t *testing.T) {
	authz := customerAuthorizer(t, allMeters(), meter1UUID)

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)

	require.NoError(t, authz.Invalidate(context.Background(), testDomain))
	require.NoError(t, authz.Invalidate(context.Background(), testDomain), "invalidating an already-empty domain must not error")
}

// A regression test for R5's first residual: an invalidation landing while a
// load is in flight must not be served to the caller that triggered it, only
// left out of the cache. Before the fix, doLoad's tolerant epoch check
// stopped the stale grant from being cached but grant() still returned it
// directly from singleflight.Do.
func TestReadAuthorizerRetriesWhenInvalidationLandsMidLoad(t *testing.T) {
	evaluator := policymocks.NewEvaluator(t)
	lister := policymocks.NewService(t)
	evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))

	var authz *readAuthorizer
	first := true
	lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, _ policies.Policy) (policies.PolicyPage, error) {
			if first {
				first = false
				// Simulate a revocation landing while this very load is in
				// flight, before it returns the pre-revocation grant.
				require.NoError(t, authz.Invalidate(ctx, testDomain))
				return policies.PolicyPage{Policies: []string{meter1UUID}}, nil
			}
			// The retry this must trigger sees the post-revocation state.
			return policies.PolicyPage{Policies: nil}, nil
		})

	authz = newReadAuthorizer(evaluator, lister, allMeters())
	authz.ttl = time.Hour

	got, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	assert.True(t, got.noAccess, "a grant that went stale mid-load must not be served, only retried")
	assert.False(t, first, "the retry must have run")
}

// A regression test for R5's second residual: a domain's epoch entry must not
// accumulate forever once nothing depends on it. Safe to prune only once no
// load for the domain is in flight — TestReadAuthorizerRetriesWhenInvalidationLandsMidLoad
// covers the case where one is.
func TestReadAuthorizerInvalidatePrunesEpochOnceNoLoadIsInFlight(t *testing.T) {
	authz := customerAuthorizer(t, allMeters(), meter1UUID)

	_, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil)
	require.NoError(t, err)
	require.NoError(t, authz.Invalidate(context.Background(), testDomain))

	authz.mu.Lock()
	_, tracked := authz.epoch[testDomain]
	authz.mu.Unlock()
	assert.False(t, tracked, "a domain's epoch entry must be forgotten once its cache is clear and nothing is loading")
}

func TestReadAuthorizerInvalidateEmptyDomainIsNoop(t *testing.T) {
	authz := newReadAuthorizer(policymocks.NewEvaluator(t), policymocks.NewService(t), allMeters())
	assert.NoError(t, authz.Invalidate(context.Background(), ""))
}

func TestIntersect(t *testing.T) {
	cases := []struct {
		desc       string
		requested  []string
		authorized map[string]struct{}
		want       []string
		denied     bool
	}{
		{
			desc:       "nothing requested is not a denial",
			requested:  nil,
			authorized: set("a", "b"),
		},
		{
			desc:       "nothing requested against nothing authorized is still not a denial here",
			requested:  nil,
			authorized: set(),
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
			desc:       "only unauthorized ids is a denial",
			requested:  []string{"z", "y"},
			authorized: set("a", "b"),
			denied:     true,
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
			got, denied := intersect(tc.requested, tc.authorized)
			assert.Equal(t, tc.denied, denied)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}

func TestRequestedFilters(t *testing.T) {
	cases := []struct {
		desc           string
		pm             readers.PageMetadata
		wantPublishers []string
		wantDevices    []string
	}{
		{desc: "neither set", pm: readers.PageMetadata{}},
		{desc: "singular publisher only", pm: readers.PageMetadata{Publisher: "a"}, wantPublishers: []string{"a"}},
		{desc: "plural publishers only", pm: readers.PageMetadata{Publishers: []string{"a", "b"}}, wantPublishers: []string{"a", "b"}},
		{desc: "both publisher forms", pm: readers.PageMetadata{Publisher: "c", Publishers: []string{"a", "b"}}, wantPublishers: []string{"a", "b", "c"}},
		{desc: "device ids", pm: readers.PageMetadata{DeviceIDs: []string{meter1Serial}}, wantDevices: []string{meter1Serial}},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.wantPublishers, requestedPublishers(tc.pm))
			assert.Equal(t, tc.wantDevices, requestedDeviceIDs(tc.pm))
		})
	}
}

// A sanity check that resolution stays linear as the authorized set grows. This
// is the materialised path the PRD accepts as the shipping ceiling, so the point
// is to know where that ceiling is, not to defend a latency budget.
func BenchmarkResolveAuthorizedSetSizes(b *testing.B) {
	for _, size := range []int{1, 100, 10000} {
		b.Run(fmt.Sprintf("devices=%d", size), func(b *testing.B) {
			granted := make([]string, 0, size)
			serialsByUUID := make(map[string]string, size)
			for i := 0; i < size; i++ {
				id := fmt.Sprintf("uuid-%d", i)
				granted = append(granted, id)
				serialsByUUID[id] = fmt.Sprintf("serial-%d", i)
			}

			evaluator := policymocks.NewEvaluator(b)
			lister := policymocks.NewService(b)
			evaluator.EXPECT().CheckPolicy(mock.Anything, mock.Anything).Return(errors.New("denied"))
			lister.EXPECT().ListAllObjects(mock.Anything, mock.Anything).Return(policies.PolicyPage{Policies: granted}, nil)
			authz := newReadAuthorizer(evaluator, lister, &fakeResolver{serials: serialsByUUID})

			// Warm the cache: the steady state is a cache hit, since the TTL
			// covers far more than one request.
			if _, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := authz.resolve(context.Background(), testDomain, testSubject, nil, nil); err != nil {
					b.Fatal(err)
				}
			}
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
