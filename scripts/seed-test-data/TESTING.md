# Backfill Roles — Testing Guide

This document covers end-to-end testing of the migration scripts on the `migrations` branch:

| Script | Purpose |
|--------|---------|
| `scripts/re-backfill-roles/` | Backfills missing built-in admin roles for **rules** (RE service) |
| `scripts/reports-backfill-roles/` | Backfills missing built-in admin roles for **reports** |
| `scripts/seed-test-data/` | Seeds all required test data across databases and SpiceDB |
| `domains/postgres/init.go` | Migration adding `alarm_*` and `report_*` actions to the domain admin role |

---

## Prerequisites

### Infrastructure

Start the required containers:

```bash
cd docker
docker compose up -d \
    spicedb-db spicedb-migrate spicedb \
    auth-db auth \
    domains-db domains \
    re-db re \
    reports-db reports \
    alarms-db alarms
```

Wait until all services are healthy. Each service applies its own Postgres migrations on startup, creating the required schemas. The `auth` service is required because it writes the SpiceDB schema on startup — without it, the seed script and backfill scripts fail with `object definition not found`.

### Connection Details (docker-compose defaults)

| Service | Host | Port | User | Password | Database |
|---------|------|------|------|----------|----------|
| Domains DB | localhost | 6003 | magistrala | magistrala | domains |
| RE DB | localhost | 6009 | magistrala | magistrala | rules_engine |
| Reports DB | localhost | 6020 | magistrala | magistrala | reports |
| Alarms DB | localhost | 6019 | magistrala | magistrala | alarms |
| SpiceDB gRPC | localhost | 50051 | — | 12345678 (pre-shared key) | — |

### Fix Hard-Coded Configs (if needed)

The backfill scripts have hard-coded database configs. Before running, verify they match your environment:

**`scripts/re-backfill-roles/main.go` (lines 48–55):**

```go
dbConfig = pgclient.Config{
    Host:    "localhost",
    Port:    "6009",        // docker-compose: 6009 (NOT 15432)
    User:    "magistrala",  // docker-compose: magistrala (NOT postgres)
    Pass:    "magistrala",  // docker-compose: magistrala (NOT supermq)
    Name:    "rules_engine",
    SSLMode: "disable",
}
```

**`scripts/reports-backfill-roles/main.go` (lines 49–56):**

```go
dbConfig = pgclient.Config{
    Host:    "localhost",
    Port:    "6020",        // docker-compose: 6020 (NOT 15432)
    User:    "magistrala",  // docker-compose: magistrala (NOT postgres)
    Pass:    "magistrala",  // docker-compose: magistrala (NOT supermq)
    Name:    "reports",
    SSLMode: "disable",
}
```

**Both scripts — SpiceDB schema file (line 47):**

```go
spicedbSchemaFile = "docker/spicedb/schema.zed"  // NOT combined-schema.zed
```

---

## Step 1 — Seed Test Data

```bash
go run ./scripts/seed-test-data/
```

This inserts deterministic test data across all four databases and SpiceDB. It is idempotent (uses `ON CONFLICT DO NOTHING`), so re-running is safe.

### What Gets Created

**Domain:**

| ID | Name |
|----|------|
| `d0000000-0000-0000-0000-000000000001` | seed-test-domain |

**Users:**

| ID | Domain Membership |
|----|-------------------|
| `u0000000-0000-0000-0000-000000000001` (user1) | Member of domain (in `domains_role_members`) |
| `u0000000-0000-0000-0000-000000000002` (user2) | NOT a domain member |

**Rules (RE DB) — 6 rules, 4 orphans:**

| Rule ID | Name | Scenario |
|---------|------|----------|
| `r0000000-...-000000000001` | rule-1-member-creator | Orphan. `created_by=user1` (domain member). Backfill should create role **with** member. |
| `r0000000-...-000000000002` | rule-2-nonmember-creator | Orphan. `created_by=user2` (NOT member). Backfill should create role **without** member. |
| `r0000000-...-000000000003` | rule-3-spicedb-exists | Orphan. `created_by=user1`. SpiceDB parent relation **pre-seeded**. Tests `policyExists` check. |
| `r0000000-...-000000000004` | rule-4-null-creator | Orphan. `created_by=NULL`. Should be **skipped**. |
| `r0000000-...-000000000005` | rule-5-no-domain | Orphan. `domain_id=""`. Should be **skipped**. |
| `r0000000-...-000000000006` | rule-6-has-role-already | Has `rules_roles` entry. Should **NOT appear** in orphan list. |

**Reports (Reports DB) — 5 reports, 3 orphans:**

| Report ID | Name | Scenario |
|-----------|------|----------|
| `rp000000-...-000000000001` | report-1-member-creator | Orphan. `created_by=user1`. Backfill should create role **with** member. |
| `rp000000-...-000000000002` | report-2-nonmember-creator | Orphan. `created_by=user2`. Backfill should create role **without** member. |
| `rp000000-...-000000000003` | report-3-spicedb-exists | Orphan. `created_by=user1`. SpiceDB parent **pre-seeded**. Tests `policyExists`. |
| `rp000000-...-000000000004` | report-4-null-creator | Orphan. `created_by=NULL`. Should be **skipped**. |
| `rp000000-...-000000000005` | report-5-has-role-already | Has `reports_roles` entry. Should **NOT appear**. |

