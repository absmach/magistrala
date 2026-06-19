# atom-migration

Offline, idempotent migrator: Magistrala v0.30.0 (per-service Postgres) → Atom IAM
(single Postgres). See [PLAN.md](./PLAN.md) for the full mapping and runbook.

## Build

```bash
go build ./tools/atom-migration/
```

## Run (dry-run is default — writes nothing)

The migrator needs to reach every source DB **and** the Atom DB. Atom must already
have its schema applied (run Atom once, or apply `migrations/001..005`).

### Option A — as a one-shot container on the compose network (recommended)

```bash
docker run --rm --network magistrala-base-net \
  -v "$PWD":/src -w /src golang:1.26 \
  go run ./tools/atom-migration \
    --env docker/.env \
    --atom-dsn 'host=postgres port=5432 user=atom password=atom dbname=atom sslmode=disable'
```

### Option B — from host (expose source + atom ports first)

```bash
go run ./tools/atom-migration \
  --env docker/.env --from-host \
  --atom-dsn 'host=127.0.0.1 port=5432 user=atom password=atom dbname=atom sslmode=disable'
```

## Apply

```bash
go run ./tools/atom-migration ... --apply
```

## Flags

| flag | default | meaning |
|------|---------|---------|
| `--env` | `docker/.env` | Magistrala env file (reads `MG_*_DB_*`) |
| `--atom-dsn` | compose default | Atom Postgres DSN (or `ATOM_DATABASE_URL`) |
| `--from-host` | false | rewrite source hosts to 127.0.0.1 (use mapped ports) |
| `--apply` | false | perform the load (omit = dry-run) |
| `--report-dir` | `tools/atom-migration/report` | JSON+markdown report output |
| `--unmapped-action` | `manage` | fallback for unmapped MG actions: `manage` or `skip` |

## What it does NOT migrate

- **User passwords** (bcrypt → argon2 unconvertible): users land with no password
  credential. Report lists every user for the email-based reset flow.
- **PAT secrets** (hashed, unrecoverable): PAT metadata migrates; secrets must be
  re-issued. Listed in the report's `pat_reissue` TODO.
- Transient data: OTP verifications, short-lived auth `keys`, login attempts.

## Idempotency

Every write upserts on a preserved/derived primary key (`ON CONFLICT DO NOTHING`),
so the tool is safe to re-run. Derived UUIDs (roles, permission blocks, policies)
use uuidv5 so they are stable across runs.
