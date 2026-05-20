#!/usr/bin/env sh
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
ENV_FILE="$ROOT_DIR/docker/.env"
COMPOSE_FILE="$ROOT_DIR/docker/docker-compose.yaml"

HOST=${MG_PUBLIC_HOST:-}
EMAIL=${MG_LETSENCRYPT_EMAIL:-}
LETSENCRYPT_ENABLED=${MG_LETSENCRYPT_ENABLED:-true}
STAGING=${MG_LETSENCRYPT_STAGING:-true}
FORCE_RENEWAL=${MG_LETSENCRYPT_FORCE_RENEWAL:-false}
PROJECT=${DOCKER_PROJECT:-magistrala}
TIMEOUT_SECONDS=${MG_LETSENCRYPT_TIMEOUT_SECONDS:-180}

usage() {
	cat <<EOF
Usage:
  MG_PUBLIC_HOST=example.com MG_LETSENCRYPT_EMAIL=admin@example.com [MG_LETSENCRYPT_STAGING=false] $0
  MG_PUBLIC_HOST=example.com MG_LETSENCRYPT_ENABLED=false $0

Required:
  MG_PUBLIC_HOST             Public DNS name that points to this Docker host.
  MG_LETSENCRYPT_EMAIL       Email address for Let's Encrypt notices when
                             MG_LETSENCRYPT_ENABLED=true.

Optional:
  MG_LETSENCRYPT_ENABLED     true by default. Set false to use the fallback
                             Nginx certificate and comment out Let's Encrypt
                             cert/key paths in docker/.env.
  MG_LETSENCRYPT_STAGING     true by default. Set false for production certs.
  MG_LETSENCRYPT_FORCE_RENEWAL
                             false by default. Set true to replace an existing cert.
  DOCKER_PROJECT             Compose project name. Defaults to magistrala.
  MG_LETSENCRYPT_TIMEOUT_SECONDS
                             Wait time for certificate files. Defaults to 180.
EOF
}

if [ -z "$HOST" ]; then
	usage >&2
	exit 2
fi

case "$LETSENCRYPT_ENABLED" in
	true|false)
		;;
	*)
		echo "MG_LETSENCRYPT_ENABLED must be true or false." >&2
		exit 2
		;;
esac

if [ "$LETSENCRYPT_ENABLED" = "true" ] && [ -z "$EMAIL" ]; then
	usage >&2
	exit 2
fi

if [ "$LETSENCRYPT_ENABLED" = "true" ] && [ "$HOST" = "localhost" ]; then
	echo "MG_PUBLIC_HOST must be a public DNS name, not localhost." >&2
	exit 2
fi

if [ ! -f "$ENV_FILE" ]; then
	echo "Missing $ENV_FILE" >&2
	exit 1
fi

set_env() {
	key=$1
	value=$2
	tmp=$(mktemp)
	awk -v key="$key" -v value="$value" '
		BEGIN { done = 0 }
		!done && (index($0, key "=") == 1 || index($0, "#" key "=") == 1) {
			print key "=" value
			done = 1
			next
		}
		{ print }
		END {
			if (!done) {
				print key "=" value
			}
		}
	' "$ENV_FILE" > "$tmp"
	mv "$tmp" "$ENV_FILE"
}

comment_env() {
	key=$1
	value=$2
	tmp=$(mktemp)
	awk -v key="$key" -v value="$value" '
		BEGIN { done = 0 }
		!done && (index($0, key "=") == 1 || index($0, "# " key "=") == 1 || index($0, "#" key "=") == 1) {
			print "# " key "=" value
			done = 1
			next
		}
		{ print }
		END {
			if (!done) {
				print "# " key "=" value
			}
		}
	' "$ENV_FILE" > "$tmp"
	mv "$tmp" "$ENV_FILE"
}

compose() {
	docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT" "$@"
}

wait_for_nginx_http() {
	if ! command -v curl >/dev/null 2>&1; then
		return 0
	fi

	elapsed=0
	while [ "$elapsed" -lt "$TIMEOUT_SECONDS" ]; do
		status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 \
			http://127.0.0.1/.well-known/acme-challenge/magistrala-tls-probe 2>/dev/null || true)
		case "$status" in
			200|301|302|307|308|404)
				return 0
				;;
		esac
		sleep 2
		elapsed=$((elapsed + 2))
	done

	echo "Timed out waiting for Nginx to accept HTTP traffic." >&2
	docker logs --tail 80 magistrala-nginx >&2 || true
	exit 1
}