**Alarms (Alarms DB) — 2 alarms:**

| Alarm ID | Linked Rule |
|----------|-------------|
| `a0000000-...-000000000001` | rule-1 |
| `a0000000-...-000000000002` | rule-2 |

**SpiceDB (pre-seeded parent relations):**

```
rule:r0000000-...-000000000003#domain@domain:d0000000-...-000000000001
report:rp000000-...-000000000003#domain@domain:d0000000-...-000000000001
```

---

## Step 2 — Verify Seed Data

### Check orphan rules in RE DB

```bash
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT r.id, r.name, r.domain_id, r.created_by
  FROM rules r
  WHERE NOT EXISTS (SELECT 1 FROM rules_roles rr WHERE rr.entity_id = r.id)
  ORDER BY r.name;"
```

**Expected:** 5 rows (rule-1 through rule-5). **rule-6 should NOT appear** (it has a role).

### Check orphan reports in Reports DB

```bash
psql -h localhost -p 6020 -U magistrala -d reports -c "
  SELECT rc.id, rc.name, rc.domain_id, rc.created_by
  FROM report_config rc
  WHERE NOT EXISTS (SELECT 1 FROM reports_roles rr WHERE rr.entity_id = rc.id)
  ORDER BY rc.name;"
```

**Expected:** 4 rows (report-1 through report-4). **report-5 should NOT appear**.

### Check domain membership

```bash
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT * FROM domains_role_members
  WHERE entity_id = 'd0000000-0000-0000-0000-000000000001';"
```

**Expected:** 1 row for `user1`. `user2` should NOT be present.

### Check SpiceDB pre-seeded relationships

```bash
zed relationship read rule \
  --insecure --endpoint localhost:50051 --token 12345678
```

**Expected:** At least one relationship for `rule:r0000000-...-000000000003#domain@domain:d0000000-...-000000000001`.

---

## Step 3 — Test RE Backfill (Dry Run)

Set `dryRun = true` in `scripts/re-backfill-roles/main.go`, then:

```bash
go run ./scripts/re-backfill-roles/
```

### Expected Log Output

| Rule | Expected Log |
|------|-------------|
| rule-1 | `"dry run: would provision missing built-in role"` with `created_by_exists_in_domain=true` |
| rule-2 | `"created_by user is not a member of the domain"` + `"dry run: would provision missing built-in role without member"` |
| rule-3 | `"dry run: spicedb policy already exists, will not be re-added"` + `"dry run: would provision"` with `new_optional_policies=0, existing_optional_policies=1` |
| rule-4 | `"skipping rule without created_by and no default member override"` |
| rule-5 | `"skipping rule without domain_id"` |
| rule-6 | Does NOT appear at all |

### Verify No Side Effects

```bash
# Postgres: no new roles created
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT COUNT(*) FROM rules_roles
  WHERE entity_id IN (
    'r0000000-0000-0000-0000-000000000001',
    'r0000000-0000-0000-0000-000000000002',
    'r0000000-0000-0000-0000-000000000003'
  );"
```

**Expected:** `0` (dry run should not write anything).

---

## Step 4 — Test RE Backfill (Real Run)

Set `dryRun = false` in `scripts/re-backfill-roles/main.go`, then:

```bash
go run ./scripts/re-backfill-roles/
```

### Expected Log Output

| Rule | Expected Log |
|------|-------------|
| rule-1 | `"provisioned missing built-in role"` with `member_added=true` |
| rule-2 | `"provisioned missing built-in role"` with `member_added=false` |
| rule-3 | `"spicedb policy already exists, skipping re-add"` + `"provisioned missing built-in role"` with `new_optional_policies=0` |
| rule-4 | `"skipping rule without created_by"` |
| rule-5 | `"skipping rule without domain_id"` |

Final summary should show: `processed=3, skipped=2, failed=0`.

### Verify in Postgres

```bash
# New roles exist for rules 1, 2, 3
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT rr.id, rr.entity_id, rr.name, rr.created_by
  FROM rules_roles rr
  ORDER BY rr.entity_id;"

# Role members: rule-1 should have user1; rule-2 and rule-3 check based on domain membership
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT rrm.role_id, rrm.member_id, rrm.entity_id
  FROM rules_role_members rrm
  ORDER BY rrm.entity_id;"
```

### Verify in SpiceDB

```bash
zed relationship read rule \
  --insecure --endpoint localhost:50051 --token 12345678
```

**Expected:** Parent relations for rule-1 and rule-2 are newly created. Rule-3 already had one (no duplicate).

---

## Step 5 — Test Idempotency (Re-Run)

Run the same backfill again without any changes:

```bash
go run ./scripts/re-backfill-roles/
```

