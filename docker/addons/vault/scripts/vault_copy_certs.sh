#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# default env file path
env_file="docker/.env"

# default certs copy path
certs_copy_path="docker/ssl/certs/"

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --env-file)
            if [[ -z "${2:-}" ]]; then
                echo "Error: --env-file requires a non-empty option argument."
                exit 1
            fi
            env_file="$2"
            if [[ ! -f "$env_file" ]]; then
                echo "Error: .env file not found at $env_file"
                exit 1
            fi
            shift
            ;;
        --certs-copy-path)
            if [[ -z "${2:-}" ]]; then
                echo "Error: --certs-copy-path requires a non-empty option argument."
                exit 1
            fi
            certs_copy_path="$2"
            shift
            ;;
        *)
            echo "Error: Unknown parameter passed: $1"
            exit 1
            ;;
    esac
    shift
done

readDotEnv() {
    set -o allexport
    source "$env_file"
    set +o allexport
}

readDotEnv

server_name="localhost"

# Check if MG_NGINX_SERVER_NAME is set or not empty
if [ -n "${MG_NGINX_SERVER_NAME:-}" ]; then
    server_name="$MG_NGINX_SERVER_NAME"
fi

echo "Copying certificate files to ${certs_copy_path}"

if [ -e "$scriptdir/data/${server_name}.crt" ]; then
    cp -v "$scriptdir/data/${server_name}.crt" "${certs_copy_path}magistrala-server.crt"
else
    echo "${server_name}.crt file not available"
fi

if [ -e "$scriptdir/data/${server_name}.key" ]; then
    cp -v "$scriptdir/data/${server_name}.key" "${certs_copy_path}magistrala-server.key"
else
    echo "${server_name}.key file not available"
fi

if [ -e "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}.key" ]; then
    cp -v "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}.key" "${certs_copy_path}ca.key"
else
    echo "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}.key file not available"
fi

if [ -e "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt" ]; then
    cp -v "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt" "${certs_copy_path}ca.crt"
else
    echo "$scriptdir/data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt file not available"
fi

exit 0
