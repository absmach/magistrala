#!/usr/bin/env bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -Eeuo pipefail

usage() {
	cat <<'EOF'
Download Magistrala database Docker volumes from a remote host.

Usage:
  scripts/download-db-volumes.sh <host> <ssh-key> [output-directory]

Example:
  scripts/download-db-volumes.sh 203.0.113.10 ~/.ssh/id_ed25519
  SSH_USER=root scripts/download-db-volumes.sh 203.0.113.10 ~/.ssh/id_ed25519 ./backups
  VOLUME_PREFIX=client_ scripts/download-db-volumes.sh 203.0.113.10 ~/.ssh/id_ed25519

Optional environment variables:
  SSH_USER           Remote SSH user. Default: ubuntu
  SSH_PORT           Remote SSH port. Default: 22
  COMPOSE_PROJECT    Docker Compose project name. Default: magistrala
  VOLUME_PREFIX      Select database volumes by their full Docker name prefix,
                     for example "client_". By default, volumes are selected
                     using the COMPOSE_PROJECT Docker label.
  STOP_CONTAINERS    Stop containers using database volumes for a consistent
                     backup. Default: true
  BACKUP_IMAGE       Helper image used to create the archive. Default:
                     alpine:3.22

The archive preserves each full Docker volume name and is accompanied by a
.volumes.txt manifest. By default, the script temporarily stops only containers
mounting the selected database volumes and restarts exactly those containers.
EOF
}

if [[ ${1:-} == "-h" || ${1:-} == "--help" ]]; then
	usage
	exit 0
fi

if (( $# < 2 || $# > 3 )); then
	usage >&2
	exit 2
fi

host=$1
ssh_key=$2
output_dir=${3:-./magistrala-db-backups}

ssh_user=${SSH_USER:-ubuntu}
ssh_port=${SSH_PORT:-22}
compose_project=${COMPOSE_PROJECT:-magistrala}
volume_prefix=${VOLUME_PREFIX:-}
stop_containers=${STOP_CONTAINERS:-true}
backup_image=${BACKUP_IMAGE:-alpine:3.22}

if [[ ! -r $ssh_key ]]; then
	echo "SSH key is not readable: $ssh_key" >&2
	exit 1
fi

if [[ ! $ssh_port =~ ^[0-9]+$ ]] || (( ssh_port < 1 || ssh_port > 65535 )); then
	echo "SSH_PORT must be a number between 1 and 65535." >&2
	exit 2
fi

if [[ ! $compose_project =~ ^[A-Za-z0-9_.-]+$ ]]; then
	echo "COMPOSE_PROJECT contains unsupported characters." >&2
	exit 2
fi

if [[ -n $volume_prefix && ! $volume_prefix =~ ^[A-Za-z0-9_.-]+$ ]]; then
	echo "VOLUME_PREFIX contains unsupported characters." >&2
	exit 2
fi

if [[ ! $backup_image =~ ^[A-Za-z0-9._/:@-]+$ ]]; then
	echo "BACKUP_IMAGE contains unsupported characters." >&2
	exit 2
fi

case "$stop_containers" in
	true|false)
		;;
	*)
		echo "STOP_CONTAINERS must be true or false." >&2
		exit 2
		;;
esac

mkdir -p "$output_dir"

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
safe_host=$(printf '%s' "$host" | tr -c 'A-Za-z0-9_.-' '_')
archive="$output_dir/${compose_project}-db-volumes-${safe_host}-${timestamp}.tar.gz"
partial_archive="${archive}.part"
manifest="${archive}.volumes.txt"
partial_manifest="${manifest}.part"

cleanup_local() {
	rm -f "$partial_archive"
	rm -f "$partial_manifest"
}
trap cleanup_local EXIT

ssh_options=(
	-i "$ssh_key"
	-p "$ssh_port"
	-o BatchMode=yes
	-o StrictHostKeyChecking=accept-new
)
remote="${ssh_user}@${host}"

echo "Downloading database volumes from $remote..."

if ! ssh "${ssh_options[@]}" "$remote" sh -s -- \
	"$compose_project" "$stop_containers" "$backup_image" "$volume_prefix" \
	>"$partial_archive" <<'REMOTE_SCRIPT'
set -eu

project=$1
stop_containers=$2
backup_image=$3
volume_prefix=${4-}
stopped_containers=

if docker info >/dev/null 2>&1; then
	docker_mode=direct
elif command -v sudo >/dev/null 2>&1 && sudo -n docker info >/dev/null 2>&1; then
	docker_mode=sudo