**Expected:** `"loaded rules without roles" count=2` and `"backfill finished" processed=0, skipped=2, failed=0`. The two remaining rows are rule-4 (`created_by=NULL`) and rule-5 (no `domain_id`), which always re-appear in the orphan query and are skipped each run. The idempotency signal is `processed=0` — no roles or SpiceDB writes are duplicated.

---

## Step 6 — Test Partial State (policyExists Path)

This specifically validates the SpiceDB pre-check. Simulate a scenario where Postgres lost the role but SpiceDB still has the parent relation:

```bash
# Delete just the Postgres role for rule-1
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  DELETE FROM rules_roles
  WHERE entity_id = 'r0000000-0000-0000-0000-000000000001';"

# Re-run backfill
go run ./scripts/re-backfill-roles/
```

### Expected

- `"loaded rules without roles" count=1` (only rule-1 reappears)
- `"spicedb policy already exists, skipping re-add"` — the parent relation is detected and filtered out
- `"provisioned missing built-in role"` with `new_optional_policies=0, existing_optional_policies=1`
- The role row is re-created in Postgres **without** a duplicate SpiceDB write

### Verify

```bash
# Postgres: role restored
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  SELECT * FROM rules_roles
  WHERE entity_id = 'r0000000-0000-0000-0000-000000000001';"

# SpiceDB: still exactly one parent relation (no duplicate)
zed relationship read rule:r0000000-0000-0000-0000-000000000001 \
  --insecure --endpoint localhost:50051 --token 12345678
```

---

## Step 7 — Test Reports Backfill

Repeat Steps 3–6 for the reports backfill script:

```bash
# Dry run (set dryRun = true first)
go run ./scripts/reports-backfill-roles/

# Real run (set dryRun = false)
go run ./scripts/reports-backfill-roles/
```

### Expected behavior

| Report | Expected |
|--------|----------|
| report-1 | Role provisioned **with** member (user1 is domain member) |
| report-2 | Role provisioned **without** member (user2 not in domain) |
| report-3 | `"spicedb policy already exists"` + role provisioned with `new_optional_policies=0` |
| report-4 | Skipped (NULL `created_by`) |
| report-5 | Does not appear (already has role) |

### Verify

```bash
psql -h localhost -p 6020 -U magistrala -d reports -c "
  SELECT rr.id, rr.entity_id, rr.name
  FROM reports_roles rr
  ORDER BY rr.entity_id;"

zed relationship read report \
  --insecure --endpoint localhost:50051 --token 12345678
```

---

## Step 8 — Verify Domains Migration

The `domains/postgres/init.go` change adds `alarm_*` and `report_*` actions to the domain admin role. This is applied by the domains service on startup.

```bash
psql -h localhost -p 6003 -U magistrala -d domains -c "
  SELECT action FROM domains_role_actions
  WHERE role_id IN (SELECT id FROM domains_roles WHERE name = 'admin')
  ORDER BY action;"
```

**Expected:** The result should include all of these new actions:

```
alarm_acknowledge
alarm_assign
alarm_delete
alarm_read
alarm_resolve
alarm_update
report_add_role_users
report_create
report_delete
report_manage_role
report_read
report_remove_role_users
report_update
report_view_role_users
```

---

## Step 9 — Cleanup (Optional)

To reset and re-test from scratch:

```bash
# Remove all seeded data from RE DB
psql -h localhost -p 6009 -U magistrala -d rules_engine -c "
  DELETE FROM rules WHERE id LIKE 'r0000000-%';
  DELETE FROM domains WHERE id = 'd0000000-0000-0000-0000-000000000001';"

# Remove all seeded data from Reports DB
psql -h localhost -p 6020 -U magistrala -d reports -c "
  DELETE FROM report_config WHERE id LIKE 'rp000000-%';
  DELETE FROM domains WHERE id = 'd0000000-0000-0000-0000-000000000001';"

# Remove all seeded data from Alarms DB
psql -h localhost -p 6019 -U magistrala -d alarms -c "
  DELETE FROM alarms WHERE id LIKE 'a0000000-%';
  DELETE FROM domains WHERE id = 'd0000000-0000-0000-0000-000000000001';"

# Remove SpiceDB relationships
zed relationship delete rule --insecure --endpoint localhost:50051 --token 12345678
zed relationship delete report --insecure --endpoint localhost:50051 --token 12345678
```

Then re-run `go run ./scripts/seed-test-data/` to start fresh.

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `failed to connect to postgres` | Verify containers are running: `docker compose ps`. Check ports with `docker compose port re-db 5432`. |
| `failed to read spicedb relationships` | Ensure SpiceDB is running and schema is loaded. Check: `zed schema read --insecure --endpoint localhost:50051 --token 12345678` |
| `failed to load built-in role actions` | Verify `spicedbSchemaFile` points to `docker/spicedb/schema.zed` (not `combined-schema.zed`). |
| `no such table` errors during seed | Services haven't run yet to apply migrations. Start the full service (`re`, `reports`, `alarms`) at least once. |
| Script exits with `count=0` unexpectedly | All rules/reports already have roles. Check with the orphan queries from Step 2. |
