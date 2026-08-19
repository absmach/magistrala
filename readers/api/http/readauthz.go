// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	atomevents "github.com/absmach/magistrala/pkg/atom/events"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/readers"
	"golang.org/x/sync/singleflight"
)

// readAuthzCacheTTL is kept short since it backs a security control, not just a
// performance cache. It is also the revocation SLA: until MG-14 makes
// invalidation event-driven, a revoked grant keeps working for at most this long.
const readAuthzCacheTTL = 30 * time.Second

// errNoExternalIDResolver is returned rather than silently skipping the
// translation. A grant set that never reaches its serials produces a filter
// matching no gateway-relayed rows, which reads as a permissions bug instead of
// a wiring one.
var errNoExternalIDResolver = errors.New("device external id resolver is not configured")

// externalIDResolver maps Atom entity UUIDs to the external identifiers the
// entities were registered with.
type externalIDResolver interface {
	EntityExternalIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// authorizedDevice is one device under both identities it can be known by on a
// message row: its platform UUID when it publishes for itself, and its external
// serial when a gateway relays for it.
type authorizedDevice struct {
	publisherID string
	serial      string
}

// readGrant is what a (domain, subject) pair may read: either everything, or a
// bounded set of devices.
//
// The projections are built once, when the grant is loaded, and then shared by
// every request served from the cache — materialising them per request is what
// makes a large fleet expensive. Nothing downstream writes to them: the scope
// travels into PageMetadata by pointer and the repositories only read it.
type readGrant struct {
	unrestricted bool
	publisherIDs map[string]struct{}
	serials      map[string]struct{}
	scope        *readers.DeviceScope
	expiresAt    time.Time
}

// readAuthorizer resolves which devices a caller may read on a channel, and
// turns that into the scope and filters the query is allowed to run with.
type readAuthorizer struct {
	evaluator policies.Evaluator
	lister    policies.Service
	resolver  externalIDResolver
	ttl       time.Duration
	now       func() time.Time

	mu    sync.Mutex
	cache map[string]readGrant
	// epoch counts invalidations per domain. grant() captures it before
	// starting a load and re-checks it before caching the result, so a
	// revocation that lands while the load is in flight is not overwritten
	// by that load's now-stale answer.
	epoch map[string]uint64
	// group collapses concurrent cache misses for the same (domain, subject)
	// into a single load, rather than one Atom round trip per waiting
	// request.
	group singleflight.Group
}

func newReadAuthorizer(evaluator policies.Evaluator, lister policies.Service, resolver externalIDResolver) *readAuthorizer {
	return &readAuthorizer{
		evaluator: evaluator,
		lister:    lister,
		resolver:  resolver,
		ttl:       readAuthzCacheTTL,
		now:       time.Now,
		cache:     make(map[string]readGrant),
		epoch:     make(map[string]uint64),
	}
}

var _ atomevents.Invalidator = (*readAuthorizer)(nil)

// Invalidate drops every cached grant for domain (an Atom tenant id, see
// pkg/atom/events.Handler), so the next read for any subject in it
// recomputes both halves of the grant -- the policy lookup and the
// UUID-to-external_id translation -- from Atom instead of serving a stale
// cache entry for up to readAuthzCacheTTL.
//
// It clears every subject in the domain rather than one specific subject on
// purpose: some of the Atom events that trigger this (direct_policy events
// in particular) carry no subject in their payload, only the changed
// policy's own row id, and MG-14's invalidate-only design forbids inferring
// one by inspecting payload content further. Domain-wide is the coarsest
// granularity this cache supports and the only one that stays correct
// regardless of which event triggered it. It is also idempotent: deleting
// keys that are already absent -- an empty domain, a duplicate event -- is
// a no-op, which is what makes at-least-once delivery safe to wire in
// directly.
func (a *readAuthorizer) Invalidate(_ context.Context, domain string) error {
	if domain == "" {
		return nil
	}

	prefix := domain + "\x00"
	a.mu.Lock()
	defer a.mu.Unlock()
	for key := range a.cache {
		if strings.HasPrefix(key, prefix) {
			delete(a.cache, key)
		}
	}
	// Bumped even when nothing was cached to delete: this is what stops a
	// load already in flight for this domain (started before the
	// invalidation, finishing after it) from writing its now-stale result
	// back into the cache once grant() re-checks it — without this, a
	// revoked grant could keep serving for another full TTL despite the
	// cache having just been cleared.
	a.epoch[domain]++
	return nil
}

// authzScope is what the caller's request is allowed to become.
type authzScope struct {
	// noAccess means the caller may read nothing here. Treat it as zero rows,
	// never as an unfiltered read.
	noAccess bool
	// scope bounds the query to the caller's devices. nil means unbounded, which
	// only ever happens for a caller holding the tenant-wide admin capability.
	scope *readers.DeviceScope
	// publishers and deviceIDs are the caller's own filters, narrowed to what
	// they are entitled to. They stay plain column filters — the OR between the
	// two identities lives inside scope, not here — so asking for a publisher
	// still means "the publisher column", exactly as it did before.
	publishers []string
	deviceIDs  []string
}

// resolve intersects what the caller asked for with what they are entitled to.
func (a *readAuthorizer) resolve(ctx context.Context, domain, subject string, requestedPublishers, requestedDeviceIDs []string) (authzScope, error) {
	grant, err := a.grant(ctx, domain, subject)
	if err != nil {
		return authzScope{}, err
	}
	if grant.unrestricted {
		return authzScope{publishers: requestedPublishers, deviceIDs: requestedDeviceIDs}, nil
	}

	publishers, publisherDenied := intersect(requestedPublishers, grant.publisherIDs)
	deviceIDs, deviceDenied := intersect(requestedDeviceIDs, grant.serials)
	if publisherDenied || deviceDenied {
		return authzScope{noAccess: true}, nil
	}

	// A scope admitting no device at all is the sharp edge of this control: it
	// has to mean zero rows. The query would honour that too — DeviceScope binds
	// empty arrays rather than dropping the condition — but short-circuiting here
	// means it never depends on that.
	if grant.scope.Empty() {
		return authzScope{noAccess: true}, nil
	}

	// Only the caller's explicit filters are reported back; requesting nothing
	// leaves them unset so the scope alone bounds the query.
	if len(requestedPublishers) == 0 {
		publishers = nil
	}
	if len(requestedDeviceIDs) == 0 {
		deviceIDs = nil
	}

	return authzScope{scope: grant.scope, publishers: publishers, deviceIDs: deviceIDs}, nil
}

// intersect narrows a requested set to its authorized part. Ids outside the
// authorized set are dropped silently rather than rejected, so the response
// never reveals which ids exist. A request that survives with nothing left is
// reported as denied — an empty result here can never be allowed to read as
// "no filter".
func intersect(requested []string, authorized map[string]struct{}) (ids []string, denied bool) {
	if len(requested) == 0 {
		return nil, false
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
		return nil, true
	}
	return out, false
}

func publisherIDs(devices []authorizedDevice) map[string]struct{} {
	out := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		if d.publisherID != "" {
			out[d.publisherID] = struct{}{}
		}
	}
	return out
}

