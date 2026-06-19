# Magistrala v0.30.0 → Atom IAM Migration Plan

Offline, one-shot migration that reads the per-service Magistrala Postgres
databases (default Docker Compose deployment) and writes a single Atom Postgres
database. Implemented as Go scripts.

---

## 1. Decisions (locked)

| Topic | Decision |
|-------|----------|
| Passwords | **Force reset.** Users migrate with no `password` credential; they reset via Atom's email flow on first login. (bcrypt → argon2 is not convertible without plaintext.) |
| Scope | Core IAM + Roles & policies + Connections + PATs. |
| Execution | **Offline one-shot.** Stop Magistrala app services, snapshot, transform, load, start Atom. |
| IDs | **Preserve Magistrala UUIDs** as Atom UUIDs (PKs and FKs). Magistrala IDs are 36-char UUID strings — directly usable as Atom `UUID` PKs. Keeps audit trails, message payloads, external references, and SpiceDB-derived links intact. |

---

## 2. Source vs target topology

### Source (Magistrala, default compose — separate DB container per service)

All `magistrala/magistrala`, port 5432, on network `magistrala-base-net`:

| Container | DB name | Tables we read |
|-----------|---------|----------------|
| `domains-db` | `domains` | `domains`, `invitations`, `domains_roles`, `domains_role_actions`, `domains_role_members` |
| `users-db` | `users` | `users`, `users_verifications` |
| `clients-db` | `clients` | `clients`, `connections`, `clients_roles*`, plus its embedded `groups` copy |
| `channels-db` | `channels` | `channels`, `connections`, `channels_roles*` |
| `groups-db` | `groups` | `groups`, `groups_roles*` |
| `auth-db` | `auth` | `pats`, `pat_scopes` (skip `keys` — short-lived JWTs; skip legacy `policies`/`domains` mirror) |

> Note: `groups` migrations are embedded into clients **and** channels **and** the
> standalone groups service. The authoritative groups data for default compose is
> the `groups-db`/`groups` database — read groups from there.

### Target (Atom — single Postgres)

`atom` DB, schema from `migrations/001..005`. Relevant tables:
`tenants, entities, credentials, entity_emails, resources, principal_groups,
object_groups, *_group_hierarchy, object_group_entities, object_group_resources,
tenant_memberships, roles, permission_blocks, permission_block_actions,
role_permission_blocks, role_assignments, direct_policies, profiles`.

Seeded fixed UUIDs already present in Atom (do **not** collide):
`...0001` admin entity, `...0002` atom-admin role, `...0003` mg-service entity,
`...0004` mg-service role, `...0005` authenticated-users group,
`...0006` domain-creator role, `...0007/8/9` permission blocks.

---

## 3. Entity mapping

### 3.1 domains → tenants
| Magistrala `domains` | Atom `tenants` |
|---|---|
| `id` | `id` (preserved) |
| `name` | `name` |
| `route` | `alias` (must be slug: lowercase, `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`, **not** UUID-shaped — see §6) |
| `tags` | `tags` |
| `metadata` | `attributes` |
| `status` 0/1 | `status` `active`/`inactive` |
| `created_at/by`, `updated_at/by` | same (created_by/updated_by only if the user id also migrates) |

### 3.2 users → entities (kind=`human`)
| Magistrala `users` | Atom |
|---|---|
| `id` | `entities.id` (preserved), `kind='human'`, `tenant_id=NULL` (users are global; domain membership handled in §4), `profile` = seeded `user` profile |
| `first_name,last_name,username,profile_picture,auth_provider` | `entities.attributes` JSON |
| `metadata` | merged into `attributes` |
| `email` | `entity_emails.email`; `verified_at` → `entity_emails.verified_at` |
| `secret` (bcrypt) | **dropped** — no credential created (force-reset) |
| `status` 0/1 | `entities.status` `active`/`inactive` |
| `role` (0 user / 1 admin) | if admin → also assign Atom `atom-admin` role (`role_assignments`) |

`users_verifications` → not migrated (transient OTPs).

### 3.3 clients (things/devices) → entities (kind=`device`)
| Magistrala `clients` | Atom |
|---|---|
| `id` | `entities.id` (preserved), `kind='device'`, `tenant_id=domain_id`, `profile`=`client` |
| `name` | `entities.name` (per-tenant unique — handle collisions, §6) |
| `tags,metadata,private_metadata` | `attributes` |
| `identity` | `attributes.identity` (and/or `entities.alias` if slug-valid) |
| `secret` (plaintext) | `credentials` row: `kind='api_key'`, `entity_id=client.id`, `secret_hash`=argon2(secret), `identifier`=client.id, `status` per client status. **See §5 — key-format caveat.** |
| `status` 0/1 | `entities.status` |
| `parent_group_id` | `object_group_entities` membership (§3.5) |

