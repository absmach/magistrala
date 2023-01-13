#!/usr/bin/bash
set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export MAINFLUX_DIR=$scriptdir/../../../

write_env() {
    sed -i "s,MF_VAULT_UNSEAL_KEY_1=.*,MF_VAULT_UNSEAL_KEY_1=$(awk -F ": " '$1 == "Unseal Key 1" {print $2}' data/secrets)," $MAINFLUX_DIR/docker/.env
    sed -i "s,MF_VAULT_UNSEAL_KEY_2=.*,MF_VAULT_UNSEAL_KEY_2=$(awk -F ": " '$1 == "Unseal Key 2" {print $2}' data/secrets)," $MAINFLUX_DIR/docker/.env
    sed -i "s,MF_VAULT_UNSEAL_KEY_3=.*,MF_VAULT_UNSEAL_KEY_3=$(awk -F ": " '$1 == "Unseal Key 3" {print $2}' data/secrets)," $MAINFLUX_DIR/docker/.env
    sed -i "s,MF_VAULT_TOKEN=.*,MF_VAULT_TOKEN=$(awk -F ": " '$1 == "Initial Root Token" {print $2}' data/secrets)," $MAINFLUX_DIR/docker/.env
}
vault() {
    docker exec -it mainflux-vault vault "$@"
}

mkdir -p data

vault operator init 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > data/secrets)

write_env
