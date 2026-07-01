#!/usr/bin/env bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0
#
# One-command Magistrala v0.30.0 -> Atom migration.
#
# Brings up an isolated, collision-free stack (its own Compose project + private
# network, no host ports, no fixed container names), seeds the Atom schema into
# the SAME volume `make run_latest` uses, runs the migrator, then tears the stack
# down leaving every volume intact. After it finishes you can simply:
#
#     make run_latest
#
# and the new deployment serves the migrated data.
#
# Usage:
#   tools/atom-migration/migrate.sh            # dry-run (reads + validates, writes nothing)
#   tools/atom-migration/migrate.sh --apply    # perform the migration
#   tools/atom-migration/migrate.sh --verify   # reconcile source vs Atom after apply
#   tools/atom-migration/migrate.sh --apply --fresh-atom  # rebuild Atom schema from scratch
#   tools/atom-migration/migrate.sh --apply --keep   # leave the stack running for debugging
#
# --fresh-atom discards any existing Atom target volume first, so the current Atom
# image lays down the current schema. Use it when a previous run/`make run_latest`
# left an older Atom schema in the volume (symptom: "column alias does not exist").
#
# Env overrides:
#   DOCKER_PROJECT     run_latest Compose project (default: derived like the Makefile)
#   SRC_VOL_PREFIX     old DB volume name prefix   (default: magistrala_magistrala-)
#   SRC_DB_USER/PASS   old Postgres credentials     (default: magistrala/magistrala)
#   ATOM_IMAGE         Atom image used to seed schema (default: docker/.env or ghcr.io/absmach/atom:latest)
#   ATOM_IMAGE_TAG     shorthand for ghcr.io/absmach/atom:<tag> when ATOM_IMAGE is unset
#   ATOM_PULL_POLICY   Compose pull policy for Atom image (default: always for ghcr.io/absmach/atom, missing otherwise)
#   MIGRATE_PROJECT    isolated Compose project name (default: atommig)

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.migrate.yaml"
ENV_FILE="$REPO_ROOT/docker/.env"
MIGRATE_PROJECT="${MIGRATE_PROJECT:-atommig}"
SRC_VOL_PREFIX="${SRC_VOL_PREFIX:-magistrala_magistrala-}"

MIGRATOR_ARGS=()
KEEP=false
FRESH_ATOM=false
for arg in "$@"; do
	case "$arg" in
		--keep) KEEP=true ;;
		--fresh-atom) FRESH_ATOM=true ;;
		--apply|--verify|--dry-run) MIGRATOR_ARGS+=("$arg") ;;
		--unmapped-action=*|--report-dir=*) MIGRATOR_ARGS+=("$arg") ;;
		-h|--help) grep '^#' "$0" | sed 's/^# \{0,1\}//' | head -40; exit 0 ;;
		*) echo "unknown argument: $arg" >&2; exit 2 ;;
	esac
done

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