else
	echo "Cannot access Docker as the SSH user or with passwordless sudo." >&2
	exit 1
fi

docker_cmd() {
	if [ "$docker_mode" = direct ]; then
		docker "$@"
	else
		sudo -n docker "$@"
	fi
}

cleanup_remote() {
	status=$?
	trap - EXIT

	if [ -n "$stopped_containers" ]; then
		echo "Restarting database containers..." >&2
		for container in $stopped_containers; do
			if ! docker_cmd start "$container" >/dev/null; then
				echo "Failed to restart container $container." >&2
				status=1
			fi
		done
	fi

	exit "$status"
}

on_signal() {
	exit 130
}

trap cleanup_remote EXIT
trap on_signal HUP INT TERM

volumes=
if [ -n "$volume_prefix" ]; then
	available_volumes=$(docker_cmd volume ls --format '{{.Name}}')
	selection_description="name prefix '$volume_prefix'"
else
	available_volumes=$(docker_cmd volume ls \
		--filter "label=com.docker.compose.project=$project" \
		--format '{{.Name}}')
	selection_description="Compose project '$project'"
fi

for volume in $available_volumes; do
	if [ -n "$volume_prefix" ]; then
		case "$volume" in
			"$volume_prefix"*)
				;;
			*)
				continue
				;;
		esac
	fi

	logical_name=$(docker_cmd volume inspect "$volume" \
		--format '{{with index .Labels "com.docker.compose.volume"}}{{.}}{{end}}')
	candidate_name=${logical_name:-$volume}

	case "$candidate_name" in
		*-db-volume|*-redis-volume|*-timescale-writer-volume|*-openbao-data)
			volumes="${volumes}${volumes:+ }${volume}"
			;;
	esac
done

if [ -z "$volumes" ]; then
	echo "No database volumes found for $selection_description." >&2
	echo "Set VOLUME_PREFIX or COMPOSE_PROJECT to match the remote deployment." >&2
	exit 1
fi

echo "Volumes selected for backup:" >&2
for volume in $volumes; do
	echo "  $volume" >&2
done

# Pull the helper image before stopping databases to minimize downtime.
if ! docker_cmd image inspect "$backup_image" >/dev/null 2>&1; then
	echo "Pulling backup helper image $backup_image..." >&2
	docker_cmd pull "$backup_image" >&2
fi

if [ "$stop_containers" = "true" ]; then
	containers=
	for volume in $volumes; do
		for container in $(docker_cmd ps \
			--filter "volume=$volume" \
			--format '{{.ID}}'); do
			case " $containers " in
				*" $container "*)
					;;
				*)
					containers="${containers}${containers:+ }${container}"
					;;
			esac
		done
	done

	if [ -n "$containers" ]; then
		echo "Stopping database containers for a consistent backup..." >&2
		for container in $containers; do
			stopped_containers="${stopped_containers}${stopped_containers:+ }${container}"
			docker_cmd stop "$container" >/dev/null
		done
	fi
else
	echo "Warning: database containers remain running; the backup may be inconsistent." >&2
fi

set -- run --rm
for volume in $volumes; do
	set -- "$@" --mount "type=volume,src=$volume,dst=/backup/$volume,readonly"
done
set -- "$@" "$backup_image" tar -czf - -C /backup .

echo "Streaming compressed volume archive..." >&2
docker_cmd "$@"
REMOTE_SCRIPT
then
	echo "Database volume download failed." >&2
	exit 1
fi

if ! gzip -t "$partial_archive"; then
	echo "Downloaded archive failed gzip integrity validation." >&2
	exit 1
fi

tar -tzf "$partial_archive" |
	awk -F/ '$1 == "." && $2 != "" { print $2 }' |
	LC_ALL=C sort -u >"$partial_manifest"

if [[ ! -s $partial_manifest ]]; then
	echo "Downloaded archive contains no volume directories." >&2
	exit 1
fi

mv "$partial_archive" "$archive"
mv "$partial_manifest" "$manifest"
trap - EXIT

if command -v sha256sum >/dev/null 2>&1; then
	sha256sum "$archive" >"${archive}.sha256"
elif command -v shasum >/dev/null 2>&1; then
	shasum -a 256 "$archive" >"${archive}.sha256"
fi

echo "Backup downloaded successfully:"
echo "  $archive"
echo "  $manifest"
if [[ -f ${archive}.sha256 ]]; then
	echo "  ${archive}.sha256"
fi
