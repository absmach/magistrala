#!/bin/sh
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

SUPERMQ_SCHEMA="$REPO_ROOT/docker/supermq-docker/spicedb/schema.zed"
OVERRIDE_SCHEMA="$REPO_ROOT/docker/spicedb/override-schema.zed"
OUTPUT_SCHEMA="$REPO_ROOT/docker/spicedb/combined-schema.zed"

if [ ! -f "$SUPERMQ_SCHEMA" ]; then
    echo "ERROR: $SUPERMQ_SCHEMA not found" >&2
    exit 1
fi

if [ ! -f "$OVERRIDE_SCHEMA" ]; then
    echo "ERROR: $OVERRIDE_SCHEMA not found" >&2
    exit 1
fi

mkdir -p "$(dirname "$OUTPUT_SCHEMA")"

tmp_supermq_schema=$(mktemp)
tmp_supermq_merged=$(mktemp)
tmp_override_remaining=$(mktemp)
tmp_overlay_domain=$(mktemp)
tmp_overlay_team=$(mktemp)
tmp_overlay_domain_relations=$(mktemp)
tmp_overlay_team_relations=$(mktemp)
tmp_overlay_domain_permissions=$(mktemp)
tmp_overlay_membership_extension=$(mktemp)

cleanup() {
    rm -f "$tmp_supermq_schema"
    rm -f "$tmp_supermq_merged"
    rm -f "$tmp_override_remaining"
    rm -f "$tmp_overlay_domain"
    rm -f "$tmp_overlay_team"
    rm -f "$tmp_overlay_domain_relations"
    rm -f "$tmp_overlay_team_relations"
    rm -f "$tmp_overlay_domain_permissions"
    rm -f "$tmp_overlay_membership_extension"
}
trap cleanup EXIT INT TERM

cp "$SUPERMQ_SCHEMA" "$tmp_supermq_schema"

# Extract the first domain overlay block from override schema.
if ! awk '
    BEGIN {
        in_domain = 0
        found = 0
    }
    !found && /^definition domain[[:space:]]*{/ {
        in_domain = 1
        found = 1
        next
    }
    in_domain {
        if ($0 ~ /^}/) {
            in_domain = 0
            next
        }
        print
    }
    END {
        if (!found) {
            exit 1
        }
    }
' "$OVERRIDE_SCHEMA" > "$tmp_overlay_domain"; then
    echo "ERROR: definition domain block not found in $OVERRIDE_SCHEMA" >&2
    exit 1
fi

# Extract the first team overlay block from override schema.
if ! awk '
    BEGIN {
        in_team = 0
        found = 0
    }
    !found && /^definition team[[:space:]]*{/ {
        in_team = 1
        found = 1
        next
    }
    in_team {
        if ($0 ~ /^}/) {
            in_team = 0
            next
        }
        print
    }
    END {
        if (!found) {
            exit 1
        }
    }
' "$OVERRIDE_SCHEMA" > "$tmp_overlay_team"; then
    echo "ERROR: definition team block not found in $OVERRIDE_SCHEMA" >&2
    exit 1
fi

# Read explicit domain overlay relations from override domain block.
awk '
    /^[[:space:]]*relation[[:space:]]+[A-Za-z0-9_]+:[[:space:]]*role#member[[:space:]]*\|[[:space:]]*team#member[[:space:]]*$/ {
        line = $0
        sub(/^[[:space:]]*/, "\t", line)
        print line
    }
' "$tmp_overlay_domain" > "$tmp_overlay_domain_relations"

# Read explicit team overlay relations from override team block.
awk '
    /^[[:space:]]*relation[[:space:]]+[A-Za-z0-9_]+:[[:space:]]*role#member[[:space:]]*\|[[:space:]]*team#member[[:space:]]*$/ {
        line = $0
        sub(/^[[:space:]]*/, "\t", line)
        print line
    }
' "$tmp_overlay_team" > "$tmp_overlay_team_relations"

