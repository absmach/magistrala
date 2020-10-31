#!/bin/bash
set -euo pipefail

vault() {
    docker exec -it mainflux-vault vault "$@"
}

mkdir -p data

vault operator init 2>&1 | tee >(sed -r 's/\x1b\[[0-9;]*m//g' > data/secrets)
