// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

// Magistrala dropped several uniqueness constraints that Atom still enforces
// (tenants.name, object device/group name per tenant, tenant/entity/resource
// alias). buildDedup precomputes collision-free names and aliases so the load
// never trips those indexes. Renaming is deterministic: rows are processed in a
// stable order (created_at, id) and the loser of a collision keeps its original
// value plus an id-derived suffix, so re-runs produce identical output.
func (m *migrator) buildDedup(ctx context.Context, rep *report) error {
	doms, err := readDomains(ctx, m.domainsDB)
	if err != nil {
		return err
	}
	sort.SliceStable(doms, func(i, j int) bool {
		return earlier(doms[i].CreatedAt, doms[i].ID, doms[j].CreatedAt, doms[j].ID)
	})
	domSet := map[string]bool{}
	for _, d := range doms {
		domSet[d.ID] = true
	}

	tNames := newAllocator(false)
	tAlias := newAllocator(true)
	for _, d := range doms {
		base := strings.TrimSpace(nsToStr(d.Name))
		if base == "" {
			base = d.ID
		}
		final := tNames.take("", base, d.ID)
		if final != base {
			rep.warn("tenant %s name %q -> %q (tenants.name is UNIQUE)", d.ID, base, final)
			rep.count("renamed.tenants", 1)
		}
		m.tenantName[d.ID] = final
		m.tenantAlias[d.ID] = m.dedupAlias(tAlias, "", d.Route, "tenant "+d.ID, rep)
	}

	clients, err := readClients(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	sort.SliceStable(clients, func(i, j int) bool {
		return earlier(clients[i].CreatedAt, clients[i].ID, clients[j].CreatedAt, clients[j].ID)
	})
	dNames := newAllocator(false)
	cAlias := newAllocator(true)
	for _, c := range clients {
		if !domSet[c.DomainID] {
			continue
		}
		base := firstNonEmpty(strings.TrimSpace(nsToStr(c.Name)), c.ID)
		final := dNames.take(c.DomainID, base, c.ID)
		if final != base {
			rep.warn("device %s name %q -> %q (entities(name, tenant_id) is UNIQUE)", c.ID, base, final)
			rep.count("renamed.devices", 1)
		}
		m.deviceName[c.ID] = final
		m.clientAlias[c.ID] = m.dedupAlias(cAlias, c.DomainID, c.Identity, "client "+c.ID, rep)
	}

	chans, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	sort.SliceStable(chans, func(i, j int) bool {
		return earlier(chans[i].CreatedAt, chans[i].ID, chans[j].CreatedAt, chans[j].ID)
	})
	chAlias := newAllocator(true)
	for _, ch := range chans {
		if !domSet[ch.DomainID] {
			continue
		}
		m.channelAlias[ch.ID] = m.dedupAlias(chAlias, ch.DomainID, ch.Route, "channel "+ch.ID, rep)
	}

	grps, err := readGroups(ctx, m.groupsDB)
	if err != nil {
		return err
	}
	sort.SliceStable(grps, func(i, j int) bool {
		return earlier(grps[i].CreatedAt, grps[i].ID, grps[j].CreatedAt, grps[j].ID)
	})
	gNames := newAllocator(false)
	for _, g := range grps {
		if !domSet[g.DomainID] {
			continue
		}
		base := firstNonEmpty(strings.TrimSpace(g.Name), g.ID)
		final := gNames.take(g.DomainID, base, g.ID)
		if final != base {
			rep.warn("group %s name %q -> %q (object_groups(name, tenant_id) is UNIQUE)", g.ID, base, final)
			rep.count("renamed.groups", 1)
		}
		m.groupName[g.ID] = final
	}

	if err := m.dedupUsers(ctx, rep); err != nil {
		return err
	}
	return nil
}

// dedupUsers computes collision-free entity names for users. Magistrala users
// are platform-global, so they land in Atom's tenant-less namespace
// (entities.tenant_id IS NULL). Migration 006 collapsed NULL tenant_id to a
// sentinel UUID, making that whole namespace a single unique scope shared with
// the bootstrap system entities (admin, mg-service). Pre-seed those existing
// tenant-less names so a colliding migrated user is renamed rather than
// tripping idx_entities_name_tenant. Names already owned by a user we are about
// to migrate (same id) are not reserved: those re-run idempotently via
// ON CONFLICT (id) and must keep their original name.
func (m *migrator) dedupUsers(ctx context.Context, rep *report) error {
	users, err := readUsers(ctx, m.usersDB)
	if err != nil {
		return err
	}
	srcID := map[string]bool{}
	for _, u := range users {
		srcID[u.ID] = true
	}

	uNames := newAllocator(false)
	rows, err := m.atom.QueryxContext(ctx,
		`SELECT id, name FROM entities WHERE tenant_id IS NULL`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		if srcID[id] {
			continue // re-run of this user; ON CONFLICT (id) keeps its name
		}
		uNames.take("", name, id) // reserve the system/foreign name
	}
	rows.Close()

	sort.SliceStable(users, func(i, j int) bool {
		return earlier(users[i].CreatedAt, users[i].ID, users[j].CreatedAt, users[j].ID)
	})
	for _, u := range users {
		base := firstNonEmpty(u.Username.String, u.Email.String, u.ID)
		final := uNames.take("", base, u.ID)
		if final != base {
			rep.warn("user %s name %q -> %q (entities(name, tenant_id) is UNIQUE)", u.ID, base, final)
			rep.count("renamed.users", 1)
		}
		m.userName[u.ID] = final
	}
	return nil
}

// dedupAlias normalizes a candidate alias and makes it unique within scope.
// Invalid slugs are dropped (as before); a valid alias that collides keeps its
// value plus an id-derived suffix instead of blocking the load. Returns "" when
// no alias should be set (caller writes NULL).
func (m *migrator) dedupAlias(a *allocator, scope string, raw sql.NullString, label string, rep *report) string {
	if !raw.Valid || raw.String == "" {
		return ""
	}
	norm, ok := normalizeAlias(raw.String)
	if !ok {
		rep.warn("alias dropped for %s: %q not a valid slug", label, raw.String)
		rep.skip("alias_dropped")
		return ""
	}
	final := a.take(scope, norm, idSuffix(raw))
	if final != norm {
		rep.warn("alias for %s %q -> %q (alias is UNIQUE within tenant)", label, norm, final)
		rep.count("renamed.aliases", 1)
	}
	return final
}

// idSuffix derives a stable, slug-safe suffix source. raw aliases carry no id,
// so dedupAlias passes the raw alias itself; collisions then disambiguate on a
// short hash of it, which is deterministic per source value.
func idSuffix(raw sql.NullString) string {
	return shortHash(raw.String)
}

// allocator hands out unique strings within a namespace. caseFold true compares
// case-insensitively (aliases); names compare exactly.
type allocator struct {
	used     map[string]map[string]bool
	caseFold bool
}

func newAllocator(caseFold bool) *allocator {
	return &allocator{used: map[string]map[string]bool{}, caseFold: caseFold}
}

func (a *allocator) key(s string) string {
	if a.caseFold {
		return strings.ToLower(s)
	}
	return s
}

// take returns base if free in scope, else base with an id-derived suffix. Long
// alias values are trimmed so the result stays within the 63-char slug limit.
func (a *allocator) take(scope, base, id string) string {
	bucket := a.used[scope]
	if bucket == nil {
		bucket = map[string]bool{}
		a.used[scope] = bucket
	}
	if !bucket[a.key(base)] {
		bucket[a.key(base)] = true
		return base
	}
	for _, suf := range []string{short(id), id} {
		cand := withSuffix(base, suf, a.caseFold)
		if !bucket[a.key(cand)] {
			bucket[a.key(cand)] = true
			return cand
		}
	}
	// Pathological fallback: keep extending until unique.
	cand := withSuffix(base, id, a.caseFold)
	for bucket[a.key(cand)] {
		cand += "x"
	}
	bucket[a.key(cand)] = true
	return cand
}

// withSuffix appends "-suf"; for aliases it caps the total at the 63-char slug
// limit by trimming the base.
func withSuffix(base, suf string, alias bool) string {
	sep := "-"
	if !alias {
		sep = " ("
	}
	if alias {
		if max := 63 - len(suf) - 1; len(base) > max && max > 0 {
			base = strings.TrimRight(base[:max], "-")
		}
		return base + sep + suf
	}
	return base + sep + suf + ")"
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// shortHash is a small deterministic hex tag (FNV-1a, 8 hex chars).
func shortHash(s string) string {
	const (
		offset = 2166136261
		prime  = 16777619
	)
	h := uint32(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	const hexd = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = hexd[h&0xf]
		h >>= 4
	}
	return string(out)
}

// earlier orders rows by (created_at, id): the oldest row wins a collision and
// keeps its original name/alias. Rows with an unknown created_at sort last; id
// breaks ties so the order is total and stable across runs.
func earlier(ta sql.NullTime, ida string, tb sql.NullTime, idb string) bool {
	switch {
	case ta.Valid && tb.Valid && !ta.Time.Equal(tb.Time):
		return ta.Time.Before(tb.Time)
	case ta.Valid != tb.Valid:
		return ta.Valid
	default:
		return ida < idb
	}
}

// aliasOrNil converts a computed alias ("" = none) into a SQL argument.
func aliasOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}
