// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type migrator struct {
	cfg   config
	apply bool

	domainsDB  *sqlx.DB
	usersDB    *sqlx.DB
	clientsDB  *sqlx.DB
	channelsDB *sqlx.DB
	groupsDB   *sqlx.DB
	authDB     *sqlx.DB
	reDB       *sqlx.DB
	reportsDB  *sqlx.DB
	alarmsDB   *sqlx.DB
	atom       *sqlx.DB

	profileID        map[string]string // profile key (e.g. "user","client") -> uuid
	profileVersionID map[string]string // profile key -> latest active profile_versions.id
	actionID         map[string]string // action name -> uuid

	migratedUsers map[string]bool
	tenants       map[string]bool   // domain ids that became tenants
	clientDomain  map[string]string // client id -> domain id
	channelDomain map[string]string
	groupDomain   map[string]string

	// Collision-free names/aliases computed by buildDedup (Atom enforces unique
	// constraints Magistrala dropped). Keyed by source id; "" alias = NULL.
	tenantName   map[string]string
	userName     map[string]string
	deviceName   map[string]string
	groupName    map[string]string
	tenantAlias  map[string]string
	clientAlias  map[string]string
	channelAlias map[string]string

	reportDir  string
	deviceKeys [][]string // client_id, domain_id, identity, plaintext key (apply only)
}

func newMigrator(ctx context.Context, cfg config, apply bool) (*migrator, error) {
	m := &migrator{
		cfg:              cfg,
		apply:            apply,
		profileID:        map[string]string{},
		profileVersionID: map[string]string{},
		actionID:         map[string]string{},
		migratedUsers:    map[string]bool{},
		tenants:          map[string]bool{},
		clientDomain:     map[string]string{},
		channelDomain:    map[string]string{},
		groupDomain:      map[string]string{},
		tenantName:       map[string]string{},
		userName:         map[string]string{},
		deviceName:       map[string]string{},
		groupName:        map[string]string{},
		tenantAlias:      map[string]string{},
		clientAlias:      map[string]string{},
		channelAlias:     map[string]string{},
	}
	var err error
	open := func(name, dsn string) *sqlx.DB {
		if err != nil {
			return nil
		}
		var db *sqlx.DB
		db, err = openDB(ctx, name, dsn)
		return db
	}
	m.domainsDB = open("domains", cfg.Domains.DSN())
	m.usersDB = open("users", cfg.Users.DSN())
	m.clientsDB = open("clients", cfg.Clients.DSN())
	m.channelsDB = open("channels", cfg.Channels.DSN())
	m.groupsDB = open("groups", cfg.Groups.DSN())
	m.authDB = open("auth", cfg.Auth.DSN())
	m.reDB = open("rules_engine", cfg.RE.DSN())
	m.reportsDB = open("reports", cfg.Reports.DSN())
	m.alarmsDB = open("alarms", cfg.Alarms.DSN())
	m.atom = open("atom", cfg.AtomDSN)
	if err != nil {
		return nil, err
	}
	return m, nil
}

const authenticatedUsersGroupID = "00000000-0000-0000-0000-000000000005"

func (m *migrator) Close() {
	for _, db := range []*sqlx.DB{m.domainsDB, m.usersDB, m.clientsDB, m.channelsDB, m.groupsDB, m.authDB, m.reDB, m.reportsDB, m.alarmsDB, m.atom} {
		if db != nil {
			_ = db.Close()
		}
	}
}

func (m *migrator) Run(ctx context.Context, rep *report) error {
	if err := m.loadLookups(ctx); err != nil {
		return fmt.Errorf("load atom lookups: %w", err)
	}
	if err := m.preflight(ctx, rep); err != nil {
		return fmt.Errorf("preflight: %w", err)
	}
	if err := m.preflightGate(rep); err != nil {
		return err
	}
	if err := m.buildDedup(ctx, rep); err != nil {
		return fmt.Errorf("dedup: %w", err)
	}
	// Order is FK-safe; see PLAN.md §7.
	phases := []struct {
		name string
		fn   func(context.Context, *report) error
	}{
		{"tenants", m.phaseTenants},
		{"entities.users", m.phaseUsers},
		{"entities.clients", m.phaseClients},
		{"credentials.devices", m.phaseDeviceCreds},
		{"resources.channels", m.phaseChannels},
		{"resources.rules", m.phaseRules},
		{"resources.reports", m.phaseReports},
		{"resources.alarms", m.phaseAlarms},
		{"object_groups", m.phaseGroups},
		{"group_membership", m.phaseGroupMembership},
		{"roles", m.phaseRoles},
		{"connections", m.phaseConnections},
		{"credentials.pats", m.phasePATs},
		{"tenant_invitations", m.phaseInvitations},
		{"backfill.tenant_actors", m.phaseBackfill},
	}
	for _, p := range phases {
		if err := p.fn(ctx, rep); err != nil {
			return fmt.Errorf("phase %s: %w", p.name, err)
		}
	}
	if m.apply && len(m.deviceKeys) > 0 {
		if err := m.writeDeviceKeys(); err != nil {
			return fmt.Errorf("write device keys: %w", err)
		}
	}
	return nil
}