// serials skips devices with no external id. Such a device is only ever
// reachable through its publisher identity, and an empty serial in the scope
// would match every device-less row on the channel.
func serials(devices []authorizedDevice) map[string]struct{} {
	out := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		if d.serial != "" {
			out[d.serial] = struct{}{}
		}
	}
	return out
}

// sortedKeys keeps the bound arrays stable across requests, so a query plan and
// a test assertion do not depend on Go's map iteration order.
func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (a *readAuthorizer) grant(ctx context.Context, domain, subject string) (readGrant, error) {
	key := domain + "\x00" + subject

	a.mu.Lock()
	grant, ok := a.cache[key]
	epoch := a.epoch[domain]
	a.mu.Unlock()
	if ok && a.now().Before(grant.expiresAt) {
		return grant, nil
	}

	v, err, _ := a.group.Do(key, func() (any, error) {
		grant, err := a.load(ctx, domain, subject)
		if err != nil {
			return readGrant{}, err
		}
		grant.expiresAt = a.now().Add(a.ttl)

		a.mu.Lock()
		defer a.mu.Unlock()
		// Only cache the result if no invalidation landed for this domain
		// while the load was in flight — caching it unconditionally could
		// overwrite a revocation that arrived after the load read from
		// Atom but before it finished, resetting the revoked grant's TTL
		// instead of leaving it cleared.
		if a.epoch[domain] == epoch {
			a.cache[key] = grant
		}
		return grant, nil
	})
	if err != nil {
		return readGrant{}, err
	}
	return v.(readGrant), nil
}

