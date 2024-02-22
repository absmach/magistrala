#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export MAGISTRALA_DIR=$scriptdir/../../../

cd $scriptdir

readDotEnv() {
    set -o allexport
    source $MAGISTRALA_DIR/docker/.env
    set +o allexport
}

source vault_cmd.sh

readDotEnv

mkdir -p data

vault operator init -address=$MG_VAULT_ADDR 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > data/secrets)
