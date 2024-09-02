#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# default env file path
env_file="docker/.env"

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
        *)
            echo "Unknown parameter passed: $1"
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

source "$scriptdir/vault_cmd.sh"

readDotEnv

mkdir -p "$scriptdir/data"

vault operator init -address="$MG_VAULT_ADDR" 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > "$scriptdir/data/secrets")
