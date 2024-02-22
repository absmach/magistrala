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

if [ -e "data/${server_name}.crt" ]; then
    cp -v data/${server_name}.crt      ${MAGISTRALA_DIR}/docker/ssl/certs/magistrala-server.crt
else
    echo "${server_name}.crt file not available"
fi

if [ -e "data/${server_name}.key" ]; then
    cp -v data/${server_name}.key      ${MAGISTRALA_DIR}/docker/ssl/certs/magistrala-server.key
else
    echo "${server_name}.key file not available"
fi

if [ -e "data/${MG_VAULT_PKI_INT_FILE_NAME}.key" ]; then
    cp -v data/${MG_VAULT_PKI_INT_FILE_NAME}.key    ${MAGISTRALA_DIR}/docker/ssl/certs/ca.key
else
    echo "data/${MG_VAULT_PKI_INT_FILE_NAME}.key file not available"
fi

if [ -e "data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt" ]; then
    cp -v data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt     ${MAGISTRALA_DIR}/docker/ssl/certs/ca.crt
else
    echo "data/${MG_VAULT_PKI_INT_FILE_NAME}_bundle.crt file not available"
fi

exit 0
