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
	grpcDevicesV1 "github.com/absmach/magistrala/api/grpc/devices/v1"
	"github.com/absmach/magistrala/pkg/atom"
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

// errNoDeviceInfoResolver is returned rather than silently skipping the
// translation. A grant set that never reaches its serials produces a filter
// matching no gateway-relayed rows, which reads as a permissions bug instead of
// a wiring one.
var errNoDeviceInfoResolver = errors.New("device info resolver is not configured")

// deviceInfoResolver maps Atom entity UUIDs to the external identifier and
// gateway flag the entities were registered with. The second return value
// names ids that failed to read rather than simply not existing -- see
// atom.Client.EntityDeviceInfo's doc comment -- which resolveDeviceInfo uses
// to avoid caching a transient failure as a confirmed negative (Q4).
type deviceInfoResolver interface {
	EntityDeviceInfo(ctx context.Context, ids []string) (info map[string]atom.DeviceInfo, unreadable map[string]string, err error)
}

// authorizedDevice is one device under both identities it can be known by on a
// message row: its platform UUID when it publishes for itself, and its external
// serial when a gateway relays for it. isGateway is spec §8 A12's
// attributes.is_gateway — see the DeviceScope.SelfPublisherIDs doc comment in
// readers/messages.go for why it gates the R1 fix.
type authorizedDevice struct {
	publisherID string
	serial      string
	isGateway   bool
}

// deviceInfoEntry caches one entity's Atom device-info translation
// (external_id, is_gateway) independently of the authorized-set cache below.
// Keeping it separate is what lets a translation-only event (entity.create,
// entity.update -- atomevents.FamilyTranslation) be handled by dropping just
// this, instead of every subject's authorized-set grant in the domain: a
// provisioning burst of entity.create events would otherwise force every
// concurrent reader in the domain back through ListAllObjects -- the more
// expensive of the two Atom calls a grant load makes -- for no reason a
// translation-only event actually requires (N3).
type deviceInfoEntry struct {
	info      atom.DeviceInfo
	resolved  bool
	expiresAt time.Time
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
	resolver  deviceInfoResolver
	ttl       time.Duration
	now       func() time.Time

	mu    sync.Mutex
	cache map[string]readGrant
	// epoch counts invalidations per domain. grant() captures it before
	// starting a load and re-checks it before caching the result, so a
	// revocation that lands while the load is in flight is not overwritten
	// by that load's now-stale answer. It is also re-checked after the load
	// returns, so the stale answer is not served for this request either —
	// only retried once against the epoch the invalidation left behind.
	epoch map[string]uint64
	// domainLoads counts grant() calls currently loading for each domain, so
	// Invalidate can tell whether forgetting a domain's epoch entry is safe.
	// A load in flight holds a snapshot of epoch from before some
	// invalidations; deleting the entry resets the next lookup to the map's
	// zero value, which that stale snapshot could then coincidentally match
	// and pass as fresh. Pruning only when this is zero rules that out.
	domainLoads map[string]int
	// group collapses concurrent cache misses for the same (domain, subject)
	// into a single load, rather than one Atom round trip per waiting
	// request.
	group singleflight.Group

	// deviceInfoMu guards deviceInfo, kept separate from mu above so a
	// translation-only invalidation (see invalidateDeviceInfo) never
	// contends with the authorized-set cache's epoch/singleflight
	// bookkeeping, which does not need to know this cache exists at all.
	deviceInfoMu sync.Mutex
	// deviceInfo caches per-entity translation, keyed by domain + "\x00" +
	// entity id -- see the deviceInfoEntry doc comment for why this is a
	// separate cache from the one below rather than folded into readGrant.
	deviceInfo map[string]deviceInfoEntry
}

func newReadAuthorizer(evaluator policies.Evaluator, lister policies.Service, resolver deviceInfoResolver) *readAuthorizer {
	return &readAuthorizer{
		evaluator:   evaluator,
		lister:      lister,
		resolver:    resolver,
		ttl:         readAuthzCacheTTL,
		now:         time.Now,
		cache:       make(map[string]readGrant),
		epoch:       make(map[string]uint64),
		domainLoads: make(map[string]int),
		deviceInfo:  make(map[string]deviceInfoEntry),
	}
}

var _ atomevents.Invalidator = (*readAuthorizer)(nil)

