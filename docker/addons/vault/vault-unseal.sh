#!/usr/bin/bash
set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAINFLUX_DIR=$scriptdir/../../../

readDotEnv() {
    set -o allexport
    source $MAINFLUX_DIR/docker/.env
    set +o allexport
}

vault() {
    docker exec -it mainflux-vault vault "$@"
}

readDotEnv

vault operator unseal ${MF_VAULT_UNSEAL_KEY_1}
vault operator unseal ${MF_VAULT_UNSEAL_KEY_2}
vault operator unseal ${MF_VAULT_UNSEAL_KEY_3}
