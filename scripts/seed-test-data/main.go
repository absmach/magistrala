// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main seeds test data across the domains, RE, reports, and alarms
// databases so that the backfill-roles scripts can be tested end-to-end.
//
// It creates one domain, two users (one domain member, one not), several
// rules/reports with and without pre-existing roles, a couple of alarms, and
// optionally a SpiceDB parent relation for one rule to exercise the
// "policy already exists" path.
//
// All IDs are deterministic so re-running is idempotent (INSERT … ON CONFLICT DO NOTHING).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ---------------------------------------------------------------------------
// Configuration — edit these to match your environment.
// Default values target the standard docker-compose setup.
// ---------------------------------------------------------------------------

var (
	// Domains database.
	domainsDB = dbConfig{host: "localhost", port: "6003", user: "magistrala", pass: "magistrala", name: "domains"}

	// RE (rules engine) database.
	reDB = dbConfig{host: "localhost", port: "6009", user: "magistrala", pass: "magistrala", name: "rules_engine"}

	// Reports database.
	reportsDB = dbConfig{host: "localhost", port: "6020", user: "magistrala", pass: "magistrala", name: "reports"}

	// Alarms database.
	alarmsDB = dbConfig{host: "localhost", port: "6019", user: "magistrala", pass: "magistrala", name: "alarms"}

	// SpiceDB.
	spicedbHost         = "localhost"
	spicedbPort         = "50051"
	spicedbPreSharedKey = "12345678"

	// Whether to write a SpiceDB parent relation for rule-3 to test the
	// "policy already exists" code path.
	seedSpiceDB = true
)

// ---------------------------------------------------------------------------
// Deterministic test IDs
// ---------------------------------------------------------------------------

const (
	domainID   = "d0000000-0000-0000-0000-000000000001"
	domainName = "seed-test-domain"

	// user1 will be a domain member; user2 will NOT.
	user1ID = "u0000000-0000-0000-0000-000000000001"
	user2ID = "u0000000-0000-0000-0000-000000000002"

	// Domain role (admin).
	domainRoleID   = "dr000000-0000-0000-0000-000000000001"
	domainRoleName = "admin"

	// Rules — orphans (no rules_roles entry).
	rule1ID = "r0000000-0000-0000-0000-000000000001" // created_by=user1 (domain member)  → backfill should assign member
	rule2ID = "r0000000-0000-0000-0000-000000000002" // created_by=user2 (NOT member)     → backfill should provision role without member
	rule3ID = "r0000000-0000-0000-0000-000000000003" // created_by=user1, SpiceDB parent already exists → test policyExists
	rule4ID = "r0000000-0000-0000-0000-000000000004" // created_by=NULL                   → should be skipped
	rule5ID = "r0000000-0000-0000-0000-000000000005" // empty domain_id                   → should be skipped

	// Rule with pre-existing role — should NOT appear in orphan list.
	rule6ID     = "r0000000-0000-0000-0000-000000000006"
	rule6RoleID = "rr000000-0000-0000-0000-000000000006"

	// Reports — orphans (no reports_roles entry).
	report1ID = "rp000000-0000-0000-0000-000000000001" // created_by=user1 (domain member)
	report2ID = "rp000000-0000-0000-0000-000000000002" // created_by=user2 (NOT member)
	report3ID = "rp000000-0000-0000-0000-000000000003" // created_by=user1, SpiceDB parent already exists
	report4ID = "rp000000-0000-0000-0000-000000000004" // created_by=NULL → skipped

	// Report with pre-existing role.
	report5ID     = "rp000000-0000-0000-0000-000000000005"
	report5RoleID = "rpr00000-0000-0000-0000-000000000005"

	// Alarms (live in alarms DB).
	alarm1ID = "a0000000-0000-0000-0000-000000000001"
	alarm2ID = "a0000000-0000-0000-0000-000000000002"
)

type dbConfig struct {
	host, port, user, pass, name string
}

func (c dbConfig) dsn() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", c.host, c.port, c.user, c.pass, c.name)
}

