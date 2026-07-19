#!/usr/bin/env bash
# Set an Atom password credential for migrated humans (who land with none, since
# migration drops bcrypt passwords that argon2 can't verify). Inserts an argon2id
# credential with identifier=NULL so login works with either email OR username.
#
# Usage:
#   ./scripts/set-atom-passwords.sh                 # all users without a password
#   ./scripts/set-atom-passwords.sh <username>      # one user (by entities.name); replaces existing
#   PW='somelongpass' ./scripts/set-atom-passwords.sh <username>
#
# Container/db override via env: PGCONT=atom-postgres-1 PGUSER=atom PGDB=atom
set -euo pipefail

PGCONT="${PGCONT:-magistrala-atom-db}"
PGUSER="${PGUSER:-atom}"
PGDB="${PGDB:-atom}"

TARGET_USER="${1:-}"

PW="${PW:-}"
if [ -z "$PW" ]; then
	read -rsp "New password: " PW; echo
fi
[ "${#PW}" -ge 12 ] || { echo "Password must be >= 12 chars (Atom minimum)."; exit 1; }

psql() { docker exec -i "$PGCONT" psql -v ON_ERROR_STOP=1 -U "$PGUSER" -d "$PGDB" "$@"; }

# Single-quote a value for safe inline use in SQL (doubles embedded quotes).
sqlq() { printf "%s" "$1" | sed "s/'/''/g"; }

# argon2id PHC hash via throwaway alpine container. Params need not match Atom's
# defaults: the PHC string is self-describing, so verify_password works regardless.
hash_pw() {
	local salt
	salt=$(head -c12 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | cut -c1-16)
	printf '%s' "$PW" | docker run --rm -i alpine sh -c \
		"apk add -q argon2 >/dev/null && argon2 '$salt' -id -m 15 -t 2 -p 1 -e" \
		| tr -d '\r\n'
}

if [ -n "$TARGET_USER" ]; then
	# Single user, matched by entities.name (Atom's username login key).
	uq=$(sqlq "$TARGET_USER")
	ids=$(psql -tAc \
		"SELECT id FROM entities
		 WHERE kind='human' AND deleted_at IS NULL AND name = '$uq'")
	if [ -z "$ids" ]; then
		echo "No human entity with name = '$TARGET_USER'."
		exit 1
	fi
	if [ "$(printf '%s\n' "$ids" | grep -c .)" -gt 1 ]; then
		echo "Multiple entities named '$TARGET_USER'; refusing. Ids:"; echo "$ids"
		exit 1
	fi
	# Replace any existing password credential for a clean, predictable login.
	psql -c "DELETE FROM credentials WHERE entity_id = '$(sqlq "$ids")' AND kind='password'" >/dev/null
else
	# All migrated users that have no password credential yet.
	ids=$(psql -tAc \
		"SELECT id FROM entities
		 WHERE kind='human' AND deleted_at IS NULL
		   AND id NOT IN (SELECT entity_id FROM credentials WHERE kind='password')")
	if [ -z "$ids" ]; then
		echo "No users without a password credential. Nothing to do."
		exit 0
	fi
fi

# Read ids into an array first so the insert loop owns no stdin (docker exec -i
# below would otherwise consume it).
mapfile -t id_list <<EOF
$ids
EOF

n=0
for id in "${id_list[@]}"; do
	[ -n "$id" ] || continue
	hash=$(hash_pw)
	# printf (not a double-quoted string) so the argon2 hash's '$' chars stay
	# literal. SQL is piped via stdin because this psql does not interpolate
	# :'var' inside -c. id is a UUID and hash is a PHC string: neither contains
	# a single quote, so direct quoting is safe.
	printf "INSERT INTO credentials (entity_id, kind, identifier, secret_hash, status) VALUES ('%s','password',NULL,'%s','active');\n" \
		"$id" "$hash" | psql >/dev/null
	echo "  set password for $id"
	n=$((n+1))
done

echo "Done. Set passwords for $n user(s)."
echo "Login with email OR username + this password."
