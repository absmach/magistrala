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
#   tools/atom-migration/migrate.sh --apply --keep   # leave the stack running for debugging
#
# Env overrides:
#   DOCKER_PROJECT     run_latest Compose project (default: derived like the Makefile)
#   SRC_VOL_PREFIX     old DB volume name prefix   (default: magistrala_magistrala-)
#   SRC_DB_USER/PASS   old Postgres credentials     (default: magistrala/magistrala)
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
for arg in "$@"; do
	case "$arg" in
		--keep) KEEP=true ;;
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

log "Isolated project : $MIGRATE_PROJECT"
log "Source volumes   : ${SRC_VOL_PREFIX}<svc>-db-volume"
log "Atom target vol  : $ATOM_TARGET_VOLUME"

# --- preflight: every source volume must exist ---
missing=()
for svc in domains users clients channels groups auth re reports; do
	vol="${SRC_VOL_PREFIX}${svc}-db-volume"
	docker volume inspect "$vol" >/dev/null 2>&1 || missing+=("$vol")
done
((${#missing[@]} == 0)) || die "missing source volume(s): ${missing[*]}
Run this on the machine whose stopped old Magistrala stack still has these
volumes, or set SRC_VOL_PREFIX if the old Compose project used another name."

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
# ATOM_DB_* / MG_RELEASE_TAG interpolate; these exports take precedence.
export SRC_VOL_PREFIX SRC_PG_IMAGE SRC_MOUNT SRC_PGDATA ATOM_TARGET_VOLUME
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
for i in $(seq 1 60); do
	if dc exec -T atom-db psql -U "${ATOM_DB_USER:-atom}" -d "${ATOM_DB_NAME:-atom}" -tAc \
		"SELECT to_regclass('public.tenants') IS NOT NULL AND to_regclass('public.entities') IS NOT NULL;" 2>/dev/null \
		| grep -qx t; then
		log "Atom schema ready"
		break
	fi
	[[ $i -eq 60 ]] && die "Atom did not initialise its schema in time (check: dc logs atom)"
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