func main() {
	ctx := context.Background()
	now := time.Now().UTC()

	// -----------------------------------------------------------------------
	// 1. Seed Domains DB
	// -----------------------------------------------------------------------
	log.Println("connecting to domains DB ...")
	ddb := mustConnect(domainsDB)
	defer ddb.Close()

	seedDomainTables(ctx, ddb, now)
	log.Println("domains DB seeded")

	// -----------------------------------------------------------------------
	// 2. Seed RE DB (includes domain tables + rules)
	// -----------------------------------------------------------------------
	log.Println("connecting to RE DB ...")
	rdb := mustConnect(reDB)
	defer rdb.Close()

	seedDomainTables(ctx, rdb, now)
	seedRules(ctx, rdb, now)
	log.Println("RE DB seeded")

	// -----------------------------------------------------------------------
	// 3. Seed Reports DB (includes domain tables + report_config)
	// -----------------------------------------------------------------------
	log.Println("connecting to reports DB ...")
	rpdb := mustConnect(reportsDB)
	defer rpdb.Close()

	seedDomainTables(ctx, rpdb, now)
	seedReports(ctx, rpdb, now)
	log.Println("reports DB seeded")

	// -----------------------------------------------------------------------
	// 4. Seed Alarms DB (includes domain + RE tables + alarms)
	// -----------------------------------------------------------------------
	log.Println("connecting to alarms DB ...")
	adb := mustConnect(alarmsDB)
	defer adb.Close()

	seedDomainTables(ctx, adb, now)
	seedRulesMinimal(ctx, adb, now) // alarms DB has rules tables via RE migration
	seedAlarms(ctx, adb, now)
	log.Println("alarms DB seeded")

	// -----------------------------------------------------------------------
	// 5. Optionally seed SpiceDB (parent relations for rule3 and report3)
	// -----------------------------------------------------------------------
	if seedSpiceDB {
		log.Println("connecting to SpiceDB ...")
		seedSpiceDBRelationships(ctx)
		log.Println("SpiceDB seeded")
	}

	log.Println("all seed data inserted successfully")
	printSummary()
}

// ---------------------------------------------------------------------------
// Domain tables (identical across all DBs that include domain migrations)
// ---------------------------------------------------------------------------