# Read explicit domain overlay permissions from override domain block.
awk '
    /^[[:space:]]*permission[[:space:]]+[A-Za-z0-9_]+_permission[[:space:]]*=/ {
        line = $0
        sub(/^[[:space:]]*/, "\t", line)
        print line
    }
' "$tmp_overlay_domain" > "$tmp_overlay_domain_permissions"

# Read explicit domain membership extension expression from override domain block.
awk '
    /^[[:space:]]*permission[[:space:]]+membership_extension[[:space:]]*=/ {
        line = $0
        sub(/^[[:space:]]*permission[[:space:]]+membership_extension[[:space:]]*=[[:space:]]*/, "", line)
        print line
        exit
    }
' "$tmp_overlay_domain" > "$tmp_overlay_membership_extension"

if [ ! -s "$tmp_overlay_domain_relations" ]; then
    echo "ERROR: no domain relation overlay lines found in definition domain block of $OVERRIDE_SCHEMA" >&2
    exit 1
fi

if [ ! -s "$tmp_overlay_team_relations" ]; then
    echo "ERROR: no team relation overlay lines found in definition team block of $OVERRIDE_SCHEMA" >&2
    exit 1
fi

if [ ! -s "$tmp_overlay_domain_permissions" ]; then
    echo "ERROR: no domain permission overlay lines found in definition domain block of $OVERRIDE_SCHEMA" >&2
    exit 1
fi

if [ ! -s "$tmp_overlay_membership_extension" ]; then
    echo "ERROR: permission membership_extension not found in definition domain block of $OVERRIDE_SCHEMA" >&2
    exit 1
fi

# Remove the first domain and first team overlay blocks from override schema before appending.
if ! awk '
    BEGIN {
        skip_domain = 0
        skip_team = 0
        removed_domain = 0
        removed_team = 0
    }
    !removed_domain && /^definition domain[[:space:]]*{/ {
        skip_domain = 1
        removed_domain = 1
        next
    }
    skip_domain {
        if ($0 ~ /^}/) {
            skip_domain = 0
        }
        next
    }
    !removed_team && /^definition team[[:space:]]*{/ {
        skip_team = 1
        removed_team = 1
        next
    }
    skip_team {
        if ($0 ~ /^}/) {
            skip_team = 0
        }
        next
    }
    { print }
    END {
        if (!removed_domain || !removed_team) {
            exit 1
        }
    }
' "$OVERRIDE_SCHEMA" > "$tmp_override_remaining"; then
    echo "ERROR: failed to strip definition domain/team blocks from $OVERRIDE_SCHEMA" >&2
    exit 1
fi

