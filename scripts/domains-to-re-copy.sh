#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

####
## Copy domains and domain role tables from the domains database to the RE (rules engine) database.
## This is needed because the RE service maintains a local copy of domain tables for SQL JOINs
## during access-controlled listing queries.
##
## Run this after the domain_8 migration has been applied so that rule_* actions
## are already present in domains_role_actions.
####

set -e
set -o pipefail

# Source: Domains database
SRC_DB_HOST="${SRC_DB_HOST:-localhost}"
SRC_DB_PORT="${SRC_DB_PORT:-6003}"
SRC_DB_USER="${SRC_DB_USER:-magistrala}"
SRC_DB_PASSWORD="${SRC_DB_PASSWORD:-magistrala}"
SRC_DB_NAME="${SRC_DB_NAME:-domains}"

# Destination: RE database
DEST_DB_HOST="${DEST_DB_HOST:-localhost}"
DEST_DB_PORT="${DEST_DB_PORT:-6009}"
DEST_DB_USER="${DEST_DB_USER:-magistrala}"
DEST_DB_PASSWORD="${DEST_DB_PASSWORD:-magistrala}"
DEST_DB_NAME="${DEST_DB_NAME:-rules_engine}"

# Tables to copy (order matters due to foreign key constraints)
TABLES=("domains" "domains_roles" "domains_role_actions" "domains_role_members")

for TABLE_NAME in "${TABLES[@]}"; do
    echo "Copying $SRC_DB_NAME.$TABLE_NAME -> $DEST_DB_NAME.$TABLE_NAME ..."

    PGPASSWORD="$SRC_DB_PASSWORD" psql -h "$SRC_DB_HOST" -p "$SRC_DB_PORT" -U "$SRC_DB_USER" -d "$SRC_DB_NAME" -c "COPY $TABLE_NAME TO STDOUT" | \
    PGPASSWORD="$DEST_DB_PASSWORD" psql -h "$DEST_DB_HOST" -p "$DEST_DB_PORT" -U "$DEST_DB_USER" -d "$DEST_DB_NAME" -c "TRUNCATE TABLE $TABLE_NAME CASCADE; COPY $TABLE_NAME FROM STDIN"

    if [ $? -ne 0 ]; then
        echo "Error: Failed to copy table $TABLE_NAME. Exiting."
        exit 1
    fi

    echo "Table $TABLE_NAME copied successfully."
done

echo "All domain tables copied to RE database successfully!"
