#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export MAGISTRALA_DIR=$scriptdir/../../../

cd $scriptdir

vault() {
    docker exec -it magistrala-vault vault "$@"
}

mkdir -p data

vault operator init 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > data/secrets)