# Inject explicit override lines into SuperMQ domain and team definitions.
awk \
    -v domain_relations_file="$tmp_overlay_domain_relations" \
    -v team_relations_file="$tmp_overlay_team_relations" \
    -v domain_permissions_file="$tmp_overlay_domain_permissions" \
    -v membership_extension_file="$tmp_overlay_membership_extension" '
    BEGIN {
        while ((getline line < domain_relations_file) > 0) {
            domain_relations = domain_relations line ORS
        }
        close(domain_relations_file)

        while ((getline line < team_relations_file) > 0) {
            team_relations = team_relations line ORS
        }
        close(team_relations_file)

        while ((getline line < domain_permissions_file) > 0) {
            domain_permissions = domain_permissions line ORS
        }
        close(domain_permissions_file)

        membership_extension = ""
        if ((getline line < membership_extension_file) > 0) {
            membership_extension = line
        }
        close(membership_extension_file)

        in_domain = 0
        in_team = 0
        in_domain_membership = 0
        inserted_domain_relations = 0
        inserted_team_relations = 0
        inserted_domain_permissions = 0
        inserted_domain_membership = 0
    }
    {
        if ($0 ~ /^definition domain[[:space:]]*{/) {
            in_domain = 1
        } else if ($0 ~ /^definition team[[:space:]]*{/) {
            in_team = 1
        }

        if (in_domain && $0 ~ /^[[:space:]]*permission membership[[:space:]]*=/) {
            in_domain_membership = 1
        }

        if (in_domain && in_domain_membership && $0 ~ /organization->admin[[:space:]]*$/) {
            membership_tail = $0
            sub(/[[:space:]]*\+[[:space:]]*organization->admin[[:space:]]*$/, " +", membership_tail)
            print membership_tail
            print "\t" membership_extension " +"
            print "\torganization->admin"
            in_domain_membership = 0
            inserted_domain_membership = 1
            next
        }

        print $0

        if (in_domain && $0 ~ /^[[:space:]]*relation group_view_role_users: role#member \| team#member[[:space:]]*$/) {
            print ""
            print "\t// Magistrala-specific relations"
            printf "%s", domain_relations
            inserted_domain_relations = 1
        }

        if (in_team && $0 ~ /^[[:space:]]*relation group_view_role_users: role#member \| team#member[[:space:]]*$/) {
            print ""
            print "\t// Magistrala-specific relations"
            printf "%s", team_relations
            inserted_team_relations = 1
        }

        if (in_domain && $0 ~ /^[[:space:]]*permission group_view_role_users_permission = group_view_role_users \+ team->group_view_role_users \+ organization->admin[[:space:]]*$/) {
            print ""
            print "\t// Magistrala-specific permissions"
            printf "%s", domain_permissions
            inserted_domain_permissions = 1
        }

        if (in_domain && $0 ~ /^}/) {
            in_domain = 0
            in_domain_membership = 0
        } else if (in_team && $0 ~ /^}/) {
            in_team = 0
        }
    }
    END {
        if (!inserted_domain_relations || !inserted_team_relations || !inserted_domain_permissions || !inserted_domain_membership) {
            exit 1
        }
    }
' "$tmp_supermq_schema" > "$tmp_supermq_merged" || {
    echo "ERROR: failed to merge override schema into SuperMQ schema" >&2
    exit 1
}

first_domain_relation=$(awk 'NR == 1 {print $2}' "$tmp_overlay_domain_relations")
first_team_relation=$(awk 'NR == 1 {print $2}' "$tmp_overlay_team_relations")
first_domain_permission=$(awk 'NR == 1 {print $2}' "$tmp_overlay_domain_permissions")

if [ -z "$first_domain_relation" ] || [ -z "$first_team_relation" ] || [ -z "$first_domain_permission" ]; then
    echo "ERROR: failed to verify merged overlay lines" >&2
    exit 1
fi

sub_first_domain_relation=${first_domain_relation%%:*}
sub_first_team_relation=${first_team_relation%%:*}

if ! grep -Fq "relation $sub_first_domain_relation: role#member | team#member" "$tmp_supermq_merged"; then
    echo "ERROR: merged schema is missing domain relation $sub_first_domain_relation" >&2
    exit 1
fi

if ! grep -Fq "relation $sub_first_team_relation: role#member | team#member" "$tmp_supermq_merged"; then
    echo "ERROR: merged schema is missing team relation $sub_first_team_relation" >&2
    exit 1
fi

if ! grep -Fq "permission $first_domain_permission" "$tmp_supermq_merged"; then
    echo "ERROR: merged schema is missing domain permission $first_domain_permission" >&2
    exit 1
fi

membership_extension_value=$(cat "$tmp_overlay_membership_extension")
if ! grep -Fq "$membership_extension_value +" "$tmp_supermq_merged"; then
    echo "ERROR: merged schema is missing domain membership_extension values" >&2
    exit 1
fi

{
    cat <<'EOF'
// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
// Code generated by scripts/combine-schema.sh. DO NOT EDIT.
//
// Combined from:
// - docker/supermq-docker/spicedb/schema.zed
// - docker/spicedb/override-schema.zed

EOF
    cat "$tmp_supermq_merged" "$tmp_override_remaining"
} > "$OUTPUT_SCHEMA"

echo "Combined schema generated at: $OUTPUT_SCHEMA"