### 3.4 channels → resources (kind=`channel`)
| Magistrala `channels` | Atom `resources` |
|---|---|
| `id` | `id` (preserved), `kind='channel'`, `tenant_id=domain_id` |
| `name` | `name` |
| `route` | `alias` (slug rules, §6) |
| `tags,metadata` | `attributes` |
| `created_by` | `owner_id` (if the user migrated) |
| `parent_group_id` | `object_group_resources` membership |

### 3.5 groups → object_groups
Magistrala groups organize clients/channels within a domain (hierarchical,
`parent_id` + `path` ltree). Map to **object_groups**:
| Magistrala `groups` | Atom `object_groups` |
|---|---|
| `id,name,description,metadata,status` | same fields (`metadata`→`attributes`, status mapped) |
| `domain_id` | `tenant_id` |
| `parent_id` | `object_group_hierarchy(parent_id, child_id, tenant_id)` |
Client/channel `parent_group_id` → `object_group_entities` / `object_group_resources`.

> If any Magistrala groups are used purely as *principal* groupings (users), they'd
> map to `principal_groups` instead. Default Magistrala groups are object groupings,
> so use `object_groups`. Flag mixed cases in the dry-run report.

---

## 4. Roles, policies, memberships

Magistrala authz = per-service role tables (`<prefix>_roles`, `_role_actions`,
`_role_members`) enforced via SpiceDB. Atom replaces SpiceDB; we reconstruct authz
**from the SQL role tables** (no need to read SpiceDB directly).

Magistrala role shape: each role row is bound to a specific object instance
(`entity_id` = the domain/client/channel/group id), has a set of action strings,
and a set of member ids (users).

Mapping per Magistrala role row → Atom:
1. **`roles`** row (preserve `id`, `name`, `tenant_id` = owning domain when known).
2. **`permission_blocks`** row scoped to the object:
   - domain roles → `scope_mode='tenant'`, `tenant_id=entity_id`
   - client/channel/group roles → `scope_mode='object'`, `object_id=entity_id`
   - `effect='allow'`, `conditions='{}'`
3. **`permission_block_actions`** ← map each Magistrala action string to Atom actions
   via a translation table (below).
4. **`role_permission_blocks`** links role → block.
5. **`role_assignments`** one per `_role_member` (`subject_kind='entity'`,
   `subject_id=member_id`, `role_id`, `tenant_id`).
6. For **domain** role members, also insert **`tenant_memberships`**
   (`tenant_id`, `entity_id`, `status='active'`).

### Action translation table (Magistrala → Atom action names)
Atom actions: `read, create, write, delete, revoke, rotate, publish, subscribe,
execute, manage, policy.manage, role.manage, authz.check`.

| Magistrala action (examples) | Atom action |
|---|---|
| `read`, `view`, `*_read`, `*_view_*` | `read` |
| `create`, `*_create*` | `create` |
| `update`, `*_update` | `write` |
| `delete`, `*_delete*` | `delete` |
| `publish` | `publish` |
| `subscribe` | `subscribe` |
| `admin`, `manage`, `*_manage_role` | `manage` |
| `*_add_role_users`, `*_remove_role_users`, `membership*` | `policy.manage` |
| `*_view_role_users` | `read` |
| (unmapped) | log + default to `manage`, or skip — configurable |

> Magistrala's full action vocabulary is generated from the SpiceDB schema
> (`docker/spicedb/schema.zed`). The migrator ships a complete map derived from
> that schema; anything missing is reported in the dry-run, never silently dropped.

### Invitations
`domains.invitations` → `tenant_invitations` (preserve domain_id, invitee_user_id,
invited_by, role_id; map confirmed/rejected timestamps). Pending only; accepted
ones are already reflected as memberships.

---

## 5. Credentials (PATs + device secrets)

### Device secrets (clients.secret)
Plaintext in Magistrala (looked up `WHERE secret = ...`). Stored in Atom as
`credentials(kind='api_key', secret_hash=argon2(secret))`.

**Caveat — key format:** Atom's native API-key format is
`atom_<credId>_<secret>` with O(1) lookup by embedded id. A raw Magistrala secret
has no envelope. Two sub-options (pick during implementation, default = A):
- **A. Carry raw secret as-is** into `secret_hash`=argon2(secret), `identifier`=
  client id. Requires Atom's device-auth path to accept a bare secret and look up
  by identifier. Confirm Atom supports this lookup mode; if not, choose B.
- **B. Re-issue keys.** Generate new `atom_...` keys; export an id→newkey CSV so
  devices/bootstrap configs can be reflashed. Higher operational cost.

This is the main Atom-side compatibility question to confirm before running.

### PATs (auth.pats + pat_scopes)
`pats.secret` is hashed (Magistrala PAT format), so plaintext isn't recoverable.
- Migrate metadata as `credentials(kind='api_key', entity_id=user_id,
  identifier=pat.id, metadata={name,description,scopes,expires_at,...},
  status=revoked?‘revoked’:‘active’, expires_at)`.
