// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"strings"
)

// preflight runs read-only data-quality checks before any write. Blocking issues
// (rep.block) abort an --apply run; warnings (rep.warn) are advisory. See PLAN §6.
func (m *migrator) preflight(ctx context.Context, rep *report) error {
	checks := []func(context.Context, *report) error{
		m.pfEmails,
		m.pfHumanNames,
		m.pfTenantNames,
		m.pfTenantAlias,
		m.pfEntityResourceAlias,
		m.pfDeviceNames,
		m.pfGroupNames,
		m.pfOrphans,
	}
	for _, c := range checks {
		if err := c(ctx, rep); err != nil {
			return err
		}
	}
	return nil
}

// dupGroups returns the keys that appear more than once (with their count).
func dupGroups(keyOf func() []string) map[string]int {
	seen := map[string]int{}
	for _, k := range keyOf() {
		if k != "" {
			seen[k]++
		}
	}
	for k, n := range seen {
		if n < 2 {
			delete(seen, k)
		}
	}
	return seen
}

// pfEmails: Atom entity_emails.email is globally UNIQUE. Magistrala enforces this
// in the users table, but a dump merged across instances can break it.
func (m *migrator) pfEmails(ctx context.Context, rep *report) error {
	users, err := readUsers(ctx, m.usersDB)
	if err != nil {
		return err
	}
	dups := dupGroups(func() []string {
		out := make([]string, 0, len(users))
		for _, u := range users {
			if u.Email.Valid {
				out = append(out, strings.ToLower(strings.TrimSpace(u.Email.String)))
			}
		}
		return out
	})
	for email, n := range dups {
		rep.blockf("email %q used by %d users (entity_emails.email is UNIQUE)", email, n)
	}
	return nil
}

// pfHumanNames: entities(name, tenant_id) is UNIQUE; humans have tenant_id NULL so
// they share one global name namespace. name = first of username/email/id.
func (m *migrator) pfHumanNames(ctx context.Context, rep *report) error {
	users, err := readUsers(ctx, m.usersDB)
	if err != nil {
		return err
	}
	dups := dupGroups(func() []string {
		out := make([]string, 0, len(users))
		for _, u := range users {
			out = append(out, firstNonEmpty(u.Username.String, u.Email.String, u.ID))
		}
		return out
	})
	for name, n := range dups {
		// entities(name, tenant_id) is NULLS DISTINCT and humans have tenant_id
		// NULL, so duplicate human names do NOT violate the index — advisory only.
		rep.warnf("human entity name %q used by %d users (allowed; tenant NULL is NULLS DISTINCT)", name, n)
	}
	return nil
}

// pfTenantNames: tenants.name is UNIQUE and NOT NULL. Magistrala workspace.name is
// nullable and non-unique, so empty or duplicate names break the load.
func (m *migrator) pfTenantNames(ctx context.Context, rep *report) error {
	doms, err := readWorkspaces(ctx, m.workspacesDB)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(doms))
	for _, d := range doms {
		if !d.Name.Valid || strings.TrimSpace(d.Name.String) == "" {
			rep.warnf("workspace %s has empty name -> will use its id (tenants.name is NOT NULL UNIQUE)", d.ID)
			continue
		}
		names = append(names, d.Name.String)
	}
	for name, n := range dupGroups(func() []string { return names }) {
		rep.warnf("workspace name %q used by %d workspaces -> duplicates auto-renamed (tenants.name is UNIQUE)", name, n)
	}
	return nil
}

// pfGroupNames: object_groups(name, tenant_id) is UNIQUE. Magistrala dropped the
// groups (workspace_id, name) constraint, so same-workspace dups are possible.
func (m *migrator) pfGroupNames(ctx context.Context, rep *report) error {
	grps, err := readGroups(ctx, m.groupsDB)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(grps))
	for _, g := range grps {
		keys = append(keys, g.WorkspaceID+"|"+g.Name)
	}
	for k, n := range dupGroups(func() []string { return keys }) {
		rep.warnf("group name collision (%s) across %d groups in one tenant -> duplicates auto-renamed", k, n)
	}
	return nil
}

// pfTenantAlias: workspace.route -> tenants.alias. Globally unique, case-folded,
// slug-shaped, not UUID-shaped. Invalid shape => alias dropped (warn). Case-fold
// collision among otherwise-valid aliases => block.
func (m *migrator) pfTenantAlias(ctx context.Context, rep *report) error {
	doms, err := readWorkspaces(ctx, m.workspacesDB)
	if err != nil {
		return err
	}
	valid := []string{}
	for _, d := range doms {
		if !d.Route.Valid || d.Route.String == "" {
			continue
		}
		a, ok := normalizeAlias(d.Route.String)
		if !ok {
			rep.warnf("tenant %s alias %q invalid slug -> dropped", d.ID, d.Route.String)
			continue
		}
		valid = append(valid, a)
	}
	for a, n := range dupGroups(func() []string { return valid }) {
		rep.warnf("tenant alias %q collides case-insensitively across %d workspaces -> duplicates auto-suffixed", a, n)
	}
	return nil
}

