# atom-migration

Offline, idempotent migrator: Magistrala v0.30.0 (per-service Postgres) → Atom IAM
(single Postgres). See [PLAN.md](./PLAN.md) for the full mapping and runbook.

## Build

Plain binary:

```bash
go build ./tools/atom-migration/
```

Or a Docker image, so you can run the tool repeatedly without `go run` and
without a Go toolchain on the host (build context is the repo root):

```bash
docker build -f tools/atom-migration/Dockerfile -t magistrala/atom-migration:dev .
```

## Start only the source databases

The migrator reads Postgres directly — it does **not** need the Magistrala app
services running. To migrate from restored volumes, start just the six source DB
containers (`--no-deps` keeps compose from pulling in the app services they
depend on):

```bash
docker compose -f docker/docker-compose.yaml up -d --no-deps \
  auth-db users-db domains-db clients-db channels-db groups-db
```

They mount the `magistrala_magistrala-<svc>-db-volume` volumes and attach to
`magistrala-base-net`, where the migrator resolves them by service name.

## Run (dry-run is default — writes nothing)

The migrator needs to reach every source DB **and** the Atom DB. Atom must already
have its schema applied (run Atom once, or apply `migrations/001_initial.sql`).

Run the image on the compose network. Mount the repo at `/work` so the default
`--env docker/.env` and `--report-dir` resolve, and reach source DBs by their
compose service names:

If Atom runs as its own compose project (its containers live on `atom_default`),
that network is isolated from `magistrala-base-net`, so the migrator cannot reach
the Atom DB yet. Bridge the Atom Postgres onto the migrator's network once, then
address it by its container name `atom-postgres-1`:

```bash
docker network connect magistrala-base-net atom-postgres-1
```

```bash
docker run --rm --network magistrala-base-net \
  --user "$(id -u):$(id -g)" \
  -v "$PWD":/work -w /work \
  magistrala/atom-migration:dev \
    --env docker/.env \
    --atom-dsn 'host=atom-postgres-1 port=5432 user=atom password=atom dbname=atom sslmode=disable'
```

`--user` makes the container write the report as your host user; without it the
image's non-root user cannot create `report/` in the bind-mounted repo.

The `network connect` is **not persistent** — re-run it after any
`docker compose down`/recreate of the Atom stack (the bridge silently drops and
`postgres` stops resolving). For a permanent setup, declare `magistrala-base-net`
as an external network in Atom's compose instead.

Add `--apply` / `--verify` as described below.

### As a one-shot container on the compose network (recommended)

```bash
go run ./tools/atom-migration \
  --env docker/.env --from-host \
  --atom-dsn 'host=127.0.0.1 port=5432 user=atom password=atom dbname=atom sslmode=disable'
```

## Apply

```bash
go run ./tools/atom-migration ... --apply
```

A pre-flight runs first (read-only). Same checks run in dry-run for the report.

Magistrala dropped several uniqueness constraints that Atom still enforces
(`tenants.name`; device and group `name` per tenant; tenant/entity/resource
`alias`). The migrator resolves these automatically: in each collision the
oldest row keeps its value and the rest get a deterministic, id-derived suffix
(`name (a1b2c3d4)` / `alias-a1b2c3d4`). Renames are reported as `renamed.*`
counts plus a warning per row, and are stable across re-runs. So these no longer
block the apply — only issues the tool cannot safely auto-fix do (e.g. duplicate
user emails, which are identities).

## Verify (after apply)

```bash
go run ./tools/atom-migration ... --verify
```

Read-only reconciliation: every source row that should have migrated must exist in
Atom (tenants, entities, resources, object_groups) and every device→channel
connection must have a matching authz edge. Missing rows are reported as blocking.

## Flags

| flag                | default                       | meaning                                              |
| ------------------- | ----------------------------- | ---------------------------------------------------- |
| `--env`             | `docker/.env`                 | Magistrala env file (reads `MG_*_DB_*`)              |
| `--atom-dsn`        | compose default               | Atom Postgres DSN (or `ATOM_DATABASE_URL`)           |
| `--from-host`       | false                         | rewrite source hosts to 127.0.0.1 (use mapped ports) |
| `--apply`           | false                         | perform the load (omit = dry-run)                    |
| `--report-dir`      | `tools/atom-migration/report` | JSON+markdown report output                          |
| `--unmapped-action` | `manage`                      | fallback for unmapped MG actions: `manage` or `skip` |

## Credentials are re-issued, not carried

Atom authenticates API keys by a credential UUID embedded in the key
(`atom_<32hex>_<64hex>`, argon2 over the raw 32-byte secret — see Atom
`src/auth.rs`). Magistrala secrets fit neither the format nor the lookup, so:

- **Device keys** are re-issued. On `--apply` the migrator writes
  `report/device-keys-<stamp>.csv` (`client_id,domain_id,identity,api_key`, mode
  0600). Re-provision devices/bootstrap configs from it, then delete it — the
  plaintext secret is shown only once.
- **User passwords** (bcrypt → argon2 unconvertible): users land with no password
  credential. Report's `password_reset` TODO lists every user for the email reset.
- **PAT secrets** (hashed + format): metadata migrates, secret must be re-issued.
  Report's `pat_reissue` TODO.
- Transient data not migrated: OTP verifications, short-lived auth `keys`, login
  attempts.

## Idempotency

Every write upserts on a preserved/derived primary key (`ON CONFLICT DO NOTHING`),
so the tool is safe to re-run. Derived UUIDs (roles, permission blocks, policies)
use uuidv5 so they are stable across runs.