cert_path="./ssl/letsencrypt/live/$HOST/fullchain.pem"
key_path="./ssl/letsencrypt/live/$HOST/privkey.pem"
cert_file="$ROOT_DIR/docker/ssl/letsencrypt/live/$HOST/fullchain.pem"
key_file="$ROOT_DIR/docker/ssl/letsencrypt/live/$HOST/privkey.pem"

if [ "$LETSENCRYPT_ENABLED" = "false" ]; then
	FORCE_RENEWAL=false
fi

if [ "$LETSENCRYPT_ENABLED" = "true" ] && [ "$STAGING" = "false" ] && [ -f "$cert_file" ]; then
	if openssl x509 -in "$cert_file" -noout -issuer 2>/dev/null | grep -q "STAGING"; then
		FORCE_RENEWAL=true
	fi
fi

echo "Configuring docker/.env for $HOST"
set_env MG_RELEASE_TAG latest
set_env MG_PUBLIC_HOST "$HOST"
set_env MG_UI_HOST "${MG_UI_HOST:-ui}"
set_env MG_LETSENCRYPT_ENABLED "$LETSENCRYPT_ENABLED"
set_env MG_LETSENCRYPT_EMAIL "$EMAIL"
set_env MG_LETSENCRYPT_STAGING "$STAGING"
set_env MG_LETSENCRYPT_FORCE_RENEWAL "$FORCE_RENEWAL"
set_env MG_NGINX_SERVER_NAME "$HOST"
comment_env MG_NGINX_SERVER_CERT "$cert_path"
comment_env MG_NGINX_SERVER_KEY "$key_path"
set_env MG_UI_DOCKER_ACCEPT_EULA yes

set_env MG_OAUTH_UI_REDIRECT_URL "https://$HOST/api/auth/token"
set_env MG_OAUTH_UI_ERROR_URL "https://$HOST/login"
set_env MG_PASSWORD_RESET_URL_PREFIX "https://$HOST/password-reset"
set_env MG_VERIFICATION_URL_PREFIX "https://$HOST/verify-email"
set_env MG_GOOGLE_REDIRECT_URL "https://$HOST/oauth/callback/google"
set_env NEXTAUTH_URL "https://$HOST"
set_env MG_HOST_URL "https://$HOST"
set_env MG_UI_BASEURL "https://$HOST"
set_env MG_UI_CLI_MQTT_HOST "$HOST"
set_env MG_UI_CLI_WS_URL "wss://$HOST/mqtt"
set_env MG_UI_CLI_COAP_HOST "$HOST"
set_env MG_UI_CLI_HTTP_URL "https://$HOST/http"

mkdir -p "$ROOT_DIR/docker/ssl/letsencrypt" "$ROOT_DIR/docker/ssl/certbot-www"

if [ "$LETSENCRYPT_ENABLED" = "false" ]; then
	echo "Starting Magistrala with the fallback Nginx certificate"
	MG_UI_DOCKER_ACCEPT_EULA=yes compose up -d
	MG_UI_DOCKER_ACCEPT_EULA=yes COMPOSE_PROFILES=letsencrypt compose stop certbot >/dev/null 2>&1 || true
	echo "Let's Encrypt disabled. Nginx cert/key paths are commented in docker/.env."
	echo "Fallback TLS setup complete: https://$HOST/"
	exit 0
fi

echo "Starting Magistrala with the fallback Nginx certificate"
MG_UI_DOCKER_ACCEPT_EULA=yes compose up -d
wait_for_nginx_http

echo "Requesting Let's Encrypt certificate for $HOST"
MG_UI_DOCKER_ACCEPT_EULA=yes COMPOSE_PROFILES=letsencrypt compose up -d --force-recreate certbot

elapsed=0
while [ "$elapsed" -lt "$TIMEOUT_SECONDS" ]; do
	if [ -s "$cert_file" ] && [ -s "$key_file" ]; then
		break
	fi
	sleep 2
	elapsed=$((elapsed + 2))
done

if [ ! -s "$cert_file" ] || [ ! -s "$key_file" ]; then
	echo "Timed out waiting for Let's Encrypt certificate files." >&2
	docker logs --tail 80 magistrala-certbot >&2 || true
	exit 1
fi

echo "Switching Nginx to the issued certificate"
set_env MG_NGINX_SERVER_CERT "$cert_path"
set_env MG_NGINX_SERVER_KEY "$key_path"
set_env MG_LETSENCRYPT_FORCE_RENEWAL false

MG_UI_DOCKER_ACCEPT_EULA=yes compose up -d --force-recreate nginx

echo "TLS setup complete: https://$HOST/"