// invalidatorFunc adapts a plain function to atomevents.Invalidator, the way
// http.HandlerFunc adapts a function to http.Handler. readAuthorizer needs
// two independent Invalidator values -- one per cache -- registered against
// two different families; a bare method value cannot satisfy an interface on
// its own, so this wraps invalidateDeviceInfo for its registration in
// MakeHandler without needing a second named type.
type invalidatorFunc func(ctx context.Context, key string) error

func (f invalidatorFunc) Invalidate(ctx context.Context, key string) error {
	return f(ctx, key)
}

// Invalidate drops every cached grant for domain (an Atom tenant id, see
// pkg/atom/events.Handler), so the next read for any subject in it
// recomputes the policy lookup half of the grant from Atom instead of
// serving a stale cache entry for up to readAuthzCacheTTL. It is registered
// against atomevents.FamilyAuthorizedSet only -- the UUID-to-external_id
// translation half lives in the separate deviceInfo cache below, dropped by
// invalidateDeviceInfo against FamilyTranslation instead, so that an
// entity.create/entity.update event does not have to pay for recomputing
// every subject's authorized set just to refresh a translation (N3).
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
	// Safe to forget the counter only when no load for this domain is
	// currently in flight (see the domainLoads doc comment) — every cache
	// entry for it was just cleared above, so at that point nothing depends
	// on this domain's epoch value any more. Otherwise every domain ever
	// invalidated keeps one entry forever, unlike the cache entries beside it
	// which at least expire.
	if a.domainLoads[domain] == 0 {
		delete(a.epoch, domain)
	}
	return nil
}

// invalidateDeviceInfo drops every cached translation entry for domain,
// registered against atomevents.FamilyTranslation. Deliberately simpler than
// Invalidate above: this cache carries no epoch -- it is repopulated lazily,
// entry by entry, by resolveDeviceInfo on the next lookup that misses, so
// there is no in-flight-load race to guard against -- a load that started
// before this runs simply overwrites its own now-stale entries once it
// finishes, same as any ordinary TTL expiry would.
func (a *readAuthorizer) invalidateDeviceInfo(_ context.Context, domain string) error {
	if domain == "" {
		return nil
	}

	prefix := domain + "\x00"
	a.deviceInfoMu.Lock()
	defer a.deviceInfoMu.Unlock()
	for key := range a.deviceInfo {
		if strings.HasPrefix(key, prefix) {
			delete(a.deviceInfo, key)
		}
	}
	return nil
}

