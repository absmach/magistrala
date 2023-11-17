#!/usr/bin/dumb-init /bin/sh
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

VAULT_CONFIG_DIR=/vault/config

docker-entrypoint.sh server &
VAULT_PID=$!

sleep 2

echo $MG_VAULT_UNSEAL_KEY_1
echo $MG_VAULT_UNSEAL_KEY_2
echo $MG_VAULT_UNSEAL_KEY_3

if [[ ! -z "${MG_VAULT_UNSEAL_KEY_1}" ]] &&
   [[ ! -z "${MG_VAULT_UNSEAL_KEY_2}" ]] &&
   [[ ! -z "${MG_VAULT_UNSEAL_KEY_3}" ]]; then
	echo "Unsealing Vault"
	vault operator unseal ${MG_VAULT_UNSEAL_KEY_1}
	vault operator unseal ${MG_VAULT_UNSEAL_KEY_2}
	vault operator unseal ${MG_VAULT_UNSEAL_KEY_3}
fi

wait $VAULT_PID