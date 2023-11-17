#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAGISTRALA_DIR=$scriptdir/../../../

readDotEnv() {
    set -o allexport
    source $MAGISTRALA_DIR/docker/.env
    set +o allexport
}

vault() {
    docker exec -it magistrala-vault vault "$@"
}

readDotEnv

vault operator unseal ${MG_VAULT_UNSEAL_KEY_1}
vault operator unseal ${MG_VAULT_UNSEAL_KEY_2}
vault operator unseal ${MG_VAULT_UNSEAL_KEY_3}