// pfEntityResourceAlias: client.identity / channel.route are unique per tenant
// (case-folded). Collision within a workspace => block (would violate Atom's unique
// index mid-apply). Invalid shape => warn (dropped).
func (m *migrator) pfEntityResourceAlias(ctx context.Context, rep *report) error {
	devices, err := readDevices(ctx, m.devicesDB)
	if err != nil {
		return err
	}
	chans, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	// key = workspace|alias
	cKeys := []string{}
	for _, c := range devices {
		if !c.Identity.Valid || c.Identity.String == "" {
			continue
		}
		a, ok := normalizeAlias(c.Identity.String)
		if !ok {
			rep.warnf("device %s alias %q invalid slug -> dropped", c.ID, c.Identity.String)
			continue
		}
		cKeys = append(cKeys, c.WorkspaceID+"|"+a)
	}
	for k, n := range dupGroups(func() []string { return cKeys }) {
		rep.warnf("device alias collision (%s) across %d devices in one tenant -> duplicates auto-suffixed", k, n)
	}
	rKeys := []string{}
	for _, ch := range chans {
		if !ch.Route.Valid || ch.Route.String == "" {
			continue
		}
		a, ok := normalizeAlias(ch.Route.String)
		if !ok {
			rep.warnf("channel %s alias %q invalid slug -> dropped", ch.ID, ch.Route.String)
			continue
		}
		rKeys = append(rKeys, ch.WorkspaceID+"|"+a)
	}
	for k, n := range dupGroups(func() []string { return rKeys }) {
		rep.warnf("channel alias collision (%s) across %d channels in one tenant -> duplicates auto-suffixed", k, n)
	}
	return nil
}

// pfDeviceNames: device entities are unique on (name, tenant_id). Magistrala
// dropped the (workspace_id, name) unique constraint, so same-workspace name dups are
// possible and would break the Atom insert.
func (m *migrator) pfDeviceNames(ctx context.Context, rep *report) error {
	devices, err := readDevices(ctx, m.devicesDB)
	if err != nil {
		return err
	}
	keys := []string{}
	for _, c := range devices {
		keys = append(keys, c.WorkspaceID+"|"+firstNonEmpty(c.Name.String, c.ID))
	}
	for k, n := range dupGroups(func() []string { return keys }) {
		rep.warnf("device name collision (%s) across %d devices in one tenant -> duplicates auto-renamed", k, n)
	}
	return nil
}

// pfOrphans: devices/channels/groups whose workspace_id has no surviving workspace are
// skipped during load. Advisory only.
func (m *migrator) pfOrphans(ctx context.Context, rep *report) error {
	doms, err := readWorkspaces(ctx, m.workspacesDB)
	if err != nil {
		return err
	}
	domSet := map[string]bool{}
	for _, d := range doms {
		domSet[d.ID] = true
	}
	count := func(get func() []string, label string) {
		n := 0
		for _, id := range get() {
			if !domSet[id] {
				n++
			}
		}
		if n > 0 {
			rep.warnf("%d %s reference a missing workspace -> will be skipped", n, label)
		}
	}
	devices, err := readDevices(ctx, m.devicesDB)
	if err != nil {
		return err
	}
	count(func() []string {
		out := make([]string, len(devices))
		for i, c := range devices {
			out[i] = c.WorkspaceID
		}
		return out
	}, "devices")
	chans, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	count(func() []string {
		out := make([]string, len(chans))
		for i, c := range chans {
			out[i] = c.WorkspaceID
		}
		return out
	}, "channels")
	grps, err := readGroups(ctx, m.groupsDB)
	if err != nil {
		return err
	}
	count(func() []string {
		out := make([]string, len(grps))
		for i, g := range grps {
			out[i] = g.WorkspaceID
		}
		return out
	}, "groups")
	rules, err := readRules(ctx, m.reDB)
	if err != nil {
		return err
	}
	count(func() []string {
		out := make([]string, len(rules))
		for i, r := range rules {
			out[i] = r.WorkspaceID
		}
		return out
	}, "rules")
	reports, err := readReports(ctx, m.reportsDB)
	if err != nil {
		return err
	}
	count(func() []string {
		out := make([]string, len(reports))
		for i, r := range reports {
			out[i] = r.WorkspaceID
		}
		return out
	}, "reports")
	return nil
}

// preflightGate aborts an --apply run when blocking issues exist.
func (m *migrator) preflightGate(rep *report) error {
	if m.apply && rep.HasBlocking() {
		return fmt.Errorf("preflight found %d blocking issue(s); aborting apply (see report)", len(rep.Blocking))
	}
	return nil
}
