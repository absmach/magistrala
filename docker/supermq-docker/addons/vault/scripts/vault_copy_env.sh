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

write_env() {
    if [ -e "$scriptdir/data/secrets" ]; then
        if [[ "$(uname)" == "Darwin" ]]; then
            SED_OPT=(-i '')
        else
            SED_OPT=(-i)
        fi
    
        sed "${SED_OPT[@]}" "s,SMQ_VAULT_UNSEAL_KEY_1=.*,SMQ_VAULT_UNSEAL_KEY_1=$(awk -F ': ' '$1 == "Unseal Key 1" {print $2}' "$scriptdir/data/secrets")," "$env_file"
        sed "${SED_OPT[@]}" "s,SMQ_VAULT_UNSEAL_KEY_2=.*,SMQ_VAULT_UNSEAL_KEY_2=$(awk -F ': ' '$1 == "Unseal Key 2" {print $2}' "$scriptdir/data/secrets")," "$env_file"
        sed "${SED_OPT[@]}" "s,SMQ_VAULT_UNSEAL_KEY_3=.*,SMQ_VAULT_UNSEAL_KEY_3=$(awk -F ': ' '$1 == "Unseal Key 3" {print $2}' "$scriptdir/data/secrets")," "$env_file"
        sed "${SED_OPT[@]}" "s,SMQ_VAULT_TOKEN=.*,SMQ_VAULT_TOKEN=$(awk -F ': ' '$1 == "Initial Root Token" {print $2}' "$scriptdir/data/secrets")," "$env_file"
        echo "Vault environment variables are set successfully in $env_file"
    else
        echo "Error: Source file '$scriptdir/data/secrets' not found."
    fi
}

write_env