// writeDeviceKeys exports the re-issued device API keys (secret shown once) for
// re-provisioning. Treat the file as a secret and delete after use.
func (m *migrator) writeDeviceKeys() error {
	if err := os.MkdirAll(m.reportDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(m.reportDir, "device-keys-"+time.Now().UTC().Format("20060102-150405")+".csv")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"client_id", "domain_id", "identity", "api_key"}); err != nil {
		return err
	}
	return w.WriteAll(m.deviceKeys)
}

func (m *migrator) loadLookups(ctx context.Context) error {
	rows, err := m.atom.QueryxContext(ctx,
		`SELECT p.key, p.id, pv.id
		   FROM profiles p
		   LEFT JOIN LATERAL (
		       SELECT id
		       FROM profile_versions
		       WHERE profile_id = p.id
		         AND status = 'active'
		       ORDER BY version DESC
		       LIMIT 1
		   ) pv ON TRUE
		  WHERE p.object_kind = 'entity'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var k, id string
		var versionID sql.NullString
		if err := rows.Scan(&k, &id, &versionID); err != nil {
			return err
		}
		m.profileID[k] = id
		if versionID.Valid {
			m.profileVersionID[k] = versionID.String
		}
	}
	rows.Close()

	ar, err := m.atom.QueryxContext(ctx, `SELECT name, id FROM actions`)
	if err != nil {
		return err
	}
	defer ar.Close()
	for ar.Next() {
		var n, id string
		if err := ar.Scan(&n, &id); err != nil {
			return err
		}
		m.actionID[n] = id
	}
	return nil
}

// exec runs a write in apply mode; in dry-run it is a no-op. Returns the row
// count target for reporting (always 1 on success path).
func (m *migrator) exec(ctx context.Context, query string, args ...any) error {
	if !m.apply {
		return nil
	}
	_, err := m.atom.ExecContext(ctx, query, args...)
	return err
}

// --- phases ---

func (m *migrator) phaseTenants(ctx context.Context, rep *report) error {
	rows, err := readDomains(ctx, m.domainsDB)
	if err != nil {
		return err
	}
	for _, d := range rows {
		alias := aliasOrNil(m.tenantAlias[d.ID])
		if err := m.exec(ctx,
			`INSERT INTO tenants (id, name, alias, status, tags, attributes, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
			d.ID, m.tenantName[d.ID], alias, tenantStatus(d.Status),
			pqArr(d.Tags), attrs(d.Metadata, nil), ntToTime(d.CreatedAt), ntPtr(d.UpdatedAt),
		); err != nil {
			return err
		}
		m.tenants[d.ID] = true
		rep.count("tenants", 1)
	}
	return nil
}

