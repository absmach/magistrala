#!/usr/bin/dumb-init /bin/sh

VAULT_CONFIG_DIR=/vault/config

docker-entrypoint.sh server &
VAULT_PID=$!

sleep 2

echo $MF_VAULT_UNSEAL_KEY_1
echo $MF_VAULT_UNSEAL_KEY_2
echo $MF_VAULT_UNSEAL_KEY_3

if [[ ! -z "${MF_VAULT_UNSEAL_KEY_1}" ]] &&
   [[ ! -z "${MF_VAULT_UNSEAL_KEY_2}" ]] &&
   [[ ! -z "${MF_VAULT_UNSEAL_KEY_3}" ]]; then
	echo "Unsealing Vault"
	vault operator unseal ${MF_VAULT_UNSEAL_KEY_1}
	vault operator unseal ${MF_VAULT_UNSEAL_KEY_2}
	vault operator unseal ${MF_VAULT_UNSEAL_KEY_3}
fi

wait $VAULT_PID