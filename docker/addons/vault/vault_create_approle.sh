#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAGISTRALA_DIR=$scriptdir/../../../

cd $scriptdir

SKIP_ENABLE_APP_ROLE=${1:-}

readDotEnv() {
    set -o allexport
    source $MAGISTRALA_DIR/docker/.env
    set +o allexport
}

source vault_cmd.sh

vaultCreatePolicyFile() {
    envsubst '
    ${MG_VAULT_PKI_INT_PATH}
    ${MG_VAULT_PKI_INT_THINGS_CERTS_ROLE_NAME}
    ' <  magistrala_things_certs_issue.template.hcl >  magistrala_things_certs_issue.hcl
}
vaultCreatePolicy() {
    echo "Creating new policy for AppRole"
    if is_container_running "magistrala-vault"; then
        docker cp magistrala_things_certs_issue.hcl magistrala-vault:/vault/magistrala_things_certs_issue.hcl
        vault policy write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} magistrala_things_certs_issue /vault/magistrala_things_certs_issue.hcl
    else
        vault policy write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} magistrala_things_certs_issue magistrala_things_certs_issue.hcl
    fi
}

vaultEnableAppRole() {
   if [ "$SKIP_ENABLE_APP_ROLE" == "--skip-enable-approle" ]; then
        echo "Skipping Enable AppRole"
    else
        echo "Enabling AppRole"
        vault auth enable -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} approle
    fi
}

vaultDeleteRole() {
    echo "Deleteing old AppRole"
    vault delete -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/role/magistrala_things_certs_issuer
}

vaultCreateRole() {
    echo "Creating new AppRole"
    vault write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/role/magistrala_things_certs_issuer \
    token_policies=magistrala_things_certs_issue  secret_id_num_uses=0 \
    secret_id_ttl=0 token_ttl=1h token_max_ttl=3h  token_num_uses=0
}

vaultWriteCustomRoleID(){
    echo "Writing custom role id"
    vault read -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/role/magistrala_things_certs_issuer/role-id
    vault write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/role/magistrala_things_certs_issuer/role-id role_id=${MG_VAULT_THINGS_CERTS_ISSUER_ROLEID}
}

vaultWriteCustomSecret() {
    echo "Writing custom secret"
    vault write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} -f auth/approle/role/magistrala_things_certs_issuer/secret-id
    vault write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/role/magistrala_things_certs_issuer/custom-secret-id secret_id=${MG_VAULT_THINGS_CERTS_ISSUER_SECRET} num_uses=0 ttl=0
}

vaultTestRoleLogin() {
    echo "Testing custom roleid secret by logging in"
    vault write -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} auth/approle/login \
        role_id=${MG_VAULT_THINGS_CERTS_ISSUER_ROLEID} \
        secret_id=${MG_VAULT_THINGS_CERTS_ISSUER_SECRET}

}
if ! command -v jq &> /dev/null
then
    echo "jq command could not be found, please install it and try again."
    exit
fi

readDotEnv

vault login  -namespace=${MG_VAULT_NAMESPACE} -address=${MG_VAULT_ADDR} ${MG_VAULT_TOKEN}

vaultCreatePolicyFile
vaultCreatePolicy
vaultEnableAppRole
vaultDeleteRole
vaultCreateRole
vaultWriteCustomRoleID
vaultWriteCustomSecret
vaultTestRoleLogin

exit 0
