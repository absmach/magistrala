#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# default env file path
env_file="docker/.env"

SKIP_ENABLE_APP_ROLE=""

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --env-file)
            if [[ -z "${2:-}" ]]; then
                echo "Error: --env-file requires a non-empty option argument."
                exit 1
            fi
            env_file="$2"
            if [[ ! -f "$env_file" ]]; then
                echo "Error: .env file not found at $env_file"
                exit 1
            fi
            shift
            ;;
        --skip-enable-approle)
            SKIP_ENABLE_APP_ROLE="true"
            ;;
        *)
            echo "Unknown parameter passed: $1"
            exit 1
            ;;
    esac
    shift
done

readDotEnv() {
    set -o allexport
    source "$env_file"
    set +o allexport
}

source "$scriptdir/vault_cmd.sh"

vaultCreatePolicyFile() {
    envsubst '
    ${SMQ_VAULT_PKI_INT_PATH}
    ${SMQ_VAULT_PKI_INT_CLIENTS_CERTS_ROLE_NAME}
    ' < "$scriptdir/supermq_clients_certs_issue.template.hcl" > "$scriptdir/supermq_clients_certs_issue.hcl"
}

vaultCreatePolicy() {
    echo "Creating new policy for AppRole"
    if is_container_running "supermq-vault"; then
        docker cp "$scriptdir/supermq_clients_certs_issue.hcl" supermq-vault:/vault/supermq_clients_certs_issue.hcl
        vault policy write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} supermq_clients_certs_issue /vault/supermq_clients_certs_issue.hcl
    else
        vault policy write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} supermq_clients_certs_issue "$scriptdir/supermq_clients_certs_issue.hcl"
    fi
}

vaultEnableAppRole() {
    if [[ "$SKIP_ENABLE_APP_ROLE" == "true" ]]; then
        echo "Skipping Enable AppRole"
    else
        echo "Enabling AppRole"
        vault auth enable -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} approle
    fi
}

vaultDeleteRole() {
    echo "Deleting old AppRole"
    vault delete -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/role/supermq_clients_certs_issuer
}

vaultCreateRole() {
    echo "Creating new AppRole"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/role/supermq_clients_certs_issuer \
    token_policies=supermq_clients_certs_issue secret_id_num_uses=0 \
    secret_id_ttl=0 token_ttl=1h token_max_ttl=3h token_num_uses=0
}

vaultWriteCustomRoleID() {
    echo "Writing custom role id"
    vault read -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/role/supermq_clients_certs_issuer/role-id
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/role/supermq_clients_certs_issuer/role-id role_id=${SMQ_VAULT_CLIENTS_CERTS_ISSUER_ROLEID}
}

vaultWriteCustomSecret() {
    echo "Writing custom secret"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -f auth/approle/role/supermq_clients_certs_issuer/secret-id
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/role/supermq_clients_certs_issuer/custom-secret-id secret_id=${SMQ_VAULT_CLIENTS_CERTS_ISSUER_SECRET} num_uses=0 ttl=0
}

vaultTestRoleLogin() {
    echo "Testing custom roleid secret by logging in"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} auth/approle/login \
        role_id=${SMQ_VAULT_CLIENTS_CERTS_ISSUER_ROLEID} \
        secret_id=${SMQ_VAULT_CLIENTS_CERTS_ISSUER_SECRET}
}

if ! command -v jq &> /dev/null; then
    echo "jq command could not be found, please install it and try again."
    exit 1
fi

readDotEnv

vault login -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_TOKEN}

vaultCreatePolicyFile
vaultCreatePolicy
vaultEnableAppRole
vaultDeleteRole
vaultCreateRole
vaultWriteCustomRoleID
vaultWriteCustomSecret
vaultTestRoleLogin

exit 0