func (a *readAuthorizer) load(ctx context.Context, domain, subject string) (readGrant, error) {
	if a.isDomainAdmin(ctx, domain, subject) {
		return readGrant{unrestricted: true}, nil
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
		return readGrant{}, err
	}
	if len(page.Policies) == 0 {
		return newReadGrant(nil), nil
	}

	// Policy lookups deal in Atom UUIDs; message rows carry the device's serial
	// verbatim, because the publish path performs no lookup. Without this step
	// the device half of the scope matches nothing at all.
	if a.resolver == nil {
		return readGrant{}, errNoExternalIDResolver
	}
	external, err := a.resolver.EntityExternalIDs(ctx, page.Policies)
	if err != nil {
		return readGrant{}, err
	}

	devices := make([]authorizedDevice, 0, len(page.Policies))
	for _, id := range page.Policies {
		devices = append(devices, authorizedDevice{publisherID: id, serial: external[id]})
	}
	return newReadGrant(devices), nil
}

func newReadGrant(devices []authorizedDevice) readGrant {
	publishers := publisherIDs(devices)
	serialSet := serials(devices)
	return readGrant{
		publisherIDs: publishers,
		serials:      serialSet,
		scope: &readers.DeviceScope{
			PublisherIDs: sortedKeys(publishers),
			DeviceIDs:    sortedKeys(serialSet),
		},
	}
}

// isDomainAdmin checks a tenant-wide admin capability rather than inferring it from a role name.
// CheckPolicy reports denial and request failure alike as a non-nil error, so both count as "not
// admin" here; a real Atom outage still surfaces as an error from the per-device lookup below.
func (a *readAuthorizer) isDomainAdmin(ctx context.Context, domain, subject string) bool {
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

// authzDeviceView resolves whether the caller may run one of the MG-15
// observed-device aggregation queries, and what to narrow it with.
//
// It mirrors authnAuthz: authenticate, then a channel-level subscribe check,
// then — for a non-admin domain user — per-device scope resolution. The
// requestedPublishers/requestedDeviceIDs passed here differ by direction: the
// device->gateways endpoint names the device the caller must be authorized
// for, so it is passed as a one-element requestedDeviceIDs and must survive
// the same intersection a filter would. The gateway->devices endpoint names
// the gateway only as the source of the roster — a single gateway can relay
// for devices belonging to more than one customer — so it passes neither
// identity and relies on the resolved scope alone to bound the query: the
// caller must hold at least one device grant (otherwise noAccess, reported as
// an empty page, never an error), but never needs to hold a grant on the
// gateway client itself.
func authzDeviceView(ctx context.Context, chanID, domain, token, key string, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer, requestedPublishers, requestedDeviceIDs []string) (scope *readers.DeviceScope, noAccess bool, err error) {
	clientID, clientType, superAdmin, err := authenticate(ctx, token, key, domain, authn, clients)
	if err != nil {
		return nil, false, err
	}
	if err := authorize(ctx, clientID, clientType, chanID, domain, channels); err != nil {
		return nil, false, err
	}

	if superAdmin {
		return nil, false, nil
	}

	// Per-device grants only exist for domain users, not for clients
	// authenticating with a secret key, so a client cannot be narrowed by
	// DeviceScope. The roster is therefore returned unscoped: the client is
	// still bound by the channel grant just checked, and so sees the full set
	// of distinct devices (or gateways) on that channel — the same
	// channel-level visibility its ReadMessages calls already have. This is
	// deliberate: secret-key clients predate per-device grants, and narrowing
	// them to a scope they cannot meaningfully be granted is not possible, so
	// the honest behaviour is channel-scoped and full, mirroring ReadMessages
	// (MG-08).
	if clientType != policies.UserType {
		return nil, false, nil
	}

	resolved, err := readAuthz.resolve(ctx, domain, clientID, requestedPublishers, requestedDeviceIDs)
	if err != nil {
		return nil, false, err
	}
	if resolved.noAccess {
		return nil, true, nil
	}

	return resolved.scope, false, nil
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

// requestedDeviceIDs collects the device serials the caller asked to filter by.
// There is no singular form: serials are format-unconstrained (MG-09), so the
// plural parameter is repeated rather than separated.
func requestedDeviceIDs(pm readers.PageMetadata) []string {
	return pm.DeviceIDs
}
