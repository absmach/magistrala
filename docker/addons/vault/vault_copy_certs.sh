#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAGISTRALA_DIR=$scriptdir/../../../

cd $scriptdir

readDotEnv() {
    set -o allexport
    source $MAGISTRALA_DIR/docker/.env
    set +o allexport
}

readDotEnv

server_name="localhost"

# Check if MG_NGINX_SERVER_NAME is set or not empty
if [ -n "${MG_NGINX_SERVER_NAME:-}" ]; then
    server_name="$MG_NGINX_SERVER_NAME"
fi

echo "Copying certificate files"
cp -v data/${server_name}.crt      ${MAGISTRALA_DIR}/docker/ssl/certs/magistrala-server.crt
cp -v data/${server_name}.key      ${MAGISTRALA_DIR}/docker/ssl/certs/magistrala-server.key
cp -v data/${MG_VAULT_PKI_INT_FILE_NAME}.key    ${MAGISTRALA_DIR}/docker/ssl/certs/ca.key
cp -v data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt     ${MAGISTRALA_DIR}/docker/ssl/certs/ca.crt

exit 0