func seedDomainTables(ctx context.Context, db *sql.DB, now time.Time) {
	mustExec(ctx, db, `
		INSERT INTO domains (id, name, tags, metadata, route, created_at, updated_at, created_by, status)
		VALUES ($1, $2, '{}', '{}', $3, $4, $4, $5, 0)
		ON CONFLICT (id) DO NOTHING`,
		domainID, domainName, "seed-test-domain", now, user1ID)

	// Domain admin role
	mustExec(ctx, db, `
		INSERT INTO domains_roles (id, name, entity_id, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $4, $5)
		ON CONFLICT (id) DO NOTHING`,
		domainRoleID, domainRoleName, domainID, now, user1ID)

	// Domain role actions (a representative subset)
	actions := []string{
		"domain_update", "domain_read", "domain_membership",
		"domain_manage_role", "domain_add_role_users", "domain_remove_role_users", "domain_view_role_users",
		"rule_create", "rule_read", "rule_update", "rule_delete",
		"rule_manage_role", "rule_add_role_users", "rule_remove_role_users", "rule_view_role_users",
		"report_create", "report_read", "report_update", "report_delete",
		"report_manage_role", "report_add_role_users", "report_remove_role_users", "report_view_role_users",
		"alarm_update", "alarm_read", "alarm_delete", "alarm_assign", "alarm_acknowledge", "alarm_resolve",
	}
	for _, action := range actions {
		mustExec(ctx, db, `
			INSERT INTO domains_role_actions (role_id, action)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING`,
			domainRoleID, action)
	}

	// user1 is a domain member; user2 is NOT.
	mustExec(ctx, db, `
		INSERT INTO domains_role_members (role_id, member_id, entity_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		domainRoleID, user1ID, domainID)
}

// ---------------------------------------------------------------------------
// Rules (RE DB)
// ---------------------------------------------------------------------------

func seedRules(ctx context.Context, db *sql.DB, now time.Time) {
	type rule struct {
		id, name, domainID string
		createdBy          *string
	}

	u1 := strPtr(user1ID)
	u2 := strPtr(user2ID)

	rules := []rule{
		{rule1ID, "rule-1-member-creator", domainID, u1},
		{rule2ID, "rule-2-nonmember-creator", domainID, u2},
		{rule3ID, "rule-3-spicedb-exists", domainID, u1},
		{rule4ID, "rule-4-null-creator", domainID, nil},
		{rule5ID, "rule-5-no-domain", "", u1},
		{rule6ID, "rule-6-has-role-already", domainID, u1},
	}

	for _, r := range rules {
		mustExec(ctx, db, `
			INSERT INTO rules (id, name, domain_id, created_by, created_at, status, logic_type)
			VALUES ($1, $2, $3, $4, $5, 0, 0)
			ON CONFLICT (id) DO NOTHING`,
			r.id, r.name, r.domainID, r.createdBy, now)
	}

	// rule6 already has a role → should NOT appear in orphan list.
	mustExec(ctx, db, `
		INSERT INTO rules_roles (id, name, entity_id, created_at, updated_at, created_by)
		VALUES ($1, 'admin', $2, $3, $3, $4)
		ON CONFLICT (id) DO NOTHING`,
		rule6RoleID, rule6ID, now, user1ID)

	mustExec(ctx, db, `
		INSERT INTO rules_role_actions (role_id, action)
		VALUES ($1, 'rule_read')
		ON CONFLICT DO NOTHING`,
		rule6RoleID)

	mustExec(ctx, db, `
		INSERT INTO rules_role_members (role_id, member_id, entity_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		rule6RoleID, user1ID, rule6ID)
}

// seedRulesMinimal inserts the same rules into the alarms DB (which has RE
// tables) so that foreign key constraints on rule_id can be satisfied.
func seedRulesMinimal(ctx context.Context, db *sql.DB, now time.Time) {
	for _, r := range []struct{ id, name string }{
		{rule1ID, "rule-1-member-creator"},
		{rule2ID, "rule-2-nonmember-creator"},
	} {
		mustExec(ctx, db, `
			INSERT INTO rules (id, name, domain_id, created_by, created_at, status, logic_type)
			VALUES ($1, $2, $3, $4, $5, 0, 0)
			ON CONFLICT (id) DO NOTHING`,
			r.id, r.name, domainID, user1ID, now)
	}
}

// ---------------------------------------------------------------------------
// Reports
// ---------------------------------------------------------------------------

func seedReports(ctx context.Context, db *sql.DB, now time.Time) {
	type report struct {
		id, name, domainID string
		createdBy          *string
	}

	u1 := strPtr(user1ID)
	u2 := strPtr(user2ID)

	reports := []report{
		{report1ID, "report-1-member-creator", domainID, u1},
		{report2ID, "report-2-nonmember-creator", domainID, u2},
		{report3ID, "report-3-spicedb-exists", domainID, u1},
		{report4ID, "report-4-null-creator", domainID, nil},
		{report5ID, "report-5-has-role-already", domainID, u1},
	}

	for _, r := range reports {
		mustExec(ctx, db, `
			INSERT INTO report_config (id, name, domain_id, created_by, created_at, status)
			VALUES ($1, $2, $3, $4, $5, 0)
			ON CONFLICT (id) DO NOTHING`,
			r.id, r.name, r.domainID, r.createdBy, now)
	}

	// report5 already has a role.
	mustExec(ctx, db, `
		INSERT INTO reports_roles (id, name, entity_id, created_at, updated_at, created_by)
		VALUES ($1, 'admin', $2, $3, $3, $4)
		ON CONFLICT (id) DO NOTHING`,
		report5RoleID, report5ID, now, user1ID)

	mustExec(ctx, db, `
		INSERT INTO reports_role_actions (role_id, action)
		VALUES ($1, 'report_read')
		ON CONFLICT DO NOTHING`,
		report5RoleID)

	mustExec(ctx, db, `
		INSERT INTO reports_role_members (role_id, member_id, entity_id)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`,
		report5RoleID, user1ID, report5ID)
}

// ---------------------------------------------------------------------------
// Alarms
// ---------------------------------------------------------------------------

func seedAlarms(ctx context.Context, db *sql.DB, now time.Time) {
	for _, a := range []struct{ id, ruleID string }{
		{alarm1ID, rule1ID},
		{alarm2ID, rule2ID},
	} {
		mustExec(ctx, db, `
			INSERT INTO alarms (id, rule_id, domain_id, channel_id, subtopic, client_id,
				measurement, value, unit, threshold, cause, status, severity, created_at)
			VALUES ($1, $2, $3, 'ch000000-0000-0000-0000-000000000001', 'test/topic',
				'cl000000-0000-0000-0000-000000000001', 'temperature', '42.5', 'C', '40.0',
				'exceeded threshold', 0, 1, $4)
			ON CONFLICT (id) DO NOTHING`,
			a.id, a.ruleID, domainID, now)
	}
}

// ---------------------------------------------------------------------------
// SpiceDB — write a parent relation for rule3 and report3 so the
// "policy already exists" code path is exercised.
// ---------------------------------------------------------------------------

func seedSpiceDBRelationships(ctx context.Context) {
	addr := fmt.Sprintf("%s:%s", spicedbHost, spicedbPort)
	client, err := authzed.NewClientWithExperimentalAPIs(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(spicedbPreSharedKey),
	)
	if err != nil {
		log.Printf("WARNING: failed to connect to SpiceDB at %s: %v (skipping SpiceDB seed)", addr, err)
		return
	}

	// Use TOUCH so re-running is idempotent.
	updates := []*v1.RelationshipUpdate{
		{
			Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: "rule", ObjectId: rule3ID},
				Relation: "domain",
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: "domain", ObjectId: domainID}},
			},
		},
		{
			Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
			Relationship: &v1.Relationship{
				Resource: &v1.ObjectReference{ObjectType: "report", ObjectId: report3ID},
				Relation: "domain",
				Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: "domain", ObjectId: domainID}},
			},
		},
	}

	_, err = client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{Updates: updates})
	if err != nil {
		log.Printf("WARNING: failed to write SpiceDB relationships: %v", err)
		return
	}
	log.Printf("wrote %d SpiceDB relationships (TOUCH)", len(updates))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustConnect(cfg dbConfig) *sql.DB {
	db, err := sql.Open("pgx", cfg.dsn())
	if err != nil {
		log.Fatalf("failed to open %s: %v", cfg.name, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		log.Fatalf("failed to ping %s at %s:%s: %v", cfg.name, cfg.host, cfg.port, err)
	}
	cancel()
	return db
}

func mustExec(ctx context.Context, db *sql.DB, query string, args ...any) {
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		log.Fatalf("exec failed: %v\nquery: %s\nargs: %v", err, query, args)
	}
}

