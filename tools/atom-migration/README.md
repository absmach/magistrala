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
