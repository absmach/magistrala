#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export MAGISTRALA_DIR=$scriptdir/../../../

write_env() {
    sed -i "s,MG_VAULT_UNSEAL_KEY_1=.*,MG_VAULT_UNSEAL_KEY_1=$(awk -F ": " '$1 == "Unseal Key 1" {print $2}' data/secrets)," $MAGISTRALA_DIR/docker/.env
    sed -i "s,MG_VAULT_UNSEAL_KEY_2=.*,MG_VAULT_UNSEAL_KEY_2=$(awk -F ": " '$1 == "Unseal Key 2" {print $2}' data/secrets)," $MAGISTRALA_DIR/docker/.env
    sed -i "s,MG_VAULT_UNSEAL_KEY_3=.*,MG_VAULT_UNSEAL_KEY_3=$(awk -F ": " '$1 == "Unseal Key 3" {print $2}' data/secrets)," $MAGISTRALA_DIR/docker/.env
    sed -i "s,MG_VAULT_TOKEN=.*,MG_VAULT_TOKEN=$(awk -F ": " '$1 == "Initial Root Token" {print $2}' data/secrets)," $MAGISTRALA_DIR/docker/.env
}
vault() {
    docker exec -it magistrala-vault vault "$@"
}

mkdir -p data

vault operator init 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > data/secrets)

write_env