command -v docker >/dev/null || die "docker not found"
[[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"

# DOCKER_PROJECT: match the Makefile derivation so we target the volume that
# `make run_latest` will mount.
if [[ -z "${DOCKER_PROJECT:-}" ]]; then
	repo="$(git -C "$REPO_ROOT" remote get-url origin 2>/dev/null \
		| sed -E 's@.*/([^/]+)/([^/.]+)(\.git)?@\1_\2@')"
	DOCKER_PROJECT="$(echo "$repo" | sed -E 's/[^a-zA-Z0-9]/_/g' | tr '[:upper:]' '[:lower:]')"
fi
[[ -n "$DOCKER_PROJECT" ]] || die "could not determine DOCKER_PROJECT (set it explicitly)"
ATOM_TARGET_VOLUME="${DOCKER_PROJECT}_magistrala-atom-db-volume"

# Atom DB connection settings come from docker/.env (Compose interpolates them
# from --env-file). Read the same values here so our readiness poll connects with
# the right user/db -- defaulting to "atom" would loop forever if .env overrides
# them.
envget() { sed -nE "s/^[[:space:]]*$1=//p" "$ENV_FILE" | tail -1 | tr -d '"'"'"; }
ATOM_DB_USER="$(envget ATOM_DB_USER)";     ATOM_DB_USER="${ATOM_DB_USER:-atom}"
ATOM_DB_NAME="$(envget ATOM_DB_NAME)";     ATOM_DB_NAME="${ATOM_DB_NAME:-atom}"

atom_image_from_env_file="$(envget ATOM_IMAGE)"
if [[ -n "${ATOM_IMAGE:-}" ]]; then
	:
elif [[ -n "${ATOM_IMAGE_TAG:-}" ]]; then
	ATOM_IMAGE="ghcr.io/absmach/atom:$ATOM_IMAGE_TAG"
elif [[ -n "$atom_image_from_env_file" ]]; then
	ATOM_IMAGE="$atom_image_from_env_file"
else
	ATOM_IMAGE="ghcr.io/absmach/atom:latest"
fi
if [[ -z "${ATOM_PULL_POLICY:-}" ]]; then
	if [[ "$ATOM_IMAGE" == ghcr.io/absmach/atom:* ]]; then
		ATOM_PULL_POLICY=always
	else
		ATOM_PULL_POLICY=missing
	fi
fi
export ATOM_IMAGE ATOM_PULL_POLICY

log "Isolated project : $MIGRATE_PROJECT"
log "Source volumes   : ${SRC_VOL_PREFIX}<svc>-db-volume"
log "Atom target vol  : $ATOM_TARGET_VOLUME"
log "Atom image       : $ATOM_IMAGE (pull_policy=$ATOM_PULL_POLICY)"

# --- preflight: every source volume must exist ---
missing=()
for svc in domains users clients channels groups auth re reports; do
	vol="${SRC_VOL_PREFIX}${svc}-db-volume"
	docker volume inspect "$vol" >/dev/null 2>&1 || missing+=("$vol")
done
((${#missing[@]} == 0)) || die "missing source volume(s): ${missing[*]}
Run this on the machine whose stopped old Magistrala stack still has these
volumes, or set SRC_VOL_PREFIX if the old Compose project used another name."

# --fresh-atom: rebuild the target schema from scratch. A pre-existing Atom volume
# (e.g. from an earlier `make run_latest`) keeps whatever schema it was seeded
# with -- Atom's migrations won't re-run an already-applied baseline, so an old
# schema (missing newer columns like tenants.alias) would survive and break the
# load. Removing the volume forces the current Atom image to lay down the current
# schema. Destructive: any data already in that Atom volume is discarded.
if [[ "$FRESH_ATOM" == true ]]; then
	if docker volume inspect "$ATOM_TARGET_VOLUME" >/dev/null 2>&1; then
		log "Resetting Atom target volume $ATOM_TARGET_VOLUME (--fresh-atom)"
		docker compose --env-file "$ENV_FILE" -p "$MIGRATE_PROJECT" -f "$COMPOSE_FILE" down >/dev/null 2>&1 || true
		docker volume rm -f "$ATOM_TARGET_VOLUME" >/dev/null
	fi
fi

# Atom target volume: created here if absent so the seed step can write to it;
# `make run_latest` reuses the same name.
docker volume inspect "$ATOM_TARGET_VOLUME" >/dev/null 2>&1 || {
	log "Creating Atom target volume $ATOM_TARGET_VOLUME"
	docker volume create "$ATOM_TARGET_VOLUME" >/dev/null
}

# --- detect source Postgres layout (mount point + PGDATA + image major) ---
log "Detecting source Postgres layout"
layout="$(docker run --rm -v "${SRC_VOL_PREFIX}users-db-volume":/d alpine:3.22 sh -c '
	if [ -f /d/PG_VERSION ]; then
		echo "classic $(cat /d/PG_VERSION) /"
	else
		p=$(find /d -maxdepth 4 -name PG_VERSION 2>/dev/null | head -1)
		[ -n "$p" ] && echo "nested $(cat "$p") ${p#/d/}" || echo "unknown 0 /"
	fi')"
read -r kind major rel <<<"$layout"
[[ "$kind" != "unknown" ]] || die "could not find PG_VERSION in source volume"
if [[ "$kind" == "nested" ]]; then
	# rel = <major>/docker/PG_VERSION -> PGDATA dir = /var/lib/postgresql/<major>/docker
	subdir="$(dirname "/$rel")"          # /<major>/docker
	SRC_MOUNT="/var/lib/postgresql"
	SRC_PGDATA="/var/lib/postgresql${subdir}"
else
	SRC_MOUNT="/var/lib/postgresql/data"
	SRC_PGDATA="/var/lib/postgresql/data"
fi
SRC_PG_IMAGE="postgres:${major}-alpine"
log "Source Postgres  : $SRC_PG_IMAGE  (mount $SRC_MOUNT, PGDATA $SRC_PGDATA)"

# --- build the migrator image ---
log "Building migrator image"
docker build -q -f "$SCRIPT_DIR/Dockerfile" -t magistrala/atom-migration:dev "$REPO_ROOT" >/dev/null

# Variables consumed by the compose file. docker/.env is passed via --env-file so
# ATOM_DB_* interpolate; these exports take precedence.
export SRC_VOL_PREFIX SRC_PG_IMAGE SRC_MOUNT SRC_PGDATA ATOM_TARGET_VOLUME ATOM_IMAGE ATOM_PULL_POLICY
export SRC_DB_USER="${SRC_DB_USER:-magistrala}" SRC_DB_PASS="${SRC_DB_PASS:-magistrala}"
export HOST_UID="$(id -u)" HOST_GID="$(id -g)"

dc() { docker compose --env-file "$ENV_FILE" -p "$MIGRATE_PROJECT" -f "$COMPOSE_FILE" "$@"; }

cleanup() {
	if [[ "$KEEP" == true ]]; then
		log "Leaving stack up (--keep). Tear down with:"
		echo "    docker compose -p $MIGRATE_PROJECT -f $COMPOSE_FILE down"
		return
	fi
	log "Tearing down isolated stack (volumes are external and preserved)"
	dc down --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

# --- bring up source DBs + Atom DB, seed the Atom schema ---
log "Starting source DBs, Atom DB and Atom schema seeder"
dc up -d --wait domains-db users-db clients-db channels-db groups-db auth-db re-db reports-db atom-db
dc up -d atom

log "Waiting for Atom to apply its schema into the target volume"
# Require a column from a late migration (tenants.alias), not just table
# existence: Atom adds tables early and columns later, so checking only for the
# tables can race ahead of a not-yet-complete migration and hit the same
# "column alias does not exist" error mid-apply. A timeout here with alias still
# absent means the volume holds an old/incompatible Atom schema -> --fresh-atom.
schema_ready_q="SELECT (to_regclass('public.entities') IS NOT NULL)
	AND EXISTS (SELECT 1 FROM information_schema.columns
		WHERE table_name='tenants' AND column_name='alias');"
atom_cid="$(dc ps -q atom)"
for i in $(seq 1 60); do
	if dc exec -T atom-db psql -U "$ATOM_DB_USER" -d "$ATOM_DB_NAME" -tAc \
		"$schema_ready_q" 2>/dev/null | grep -qx t; then
		log "Atom schema ready"
		break
	fi
	# If the Atom container has stopped it will never create the schema -- fail
	# fast with its logs instead of waiting out the whole timeout.
	if [[ -n "$atom_cid" ]] && [[ "$(docker inspect -f '{{.State.Running}}' "$atom_cid" 2>/dev/null)" != "true" ]]; then
		echo "----- atom logs -----" >&2; dc logs --no-color --tail 50 atom >&2 || true
		die "Atom container exited before applying its schema (see logs above).
Common causes: missing/invalid ATOM_* settings in docker/.env (JWT_SECRET,
ATOM_KEY_ENCRYPTION_KEY, ATOM_SERVICE_SECRET, ATOM_ADMIN_SECRET) or it could not
reach atom-db."
	fi
	[[ $i -eq 60 ]] && { echo "----- atom logs -----" >&2; dc logs --no-color --tail 50 atom >&2 || true; die "Atom schema in $ATOM_TARGET_VOLUME never reached the expected version
(tenants.alias missing) within the timeout. If Atom is still migrating, retry; if
the volume holds an older Atom schema, re-run with --fresh-atom (discards that
volume's Atom data) or 'docker volume rm $ATOM_TARGET_VOLUME'."; }
	sleep 2
done

# --- run the migration ---
log "Running migrator ${MIGRATOR_ARGS[*]:-(dry-run)}"
set +e
dc run --rm migrator "${MIGRATOR_ARGS[@]}"
rc=$?
set -e

echo
if [[ $rc -eq 0 ]]; then
	log "Migrator finished OK. Report: tools/atom-migration/report/"
	if printf '%s\n' "${MIGRATOR_ARGS[@]}" | grep -qx -- --apply; then
		log "Next: stop the old stack, then 'make run_latest' to serve migrated data."
	fi
else
	die "migrator exited $rc (see report in tools/atom-migration/report/)"
fi