// resolveDeviceInfo translates ids to their Atom device info, serving
// whatever is already cached for domain and batching only the misses
// through the resolver -- the same shape as EntityDeviceInfo itself, one
// level up. A miss is cached too (resolved: false, see deviceInfoEntry), so
// an id repeatedly requested but never resolvable does not re-hit Atom on
// every load within the TTL.
func (a *readAuthorizer) resolveDeviceInfo(ctx context.Context, domain string, ids []string) (map[string]atom.DeviceInfo, error) {
	out := make(map[string]atom.DeviceInfo, len(ids))
	missing := make([]string, 0, len(ids))

	now := a.now()
	a.deviceInfoMu.Lock()
	for _, id := range ids {
		entry, ok := a.deviceInfo[domain+"\x00"+id]
		if !ok || !now.Before(entry.expiresAt) {
			missing = append(missing, id)
			continue
		}
		if entry.resolved {
			out[id] = entry.info
		}
	}
	a.deviceInfoMu.Unlock()

	if len(missing) == 0 {
		return out, nil
	}

	resolved, unreadable, err := a.resolver.EntityDeviceInfo(ctx, missing)
	if err != nil {
		return nil, err
	}

	expiresAt := a.now().Add(a.ttl)
	a.deviceInfoMu.Lock()
	for _, id := range missing {
		if _, transient := unreadable[id]; transient {
			// Do not cache a transient per-entity failure as though it were
			// a confirmed, stable "no info" -- that is exactly the
			// conflation entitiesExist was fixed to refuse (R2), and caching
			// it here would lock this device out of R1's self-publish trust
			// for a full TTL after a single blip (Q4). Leave it uncached so
			// the next load retries it instead of trusting a stale
			// "could not tell". This load still treats it as unresolved --
			// the safe default -- for this one request; only the caching is
			// skipped.
			continue
		}
		info, ok := resolved[id]
		a.deviceInfo[domain+"\x00"+id] = deviceInfoEntry{info: info, resolved: ok, expiresAt: expiresAt}
		if ok {
			out[id] = info
		}
	}
	a.deviceInfoMu.Unlock()

	return out, nil
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

// selfPublisherIDs is the subset of publisherIDs not flagged as a gateway —
// see the DeviceScope.SelfPublisherIDs doc comment in readers/messages.go.
func selfPublisherIDs(devices []authorizedDevice) map[string]struct{} {
	out := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		if d.publisherID != "" && !d.isGateway {
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
	if ok && a.now().Before(grant.expiresAt) {
		a.mu.Unlock()
		return grant, nil
	}
	a.domainLoads[domain]++
	a.mu.Unlock()
	defer a.releaseDomainLoad(domain)

	epoch := a.currentEpoch(domain)
	grant, moved, err := a.doLoad(ctx, domain, subject, key, epoch)
	if err != nil {
		return readGrant{}, err
	}
	if !moved {
		return grant, nil
	}

	// An invalidation landed for this domain while the load above was in
	// flight. It was correctly left out of the cache (see doLoad), but
	// singleflight still hands its now-stale result back to every caller who
	// joined it regardless — serving it here would leak one stale read per
	// revocation despite the cache itself being clean. Retry once against
	// the epoch the invalidation left behind rather than serve it.
	epoch = a.currentEpoch(domain)
	grant, _, err = a.doLoad(ctx, domain, subject, key, epoch)
	if err != nil {
		return readGrant{}, err
	}
	return grant, nil
}

// doLoad runs (or joins) a singleflight load for key and caches the result,
// unless domain's epoch has moved past epoch by the time it finishes — which
// means an invalidation landed while it was in flight, so the result reflects
// pre-invalidation state and moved is reported true.
func (a *readAuthorizer) doLoad(ctx context.Context, domain, subject, key string, epoch uint64) (grant readGrant, moved bool, err error) {
	v, err, _ := a.group.Do(key, func() (any, error) {
		g, err := a.load(ctx, domain, subject)
		if err != nil {
			return readGrant{}, err
		}
		g.expiresAt = a.now().Add(a.ttl)

		a.mu.Lock()
		defer a.mu.Unlock()
		// Only cache the result if no invalidation landed for this domain
		// while the load was in flight — caching it unconditionally could
		// overwrite a revocation that arrived after the load read from
		// Atom but before it finished, resetting the revoked grant's TTL
		// instead of leaving it cleared.
		if a.epoch[domain] == epoch {
			a.cache[key] = g
		}
		return g, nil
	})
	if err != nil {
		return readGrant{}, false, err
	}
	return v.(readGrant), a.currentEpoch(domain) != epoch, nil
}

func (a *readAuthorizer) currentEpoch(domain string) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.epoch[domain]
}

func (a *readAuthorizer) releaseDomainLoad(domain string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.domainLoads[domain]--
	if a.domainLoads[domain] <= 0 {
		delete(a.domainLoads, domain)
	}
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
		return readGrant{}, errNoDeviceInfoResolver
	}
	info, err := a.resolveDeviceInfo(ctx, domain, page.Policies)
	if err != nil {
		return readGrant{}, err
	}

	devices := make([]authorizedDevice, 0, len(page.Policies))
	for _, id := range page.Policies {
		d, resolved := info[id]
		// An id the resolver did not return at all — as opposed to one it
		// returned with IsGateway false — could not be confirmed either way.
		// Treated as a gateway here, the safe direction: it only costs this
		// device the R1 self-publish trust for this cache window, whereas
		// trusting it wrongly would let publisher-only matching readmit
		// whatever it relays for someone else (see DeviceScope.SelfPublisherIDs).
		devices = append(devices, authorizedDevice{publisherID: id, serial: d.ExternalID, isGateway: !resolved || d.IsGateway})
	}
	return newReadGrant(devices), nil
}

func newReadGrant(devices []authorizedDevice) readGrant {
	publishers := publisherIDs(devices)
	selfPublishers := selfPublisherIDs(devices)
	serialSet := serials(devices)
	return readGrant{
		publisherIDs: publishers,
		serials:      serialSet,
		scope: &readers.DeviceScope{
			PublisherIDs:     sortedKeys(publishers),
			SelfPublisherIDs: sortedKeys(selfPublishers),
			DeviceIDs:        sortedKeys(serialSet),
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
func authzDeviceView(ctx context.Context, chanID, domain, token, key string, authn smqauthn.Authentication, clients grpcDevicesV1.DevicesServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer, requestedPublishers, requestedDeviceIDs []string) (scope *readers.DeviceScope, noAccess bool, err error) {
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