- Because the secret can't be verified by Atom argon2, **mark migrated PATs as
  needing re-issue** (same class of problem as passwords). Report them; do not
  fabricate a usable secret. `pat_scopes` preserved in credential `metadata` for
  reference / future policy reconstruction.

---

## 6. Data-quality guardrails (run as pre-flight, fail loud)

Atom migrations 004/005 enforce alias rules. Pre-flight the source data:
1. **Tenant alias** (domain.route): lowercase-fold; must match slug regex and **not**
   be UUID-shaped; globally unique case-insensitively. Offending rows → report,
   require operator fix or null the alias.
2. **Entity/resource alias** (client.identity / channel.route): same slug rule,
   unique per tenant. On violation, drop the alias (keep UUID) rather than abort.
3. **Per-tenant name uniqueness**: `entities(name, tenant_id)` and
   `resources` names — Magistrala already enforces `(domain_id, name)` so this is
   usually safe; still verify (users have no tenant, so global human-name dupes are
   fine because humans are keyed by id/email).
4. **FK integrity**: skip rows whose `domain_id` has no surviving tenant; report.
5. **Email uniqueness**: `entity_emails.email` is globally UNIQUE — dedupe/report
   conflicting user emails before load.

---

## 7. Load ordering (FK-safe)

1. tenants
2. entities (human, device) — without created_by/updated_by FKs first…
3. …then backfill tenants.created_by/updated_by and resources.owner_id
4. entity_emails
5. credentials (device api_key; PAT metadata)
6. resources (channels)
7. object_groups → object_group_hierarchy → object_group_entities/resources
8. roles → permission_blocks → permission_block_actions → role_permission_blocks
9. role_assignments, direct_policies
10. tenant_memberships
11. tenant_invitations

All inserts use deterministic PKs (preserved IDs) + `ON CONFLICT DO NOTHING`/upsert
so the migration is **idempotent / re-runnable**.

---

## 8. Go program design

```
tools/atom-migration/
  main.go            // flags: --dry-run (default), --apply, --report-dir
  config.go          // reads docker/.env for DB hosts/ports/creds/names
  source/            // one reader per source DB (sqlx/pgx), returns typed structs
    domains.go users.go clients.go channels.go groups.go auth.go
  transform/         // pure funcs: MG structs -> Atom rows; status/alias/action maps
  target/            // Atom writer: ordered, transactional, ON CONFLICT upserts
  preflight/         // §6 guardrails -> report
  report/            // JSON+markdown summary: counts, skips, conflicts, todo lists
```

- **Connectivity:** run the migrator as a one-shot container on
  `magistrala-base-net` (compose `run --rm`) so it resolves `domains-db`,
  `users-db`, … and the Atom `postgres` by service name. Alternatively expose host
  ports and run from host. DB creds/names pulled from `docker/.env` keys
  (`MG_*_DB_*`) and Atom's `.env` (`POSTGRES_*`).
- **Idempotent & resumable:** every write upserts on preserved PK; safe to re-run.
- **Dry-run first:** default mode reads + transforms + validates + writes a report,
  touches nothing. `--apply` runs the load in one transaction per phase.
- **argon2** for device secrets via `golang.org/x/crypto/argon2` matching Atom's
  params (confirm Atom's argon2 config: variant/m/t/p) so hashes verify.

---

## 9. Cutover runbook (offline)

1. `docker compose stop` Magistrala **app** services (users, clients, channels,
   groups, domains, auth, adapters) — keep the `*-db` containers running.
2. Backup: `pg_dump` each source DB.
3. Start Atom Postgres; let Atom run once to apply `migrations/001..005`
   (or run migrations standalone), then stop Atom app.
4. Run migrator `--dry-run`; review report; fix guardrail violations (§6).
5. Run migrator `--apply`.
6. Start Atom app; verify (§10).
7. Point remaining Magistrala services (messaging/bootstrap/certs/readers) at Atom
   for authn/authz; decommission SpiceDB + users/clients/channels/groups/domains/auth.

---

## 10. Verification

- Row-count reconciliation per mapping (source vs target), in the report.
- Spot auth checks via Atom `POST /authz/check` for a sample of
  (user, domain, action) and (device, channel, publish) that were allowed pre-migration.
- Admin login (seeded atom-admin) works; a migrated user can complete password reset.
- A migrated device authenticates (validates §5 key-format decision).
- Tenant/entity/resource counts and alias uniqueness hold.

---

## 11. Open items to confirm before build

1. **Device key format** (§5): does Atom's device-auth accept a bare secret looked
   up by `identifier`, or must keys be re-issued in `atom_<id>_<secret>` form?
2. **argon2 params** Atom expects (so migrator-produced hashes verify).
3. **Groups semantics** (§3.5): confirm all Magistrala groups are object groupings
   (→ object_groups) vs any principal groupings.
4. **PAT re-issue** acceptance (§5) — same plaintext limitation as passwords.
5. Whether `created_by`/`owner_id` referencing **non-migrated/system** users should
   be nulled or pointed at the seeded admin entity.