func (m *migrator) phaseUsers(ctx context.Context, rep *report) error {
	rows, err := readUsers(ctx, m.usersDB)
	if err != nil {
		return err
	}
	prof := nullStr(m.profileID["user"])
	profVer := nullStr(m.profileVersionID["user"])
	for _, u := range rows {
		extra := map[string]any{}
		putStr(extra, "first_name", u.FirstName)
		putStr(extra, "last_name", u.LastName)
		putStr(extra, "username", u.Username)
		putStr(extra, "profile_picture", u.ProfilePicture)
		putStr(extra, "auth_provider", u.AuthProvider)
		name := m.userName[u.ID]

		if err := m.exec(ctx,
			`INSERT INTO entities (id, kind, name, tenant_id, status, attributes, profile_id, profile_version_id, created_at, updated_at)
			 VALUES ($1,'human',$2,NULL,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
			u.ID, name, entityStatus(u.Status), attrs(u.Metadata, extra), prof,
			profVer, ntToTime(u.CreatedAt), ntPtr(u.UpdatedAt),
		); err != nil {
			return err
		}
		m.migratedUsers[u.ID] = true
		rep.count("entities.users", 1)

		if err := m.exec(ctx,
			`INSERT INTO principal_group_members (group_id, entity_id)
			 VALUES ($1,$2) ON CONFLICT DO NOTHING`,
			authenticatedUsersGroupID, u.ID,
		); err != nil {
			return err
		}
		rep.count("principal_group_members.authenticated_users", 1)

		// email (force-reset: no password credential migrated)
		if u.Email.Valid && u.Email.String != "" {
			if err := m.exec(ctx,
				`INSERT INTO entity_emails (entity_id, email, verified_at)
				 VALUES ($1,$2,$3) ON CONFLICT (entity_id) DO NOTHING`,
				u.ID, u.Email.String, ntPtr(u.VerifiedAt),
			); err != nil {
				// likely global email-unique conflict
				rep.warnf("user %s email %q not inserted (conflict?)", u.ID, u.Email.String)
				rep.skip("email_conflict")
			} else {
				rep.count("entity_emails", 1)
			}
		}
		rep.todo("password_reset", fmt.Sprintf("%s (%s)", u.ID, u.Email.String))

		// platform admin role assignment
		if u.Role.Valid && u.Role.Int16 == 1 {
			if err := m.exec(ctx,
				`INSERT INTO role_assignments (id, tenant_id, subject_kind, subject_id, role_id)
				 VALUES ($1,NULL,'entity',$2,'00000000-0000-0000-0000-000000000002')
				 ON CONFLICT DO NOTHING`,
				derivedUUID("ra", "admin", u.ID), u.ID,
			); err != nil {
				return err
			}
			rep.count("role_assignments", 1)
		}
	}
	return nil
}

func (m *migrator) phaseClients(ctx context.Context, rep *report) error {
	rows, err := readClients(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	prof := nullStr(m.profileID["client"])
	profVer := nullStr(m.profileVersionID["client"])
	for _, c := range rows {
		if !m.tenants[c.DomainID] {
			rep.skip("client_orphan_domain")
			rep.warnf("client %s skipped: domain %s not migrated", c.ID, c.DomainID)
			continue
		}
		m.clientDomain[c.ID] = c.DomainID
		extra := map[string]any{}
		putStr(extra, "identity", c.Identity)
		alias := aliasOrNil(m.clientAlias[c.ID])
		name := m.deviceName[c.ID]
		if err := m.exec(ctx,
			`INSERT INTO entities (id, kind, name, tenant_id, status, attributes, profile_id, profile_version_id, alias, created_at, updated_at)
			 VALUES ($1,'device',$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (id) DO NOTHING`,
			c.ID, name, c.DomainID, entityStatus(c.Status), attrs(c.Metadata, extra),
			prof, profVer, alias, ntToTime(c.CreatedAt), ntPtr(c.UpdatedAt),
		); err != nil {
			return err
		}
		rep.count("entities.clients", 1)
	}
	return nil
}

func (m *migrator) phaseDeviceCreds(ctx context.Context, rep *report) error {
	rows, err := readClients(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	for _, c := range rows {
		if _, ok := m.clientDomain[c.ID]; !ok || !c.Secret.Valid || c.Secret.String == "" {
			continue
		}
		// Atom cannot reuse the Magistrala secret (format + lookup differ); the
		// device key is re-issued in atom_<credId>_<secret> form and exported for
		// re-provisioning. See newAtomAPIKey / PLAN §5.
		credID, plaintext, hash, err := newAtomAPIKey(c.ID)
		if err != nil {
			return err
		}
		if err := m.exec(ctx,
			`INSERT INTO credentials (id, entity_id, kind, identifier, secret_hash, metadata, status)
			 VALUES ($1,$2,'api_key',$3,$4,'{"source":"magistrala-client-reissued"}',$5)
			 ON CONFLICT (id) DO NOTHING`,
			credID, c.ID, c.ID, hash, statusCred(c.Status),
		); err != nil {
			return err
		}
		if m.apply {
			m.deviceKeys = append(m.deviceKeys, []string{c.ID, c.DomainID, c.Identity.String, plaintext})
		}
		rep.count("credentials.devices", 1)
	}
	if len(m.deviceKeys) > 0 {
		rep.todo("device_reprovision", fmt.Sprintf("%d device keys re-issued -> see device-keys CSV in report dir", len(m.deviceKeys)))
	}
	return nil
}

func (m *migrator) phaseChannels(ctx context.Context, rep *report) error {
	rows, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	for _, ch := range rows {
		if !m.tenants[ch.DomainID] {
			rep.skip("channel_orphan_domain")
			continue
		}
		m.channelDomain[ch.ID] = ch.DomainID
		alias := aliasOrNil(m.channelAlias[ch.ID])
		owner := sql.NullString{}
		if ch.CreatedBy.Valid && m.migratedUsers[ch.CreatedBy.String] {
			owner = ch.CreatedBy
		}
		extra := map[string]any{"status": entityStatus(ch.Status)}
		if err := m.exec(ctx,
			`INSERT INTO resources (id, kind, name, tenant_id, owner_id, attributes, alias, created_at, updated_at)
			 VALUES ($1,'channel',$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
			ch.ID, nsToStr(ch.Name), ch.DomainID, owner, attrs(ch.Metadata, extra),
			alias, ntToTime(ch.CreatedAt), ntPtr(ch.UpdatedAt),
		); err != nil {
			return err
		}
		rep.count("resources.channels", 1)
	}
	return nil
}

// insertResource writes one row into Atom resources (kind = channel/rule/report/
// alarm). Entity-specific columns Magistrala has but Atom resources lack are
// folded into the attributes JSONB. ON CONFLICT (id) keeps it idempotent.
func (m *migrator) insertResource(ctx context.Context, id, kind, name, tenant string, owner sql.NullString, attributes string, createdAt time.Time, updatedAt any) error {
	return m.exec(ctx,
		`INSERT INTO resources (id, kind, name, tenant_id, owner_id, attributes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
		id, kind, name, tenant, owner, attributes, createdAt, updatedAt)
}

// ownerOf returns created_by only when it maps to a migrated human (resources.
// owner_id is an FK to entities); otherwise NULL.
func (m *migrator) ownerOf(createdBy sql.NullString) sql.NullString {
	if createdBy.Valid && m.migratedUsers[createdBy.String] {
		return createdBy
	}
	return sql.NullString{}
}

// uniqueResName makes a resource name unique within a tenant (Atom enforces
// resources(name, tenant_id); rules/reports/alarms carry no such Magistrala
// constraint, so same-tenant dups are possible). Suffixes -2, -3, … on collision.
func uniqueResName(seen map[string]int, tenant, name string) string {
	key := tenant + "|" + strings.ToLower(name)
	seen[key]++
	if seen[key] == 1 {
		return name
	}
	return fmt.Sprintf("%s-%d", name, seen[key])
}

// phaseRules: rules_engine.rules -> resources (kind=rule).
func (m *migrator) phaseRules(ctx context.Context, rep *report) error {
	rows, err := readRules(ctx, m.reDB)
	if err != nil {
		return err
	}
	seen := map[string]int{}
	for _, r := range rows {
		if !m.tenants[r.DomainID] {
			rep.skip("rule_orphan_domain")
			continue
		}
		name := uniqueResName(seen, r.DomainID, firstNonEmpty(nsToStr(r.Name), r.ID))
		extra := map[string]any{
			"status":     entityStatus(r.Status),
			"logic_type": r.LogicType,
		}
		putStr(extra, "input_channel", r.InputChannel)
		putStr(extra, "input_topic", r.InputTopic)
		putStr(extra, "updated_by", r.UpdatedBy)
		if len(r.Outputs) > 0 {
			extra["outputs"] = r.Outputs
		}
		if len(r.LogicValue) > 0 {
			extra["logic_value"] = r.LogicValue // base64-encoded in JSON
		}
		if r.Recurring.Valid {
			extra["recurring"] = r.Recurring.Int16
		}
		if r.RecurringPeriod.Valid {
			extra["recurring_period"] = r.RecurringPeriod.Int16
		}
		if r.Time.Valid {
			extra["time"] = r.Time.Time
		}
		if r.StartDatetime.Valid {
			extra["start_datetime"] = r.StartDatetime.Time
		}
		if len(r.Tags) > 0 {
			extra["tags"] = []string(r.Tags)
		}
		if err := m.insertResource(ctx, r.ID, "rule", name, r.DomainID, m.ownerOf(r.CreatedBy),
			attrs(r.Metadata, extra), ntToTime(r.CreatedAt), ntPtr(r.UpdatedAt)); err != nil {
			return err
		}
		rep.count("resources.rules", 1)
	}
	return nil
}

// phaseReports: reports.report_config -> resources (kind=report).
func (m *migrator) phaseReports(ctx context.Context, rep *report) error {
	rows, err := readReports(ctx, m.reportsDB)
	if err != nil {
		return err
	}
	seen := map[string]int{}
	for _, rp := range rows {
		if !m.tenants[rp.DomainID] {
			rep.skip("report_orphan_domain")
			continue
		}
		name := uniqueResName(seen, rp.DomainID, firstNonEmpty(nsToStr(rp.Name), rp.ID))
		extra := map[string]any{"status": entityStatus(rp.Status)}
		putStr(extra, "description", rp.Description)
		putStr(extra, "report_template", rp.ReportTemplate)
		putStr(extra, "updated_by", rp.UpdatedBy)
		if len(rp.Config) > 0 {
			extra["config"] = rp.Config
		}
		if len(rp.Email) > 0 {
			extra["email"] = rp.Email
		}
		if len(rp.Metrics) > 0 {
			extra["metrics"] = rp.Metrics
		}
		if rp.Due.Valid {
			extra["due"] = rp.Due.Time
		}
		if rp.Recurring.Valid {
			extra["recurring"] = rp.Recurring.Int16
		}
		if rp.RecurringPeriod.Valid {
			extra["recurring_period"] = rp.RecurringPeriod.Int16
		}
		if rp.StartDatetime.Valid {
			extra["start_datetime"] = rp.StartDatetime.Time
		}
		if err := m.insertResource(ctx, rp.ID, "report", name, rp.DomainID, m.ownerOf(rp.CreatedBy),
			attrs(nil, extra), ntToTime(rp.CreatedAt), ntPtr(rp.UpdatedAt)); err != nil {
			return err
		}
		rep.count("resources.reports", 1)
	}
	return nil
}

// phaseAlarms: alarms.alarms -> resources (kind=alarm). Alarms have no name in
// Magistrala; the measurement is used (id fallback), deduped per tenant.
func (m *migrator) phaseAlarms(ctx context.Context, rep *report) error {
	rows, err := readAlarms(ctx, m.alarmsDB)
	if err != nil {
		return err
	}
	seen := map[string]int{}
	for _, a := range rows {
		if !m.tenants[a.DomainID] {
			rep.skip("alarm_orphan_domain")
			continue
		}
		name := uniqueResName(seen, a.DomainID, firstNonEmpty(a.Measurement, a.ID))
		extra := map[string]any{
			"rule_id":      a.RuleID,
			"channel_id":   a.ChannelID,
			"client_id":    a.ClientID,
			"subtopic":     a.Subtopic,
			"measurement":  a.Measurement,
			"value":        a.Value,
			"unit":         a.Unit,
			"threshold":    a.Threshold,
			"cause":        a.Cause,
			"alarm_status": a.Status,
			"severity":     a.Severity,
		}
		putStr(extra, "assignee_id", a.AssigneeID)
		putStr(extra, "updated_by", a.UpdatedBy)
		putStr(extra, "assigned_by", a.AssignedBy)
		putStr(extra, "acknowledged_by", a.AcknowledgedBy)
		putStr(extra, "resolved_by", a.ResolvedBy)
		if a.AssignedAt.Valid {
			extra["assigned_at"] = a.AssignedAt.Time
		}
		if a.AcknowledgedAt.Valid {
			extra["acknowledged_at"] = a.AcknowledgedAt.Time
		}
		if a.ResolvedAt.Valid {
			extra["resolved_at"] = a.ResolvedAt.Time
		}
		// Alarms carry no created_by; owner_id stays NULL.
		if err := m.insertResource(ctx, a.ID, "alarm", name, a.DomainID, sql.NullString{},
			attrs(a.Metadata, extra), ntToTime(a.CreatedAt), ntPtr(a.UpdatedAt)); err != nil {
			return err
		}
		rep.count("resources.alarms", 1)
	}
	return nil
}

func (m *migrator) phaseGroups(ctx context.Context, rep *report) error {
	rows, err := readGroups(ctx, m.groupsDB)
	if err != nil {
		return err
	}
	// First pass: groups. Second pass: hierarchy (parent must exist).
	for _, g := range rows {
		if !m.tenants[g.DomainID] {
			rep.skip("group_orphan_domain")
			continue
		}
		m.groupDomain[g.ID] = g.DomainID
		if err := m.exec(ctx,
			`INSERT INTO object_groups (id, name, tenant_id, description, status, attributes, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (id) DO NOTHING`,
			g.ID, m.groupName[g.ID], g.DomainID, nsToStr(g.Description), entityStatus(g.Status),
			attrs(g.Metadata, map[string]any{"tags": []string(g.Tags)}),
			ntToTime(g.CreatedAt), ntToTime(g.UpdatedAt), // object_groups.updated_at is NOT NULL
		); err != nil {
			return err
		}
		rep.count("object_groups", 1)
	}
	for _, g := range rows {
		_, haveChild := m.groupDomain[g.ID]
		_, haveParent := m.groupDomain[g.ParentID.String]
		if !g.ParentID.Valid || !haveChild || !haveParent {
			continue
		}
		if err := m.exec(ctx,
			`INSERT INTO object_group_hierarchy (parent_id, child_id, tenant_id)
			 VALUES ($1,$2,$3) ON CONFLICT (child_id) DO NOTHING`,
			g.ParentID.String, g.ID, g.DomainID,
		); err != nil {
			return err
		}
		rep.count("object_group_hierarchy", 1)
	}
	return nil
}

func (m *migrator) phaseGroupMembership(ctx context.Context, rep *report) error {
	// clients -> object_group_entities
	clients, err := readClients(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	for _, c := range clients {
		_, haveClient := m.clientDomain[c.ID]
		_, haveGroup := m.groupDomain[c.ParentGroupID.String]
		if !c.ParentGroupID.Valid || !haveClient || !haveGroup {
			continue
		}
		if err := m.exec(ctx,
			`INSERT INTO object_group_entities (group_id, entity_id, tenant_id)
			 VALUES ($1,$2,$3) ON CONFLICT (entity_id) DO NOTHING`,
			c.ParentGroupID.String, c.ID, c.DomainID,
		); err != nil {
			return err
		}
		rep.count("object_group_entities", 1)
	}
	// channels -> object_group_resources
	chans, err := readChannels(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	for _, ch := range chans {
		_, haveChan := m.channelDomain[ch.ID]
		_, haveGroup := m.groupDomain[ch.ParentGroupID.String]
		if !ch.ParentGroupID.Valid || !haveChan || !haveGroup {
			continue
		}
		if err := m.exec(ctx,
			`INSERT INTO object_group_resources (group_id, resource_id, tenant_id)
			 VALUES ($1,$2,$3) ON CONFLICT (resource_id) DO NOTHING`,
			ch.ParentGroupID.String, ch.ID, ch.DomainID,
		); err != nil {
			return err
		}
		rep.count("object_group_resources", 1)
	}
	return nil
}

// roleScope describes how a Magistrala role family maps to a permission_block.
type roleScope struct {
	prefix    string
	db        *sqlx.DB
	domainOf  func(entityID string) (string, bool) // object id -> tenant id
	scopeMode string
	objKind   string // for object scope
}

func (m *migrator) phaseRoles(ctx context.Context, rep *report) error {
	families := []roleScope{
		{"domains", m.domainsDB, func(id string) (string, bool) { return id, m.tenants[id] }, "tenant", ""},
		{"clients", m.clientsDB, func(id string) (string, bool) { d, ok := m.clientDomain[id]; return d, ok }, "object", "entity"},
		{"channels", m.channelsDB, func(id string) (string, bool) { d, ok := m.channelDomain[id]; return d, ok }, "object", "resource"},
		{"groups", m.groupsDB, func(id string) (string, bool) { d, ok := m.groupDomain[id]; return d, ok }, "group", ""},
	}
	for _, f := range families {
		if err := m.migrateRoleFamily(ctx, rep, f); err != nil {
			return fmt.Errorf("%s roles: %w", f.prefix, err)
		}
	}
	return nil
}

func (m *migrator) migrateRoleFamily(ctx context.Context, rep *report, f roleScope) error {
	roles, acts, mems, err := readRoleFamily(ctx, f.db, f.prefix)
	if err != nil {
		return err
	}
	actsByRole := map[string][]string{}
	for _, a := range acts {
		actsByRole[a.RoleID] = append(actsByRole[a.RoleID], a.Action)
	}
	for _, r := range roles {
		tenant, ok := f.domainOf(r.EntityID)
		if !ok {
			rep.skip("role_orphan_object")
			continue
		}
		roleID := derivedUUID("role", f.prefix, r.ID)
		blockID := derivedUUID("block", f.prefix, r.ID)

		// Atom roles are unique on (name, tenant_id). Magistrala object roles are
		// per-instance, so many objects in one tenant can share a role name
		// (e.g. every client has an "admin" role). Embed the object id to keep the
		// Atom role name unique within the tenant.
		roleName := f.prefix + ":" + r.EntityID + ":" + r.Name
		if f.scopeMode == "tenant" {
			roleName = f.prefix + ":" + r.Name // domain roles: one set per tenant
		}
		if err := m.exec(ctx,
			`INSERT INTO roles (id, name, tenant_id, description, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING`,
			roleID, roleName, tenant, "migrated from "+f.prefix+"_roles "+r.ID,
			ntToTime(r.CreatedAt), ntPtr(r.UpdatedAt),
		); err != nil {
			return err
		}
		rep.count("roles", 1)

		if err := m.insertBlock(ctx, blockID, f, r.EntityID, tenant); err != nil {
			return err
		}
		if err := m.exec(ctx,
			`INSERT INTO role_permission_blocks (role_id, permission_block_id)
			 VALUES ($1,$2) ON CONFLICT DO NOTHING`, roleID, blockID); err != nil {
			return err
		}

		// actions -> permission_block_actions
		seen := map[string]bool{}
		for _, raw := range actsByRole[r.ID] {
			atom, ok := mapAction(raw)
			if !ok {
				if m.cfg.UnmappedAction == "skip" {
					rep.skip("action_unmapped_skipped")
					rep.warnf("unmapped action %q (%s) skipped", raw, f.prefix)
					continue
				}
				atom = "manage"
				rep.warnf("unmapped action %q (%s) -> manage", raw, f.prefix)
			}
			aid, ok := m.actionID[atom]
			if !ok || seen[atom] {
				continue
			}
			seen[atom] = true
			if err := m.exec(ctx,
				`INSERT INTO permission_block_actions (permission_block_id, action_id)
				 VALUES ($1,$2) ON CONFLICT DO NOTHING`, blockID, aid); err != nil {
				return err
			}
			rep.count("permission_block_actions", 1)
		}

		// members -> role_assignments (+ tenant_memberships for domain roles)
		for _, mem := range mems {
			if mem.RoleID != r.ID {
				continue
			}
			if err := m.exec(ctx,
				`INSERT INTO role_assignments (id, tenant_id, subject_kind, subject_id, role_id)
				 VALUES ($1,$2,'entity',$3,$4) ON CONFLICT DO NOTHING`,
				derivedUUID("ra", f.prefix, r.ID, mem.MemberID), tenant, mem.MemberID, roleID,
			); err != nil {
				return err
			}
			rep.count("role_assignments", 1)
			if f.prefix == "domains" {
				if err := m.exec(ctx,
					`INSERT INTO tenant_memberships (tenant_id, entity_id, status)
					 VALUES ($1,$2,'active') ON CONFLICT DO NOTHING`,
					tenant, mem.MemberID); err != nil {
					return err
				}
				rep.count("tenant_memberships", 1)
			}
		}
	}
	return nil
}

func (m *migrator) insertBlock(ctx context.Context, blockID string, f roleScope, objectID, tenant string) error {
	switch f.scopeMode {
	case "tenant":
		return m.exec(ctx,
			`INSERT INTO permission_blocks (id, tenant_id, scope_mode, effect, conditions)
			 VALUES ($1,$2,'tenant','allow','{}') ON CONFLICT (id) DO NOTHING`, blockID, tenant)
	case "group":
		return m.exec(ctx,
			`INSERT INTO permission_blocks (id, tenant_id, scope_mode, group_id, effect, conditions)
			 VALUES ($1,$2,'group',$3,'allow','{}') ON CONFLICT (id) DO NOTHING`, blockID, tenant, objectID)
	default: // object
		return m.exec(ctx,
			`INSERT INTO permission_blocks (id, tenant_id, scope_mode, object_kind, object_id, effect, conditions)
			 VALUES ($1,$2,'object',$3,$4,'allow','{}') ON CONFLICT (id) DO NOTHING`,
			blockID, tenant, f.objKind, objectID)
	}
}

func (m *migrator) phaseConnections(ctx context.Context, rep *report) error {
	// Both clients-db and channels-db keep their own connections copy; union and
	// dedup so neither side's view is missed.
	cliConns, err := readConnections(ctx, m.clientsDB)
	if err != nil {
		return err
	}
	chConns, err := readConnections(ctx, m.channelsDB)
	if err != nil {
		return err
	}
	seenConn := map[string]bool{}
	conns := make([]srcConnection, 0, len(cliConns)+len(chConns))
	for _, c := range append(cliConns, chConns...) {
		k := c.ChannelID + "|" + c.ClientID + "|" + c.DomainID + "|" + fmt.Sprint(c.Type)
		if seenConn[k] {
			continue
		}
		seenConn[k] = true
		conns = append(conns, c)
	}
	for _, c := range conns {
		dom, ok := m.channelDomain[c.ChannelID]
		if !ok {
			rep.skip("conn_orphan_channel")
			continue
		}
		act, ok := connectionAction(c.Type)
		if !ok {
			rep.skip("conn_bad_type")
			continue
		}
		aid, ok := m.actionID[act]
		if !ok {
			continue
		}
		blockID := derivedUUID("connblock", c.ChannelID, fmt.Sprint(c.Type))
		if err := m.exec(ctx,
			`INSERT INTO permission_blocks (id, tenant_id, scope_mode, object_kind, object_id, effect, conditions)
			 VALUES ($1,$2,'object','resource',$3,'allow','{}') ON CONFLICT (id) DO NOTHING`,
			blockID, dom, c.ChannelID); err != nil {
			return err
		}
		if err := m.exec(ctx,
			`INSERT INTO permission_block_actions (permission_block_id, action_id)
			 VALUES ($1,$2) ON CONFLICT DO NOTHING`, blockID, aid); err != nil {
			return err
		}
		if err := m.exec(ctx,
			`INSERT INTO direct_policies (id, tenant_id, subject_kind, subject_id, permission_block_id)
			 VALUES ($1,$2,'entity',$3,$4) ON CONFLICT (id) DO NOTHING`,
			derivedUUID("dp", c.ChannelID, c.ClientID, fmt.Sprint(c.Type)), dom, c.ClientID, blockID,
		); err != nil {
			return err
		}
		rep.count("connections", 1)
	}
	return nil
}

func (m *migrator) phasePATs(ctx context.Context, rep *report) error {
	pats, err := readPATs(ctx, m.authDB)
	if err != nil {
		return err
	}
	scopeRows, err := readPATScopes(ctx, m.authDB)
	if err != nil {
		return err
	}
	scopesByPAT := map[string][]map[string]string{}
	for _, s := range scopeRows {
		scopesByPAT[s.PatID] = append(scopesByPAT[s.PatID], map[string]string{
			"domain_id": s.DomainID.String, "entity_type": s.EntityType,
			"operation": s.Operation, "entity_id": s.EntityID,
		})
	}
	for _, p := range pats {
		if !p.UserID.Valid || !m.migratedUsers[p.UserID.String] {
			rep.skip("pat_orphan_user")
			continue
		}
		status := "active"
		if p.Revoked.Valid && p.Revoked.Bool {
			status = "revoked"
		}
		meta, err := json.Marshal(map[string]any{
			"source": "magistrala-pat", "name": p.Name, "description": p.Desc.String,
			"needs_reissue": true, "scopes": scopesByPAT[p.ID],
		})
		if err != nil {
			return err
		}
		// secret_hash NULL: Magistrala PAT secret is not convertible (see PLAN §5).
		if err := m.exec(ctx,
			`INSERT INTO credentials (id, entity_id, kind, identifier, metadata, status, expires_at)
			 VALUES ($1,$2,'api_key',$3,$4,$5,$6) ON CONFLICT (id) DO NOTHING`,
			p.ID, p.UserID.String, p.ID, string(meta), status, ntPtr(p.ExpiresAt),
		); err != nil {
			return err
		}
		rep.count("credentials.pats", 1)
		rep.todo("pat_reissue", fmt.Sprintf("%s (user %s)", p.ID, p.UserID.String))
	}
	return nil
}

func (m *migrator) phaseInvitations(ctx context.Context, rep *report) error {
	invs, err := readInvitations(ctx, m.domainsDB)
	if err != nil {
		return err
	}
	for _, iv := range invs {
		// Only pending invitations matter; accepted ones are already memberships.
		if iv.ConfirmedAt.Valid {
			rep.skip("invitation_already_accepted")
			continue
		}
		if !m.tenants[iv.DomainID] || !m.migratedUsers[iv.InvitedBy] {
			rep.skip("invitation_orphan")
			continue
		}
		invitee := sql.NullString{}
		if m.migratedUsers[iv.InviteeID] {
			invitee = sql.NullString{String: iv.InviteeID, Valid: true}
		}
		roleID := derivedUUID("role", "domains", iv.RoleID)
		if err := m.exec(ctx,
			`INSERT INTO tenant_invitations (id, tenant_id, invitee_user_id, invited_by, role_id, created_at, rejected_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`,
			derivedUUID("inv", iv.DomainID, iv.InviteeID), iv.DomainID, invitee, iv.InvitedBy,
			roleID, ntToTime(iv.CreatedAt), ntPtr(iv.RejectedAt),
		); err != nil {
			return err
		}
		rep.count("tenant_invitations", 1)
	}
	return nil
}

// phaseBackfill sets tenants.created_by/updated_by now that entities exist (the
// FK could not be satisfied during phaseTenants). Only points at migrated users.
func (m *migrator) phaseBackfill(ctx context.Context, rep *report) error {
	doms, err := readDomains(ctx, m.domainsDB)
	if err != nil {
		return err
	}
	for _, d := range doms {
		if !m.tenants[d.ID] {
			continue
		}
		cb := actorOrNil(d.CreatedBy, m.migratedUsers)
		ub := actorOrNil(d.UpdatedBy, m.migratedUsers)
		if cb == nil && ub == nil {
			continue
		}
		if err := m.exec(ctx,
			`UPDATE tenants SET created_by = COALESCE($2, created_by),
			        updated_by = COALESCE($3, updated_by) WHERE id = $1`,
			d.ID, cb, ub); err != nil {
			return err
		}
		rep.count("backfill.tenant_actors", 1)
	}
	return nil
}

func actorOrNil(ns sql.NullString, migrated map[string]bool) any {
	if ns.Valid && migrated[ns.String] {
		return ns.String
	}
	return nil
}

// --- small helpers ---

func nsToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func ntToTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Now().UTC()
}

func ntPtr(nt sql.NullTime) any {
	if nt.Valid {
		return nt.Time
	}
	return nil
}

func pqArr(a []string) any {
	if len(a) == 0 {
		return "{}"
	}
	out := "{"
	for i, s := range a {
		if i > 0 {
			out += ","
		}
		out += `"` + s + `"`
	}
	return out + "}"
}

func putStr(m map[string]any, k string, v sql.NullString) {
	if v.Valid && v.String != "" {
		m[k] = v.String
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func statusCred(s int16) string {
	if s == 0 {
		return "active"
	}
	return "revoked"
}

// attrs merges Magistrala metadata jsonb with extra keys into an Atom attributes
// JSON string.
func attrs(meta []byte, extra map[string]any) string {
	out := map[string]any{}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &out)
	}
	for k, v := range extra {
		out[k] = v
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}
