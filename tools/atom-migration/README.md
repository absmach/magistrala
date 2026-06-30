# atom-migration

Offline, idempotent migrator: Magistrala v0.30.0 (per-service Postgres) → Atom IAM
(single Postgres). See [PLAN.md](./PLAN.md) for the full mapping and runbook.

## One-command migration (recommended)

If you just want to migrate an old deployment and bring it up with
`make run_latest`, use the orchestrator — it handles all the port / volume /
container-name collisions for you:

```bash
make migrate_atom                 # dry-run: reads + validates, writes nothing
make migrate_atom args="--apply"  # perform the migration
make migrate_atom args="--verify" # reconcile source vs Atom afterwards
```

Then:

```bash
# stop the old stack, then:
make run_latest
```

…and the new deployment serves the migrated data.

How it stays collision-free: [`migrate.sh`](./migrate.sh) +
[`docker-compose.migrate.yaml`](./docker-compose.migrate.yaml) run everything in
their own Compose project (`atommig`) on a private network, binding **no host
ports** and using **no fixed container names**, so they never clash with a
running Magistrala (old or `run_latest`) stack. It:

1. mounts the eight old per-service DB volumes
   (`magistrala_magistrala-<svc>-db-volume`) into throwaway Postgres containers
   (Postgres major version + data-dir layout auto-detected from the volume);
2. brings up an Atom + Postgres on the **same** volume `make run_latest` mounts
   (`<DOCKER_PROJECT>_magistrala-atom-db-volume`), so Atom seeds its schema there
   and the migrated rows persist for the next `run_latest`;
3. runs the migrator on that private network (reaching every DB by service name);
4. tears the stack down, leaving every volume intact.

All migration volumes are declared `external`, so `down` can never destroy data.

Prerequisites: run this on the machine that hosted the old Magistrala compose
stack. Stop that stack (`docker compose ... down`, **without** `-v`) so the
per-service DB volumes are free but still present locally — the migrator mounts
them directly. On `--apply`, the device-key CSV described below lands in
`tools/atom-migration/report/`.

Env overrides: `SRC_VOL_PREFIX` (old volume prefix, default
`magistrala_magistrala-`), `SRC_DB_USER` / `SRC_DB_PASS` (old Postgres creds,
default `magistrala`), `DOCKER_PROJECT` (run_latest project; default derived like
the Makefile), `MIGRATE_PROJECT` (isolated project name, default `atommig`),
`ATOM_IMAGE_TAG` (Atom image tag to seed with, default `latest`). Pass `--keep`
to leave the stack up for debugging.

### Atom schema freshness (`column "alias" does not exist`)

The migrator writes the **current** Atom schema. The schema is created by the
Atom image (`ghcr.io/absmach/atom:latest`, force-pulled each run) the first time
its target volume is seeded. Two things follow:

- A target volume that was **already seeded by an older Atom** (e.g. a previous
  `make run_latest`) keeps that old schema — Atom does not re-run an
  already-applied baseline, so newer columns like `tenants.alias` never appear.
  The migrator then fails with `column "alias" of relation "tenants" does not
  exist`.
- The orchestrator guards against this: it waits for `tenants.alias` to exist
  before running the migrator and aborts with guidance if it never does (instead
  of failing mid-apply).

Fix: rebuild the schema from scratch with

```bash
make migrate_atom args="--apply --fresh-atom"
```

`--fresh-atom` removes the existing Atom target volume so the current image lays
down the current schema. **Destructive** for that volume only — it discards any
data already in the Atom DB (the old per-service source volumes are never
touched). Equivalent manual reset: `docker volume rm
<DOCKER_PROJECT>_magistrala-atom-db-volume`.

### How volume names are resolved

Names are **derived by convention, not auto-discovered**. There are two sets.

**Source volumes (old deployment, read-only inputs).** Built from a prefix plus
a fixed per-service suffix:

```
<SRC_VOL_PREFIX><svc>-db-volume     # svc ∈ domains users clients channels groups auth re reports
```

`SRC_VOL_PREFIX` defaults to `magistrala_magistrala-`, i.e. old Compose project
`magistrala` + Docker Compose's own `magistrala-` volume key. So
`auth` → `magistrala_magistrala-auth-db-volume`. `migrate.sh` `docker volume
inspect`s all eight up front and aborts loudly if any is missing. The same
`${SRC_VOL_PREFIX}` feeds the `external` volume names in
`docker-compose.migrate.yaml`, so the script and Compose always agree. If your
old deployment used a different Compose project name, set `SRC_VOL_PREFIX`
(e.g. `SRC_VOL_PREFIX=myproj_magistrala-`).

**Atom target volume (where migrated data is written).** Must equal exactly the
volume `make run_latest` mounts, or the new stack would come up on a different,
empty volume. `make run_latest` mounts `magistrala-atom-db-volume`, which Docker
Compose prefixes with the project name `DOCKER_PROJECT`:

```
<DOCKER_PROJECT>_magistrala-atom-db-volume
```

`DOCKER_PROJECT` is itself derived from the git remote, replicating the Makefile
formula:

```sh
repo=$(git remote get-url origin | sed -E 's@.*/([^/]+)/([^/.]+)(\.git)?@\1_\2@')   # owner_repo
DOCKER_PROJECT=$(echo "$repo" | sed -E 's/[^a-zA-Z0-9]/_/g' | tr '[:upper:]' '[:lower:]')
ATOM_TARGET_VOLUME="${DOCKER_PROJECT}_magistrala-atom-db-volume"
```

`make migrate_atom` also passes `DOCKER_PROJECT="$(DOCKER_PROJECT)"` straight from
the Makefile, so the two stay in lockstep even if the git derivation would differ.
The target volume is created if it does not yet exist (so the schema-seed step can
write to it); `make run_latest` then reuses the same name. With no usable git
remote, or a remote that does not match the run_latest project, pass
`DOCKER_PROJECT=` explicitly.

**Source Postgres layout** (mount point + `PGDATA` + image major version) is the
one thing actually probed, not assumed: `migrate.sh` mounts the `users` source
volume in a throwaway `alpine` container, locates `PG_VERSION`, and derives the
mount path / `PGDATA` / `postgres:<major>-alpine` image from it (e.g. Postgres 18
keeps data under `/var/lib/postgresql/<major>/docker`). This makes the tool work
regardless of which Postgres version the old deployment ran.

The manual, lower-level steps below are still available if you need finer control.

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
services running. To migrate from restored volumes, start just the eight source DB
containers (`--no-deps` keeps compose from pulling in the app services they
depend on):

```bash
docker compose -f docker/docker-compose.yaml up -d --no-deps \
  auth-db users-db domains-db clients-db channels-db groups-db \
  re-db reports-db
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