func strPtr(s string) *string { return &s }

func printSummary() {
	fmt.Print(`
=== SEED DATA SUMMARY ===

Domain:  ` + domainID + ` ("` + domainName + `")

Users:
  user1 (domain member): ` + user1ID + `
  user2 (NOT a member):  ` + user2ID + `

Rules (RE DB):
  ` + rule1ID + `  rule-1-member-creator       orphan, created_by=user1 → expect role WITH member
  ` + rule2ID + `  rule-2-nonmember-creator    orphan, created_by=user2 → expect role WITHOUT member
  ` + rule3ID + `  rule-3-spicedb-exists       orphan, created_by=user1, SpiceDB parent pre-seeded → test policyExists
  ` + rule4ID + `  rule-4-null-creator         orphan, created_by=NULL  → expect SKIPPED
  ` + rule5ID + `  rule-5-no-domain            orphan, domain_id=""     → expect SKIPPED
  ` + rule6ID + `  rule-6-has-role-already     HAS role entry           → should NOT appear in orphan list

Reports (Reports DB):
  ` + report1ID + `  report-1-member-creator     orphan, created_by=user1 → expect role WITH member
  ` + report2ID + `  report-2-nonmember-creator  orphan, created_by=user2 → expect role WITHOUT member
  ` + report3ID + `  report-3-spicedb-exists     orphan, created_by=user1, SpiceDB parent pre-seeded → test policyExists
  ` + report4ID + `  report-4-null-creator       orphan, created_by=NULL  → expect SKIPPED
  ` + report5ID + `  report-5-has-role-already   HAS role entry           → should NOT appear in orphan list

Alarms (Alarms DB):
  ` + alarm1ID + `  alarm-1 (rule1)
  ` + alarm2ID + `  alarm-2 (rule2)

SpiceDB:
  rule:` + rule3ID + `#domain@domain:` + domainID + ` (pre-seeded)
  report:` + report3ID + `#domain@domain:` + domainID + ` (pre-seeded)
`)
}
