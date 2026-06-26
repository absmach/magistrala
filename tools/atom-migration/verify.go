// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Verify reconciles a completed migration: every source row that should have
// migrated must exist in Atom, and a sample of reconstructed authz edges
// (device→channel publish/subscribe) must be present. Read-only.
func (m *migrator) Verify(ctx context.Context, rep *report) error {
	domSet, err := m.domainSet(ctx)
	if err != nil {
		return err
	}

	atomTenants, err := idSet(ctx, m.atom, `SELECT id::text FROM tenants`)
	if err != nil {
		return err
	}
	atomEntities, err := idSet(ctx, m.atom, `SELECT id::text FROM entities`)
	if err != nil {
		return err
	}
	atomAuthenticatedUsers, err := idSet(ctx, m.atom,
		`SELECT entity_id::text FROM principal_group_members WHERE group_id = $1`, authenticatedUsersGroupID)
	if err != nil {
		return err
	}
	atomResources, err := idSet(ctx, m.atom, `SELECT id::text FROM resources`)
	if err != nil {
		return err
	}
	atomGroups, err := idSet(ctx, m.atom, `SELECT id::text FROM object_groups`)
	if err != nil {
		return err
	}

	// 1. tenants
	doms, err := readDomains(ctx, m.domainsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "tenants", idsOf(len(doms), func(i int) (string, bool) { return doms[i].ID, true }), atomTenants)

	// 2. human entities
	users, err := readUsers(ctx, m.usersDB)
	if err != nil {
		return err
	}
	userIDs := idsOf(len(users), func(i int) (string, bool) { return users[i].ID, true })
	m.reconcile(rep, "entities.users", userIDs, atomEntities)
	m.reconcile(rep, "principal_group_members.authenticated_users", userIDs, atomAuthenticatedUsers)

	// 3. device entities (only those with a valid domain were migrated)
	clients, err := readClients(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "entities.clients", idsOf(len(clients), func(i int) (string, bool) {
		return clients[i].ID, domSet[clients[i].DomainID]
	}), atomEntities)

	// 4. resources
	chans, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "resources.channels", idsOf(len(chans), func(i int) (string, bool) {
		return chans[i].ID, domSet[chans[i].DomainID]
	}), atomResources)

	// 4b. resources: rules, reports, alarms
	rules, err := readRules(ctx, m.reDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "resources.rules", idsOf(len(rules), func(i int) (string, bool) {
		return rules[i].ID, domSet[rules[i].DomainID]
	}), atomResources)
	reports, err := readReports(ctx, m.reportsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "resources.reports", idsOf(len(reports), func(i int) (string, bool) {
		return reports[i].ID, domSet[reports[i].DomainID]
	}), atomResources)
	alarms, err := readAlarms(ctx, m.alarmsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "resources.alarms", idsOf(len(alarms), func(i int) (string, bool) {
		return alarms[i].ID, domSet[alarms[i].DomainID]
	}), atomResources)

	// 5. object_groups
	grps, err := readGroups(ctx, m.groupsDB)
	if err != nil {
		return err
	}
	m.reconcile(rep, "object_groups", idsOf(len(grps), func(i int) (string, bool) {
		return grps[i].ID, domSet[grps[i].DomainID]
	}), atomGroups)

	// 6. authz spot-check: every connection must have a device->channel policy.
	if err := m.verifyConnections(ctx, rep, domSet); err != nil {
		return err
	}
	return nil
}

func (m *migrator) domainSet(ctx context.Context) (map[string]bool, error) {
	doms, err := readDomains(ctx, m.domainsDB)
	if err != nil {
		return nil, err
	}
	s := map[string]bool{}
	for _, d := range doms {
		s[d.ID] = true
	}
	return s, nil
}

// reconcile counts how many expected ids are missing from the atom set.
func (m *migrator) reconcile(rep *report, label string, expected []string, atom map[string]bool) {
	missing := 0
	for _, id := range expected {
		if !atom[id] {
			missing++
		}
	}
	rep.count("verify."+label+".expected", len(expected))
	if missing > 0 {
		rep.blockf("verify %s: %d of %d expected rows missing from Atom", label, missing, len(expected))
	} else {
		rep.count("verify."+label+".ok", len(expected))
	}
}

func (m *migrator) verifyConnections(ctx context.Context, rep *report, domSet map[string]bool) error {
	cli, err := readConnections(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	ch, err := readConnections(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	// Build atom edge set: subject_id | channel(object_id) | action.
	edges := map[string]bool{}
	rows, err := m.atom.QueryxContext(ctx, `
		SELECT dp.subject_id::text, pb.object_id::text, a.name
		FROM direct_policies dp
		JOIN permission_blocks pb ON pb.id = dp.permission_block_id
		JOIN permission_block_actions pba ON pba.permission_block_id = pb.id
		JOIN actions a ON a.id = pba.action_id
		WHERE pb.scope_mode = 'object' AND pb.object_kind = 'resource'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var s, o, act string
		if err := rows.Scan(&s, &o, &act); err != nil {
			rows.Close()
			return err
		}
		edges[s+"|"+o+"|"+act] = true
	}
	rows.Close()

	seen := map[string]bool{}
	expected, missing := 0, 0
	for _, c := range append(cli, ch...) {
		act, ok := connectionAction(c.Type)
		if !ok || !domSet[c.DomainID] {
			continue
		}
		k := c.ClientID + "|" + c.ChannelID + "|" + act
		if seen[k] {
			continue
		}
		seen[k] = true
		expected++
		if !edges[k] {
			missing++
		}
	}
	rep.count("verify.connections.expected", expected)
	if missing > 0 {
		rep.blockf("verify connections: %d of %d device->channel edges missing", missing, expected)
	} else {
		rep.count("verify.connections.ok", expected)
	}
	return nil
}

// --- helpers ---

func idSet(ctx context.Context, db *sqlx.DB, query string, args ...any) (map[string]bool, error) {
	rows, err := db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	s := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		s[id] = true
	}
	return s, rows.Err()
}

// idsOf collects ids for indices 0..n-1 where the picker's second return is true.
func idsOf(n int, pick func(i int) (string, bool)) []string {
	out := make([]string, 0, n)
	for i := range n {
		if id, ok := pick(i); ok {
			out = append(out, id)
		}
	}
	return out
}
