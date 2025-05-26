#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

scriptdir="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# edfault env file path
env_file="docker/.env"

SKIP_SERVER_CERT=""

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
        --skip-server-cert)
            SKIP_SERVER_CERT="--skip-server-cert"
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

readDotEnv

server_name="localhost"

# Check if SMQ_NGINX_SERVER_NAME is set or not empty
if [ -n "${SMQ_NGINX_SERVER_NAME:-}" ]; then
    server_name="$SMQ_NGINX_SERVER_NAME"
fi

source "$scriptdir/vault_cmd.sh"

vaultEnablePKI() {
    vault secrets enable -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -path ${SMQ_VAULT_PKI_PATH} pki
    vault secrets tune -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -max-lease-ttl=87600h ${SMQ_VAULT_PKI_PATH}
}

vaultConfigPKIClusterPath() {
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_PATH}/config/cluster aia_path=${SMQ_VAULT_PKI_CLUSTER_AIA_PATH} path=${SMQ_VAULT_PKI_CLUSTER_PATH}
}

vaultConfigPKICrl() {
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_PATH}/config/crl expiry="5m"  ocsp_disable=false ocsp_expiry=0 auto_rebuild=true auto_rebuild_grace_period="2m" enable_delta=true delta_rebuild_interval="1m"
}

vaultAddRoleToSecret() {
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_PATH}/roles/${SMQ_VAULT_PKI_ROLE_NAME} \
        allow_any_name=true \
        max_ttl="8760h" \
        default_ttl="8760h" \
        generate_lease=true
}

vaultGenerateRootCACertificate() {
    echo "Generate root CA certificate"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -format=json ${SMQ_VAULT_PKI_PATH}/root/generate/exported \
        common_name="\"$SMQ_VAULT_PKI_CA_CN\"" \
        ou="\"$SMQ_VAULT_PKI_CA_OU\"" \
        organization="\"$SMQ_VAULT_PKI_CA_O\"" \
        country="\"$SMQ_VAULT_PKI_CA_C\"" \
        locality="\"$SMQ_VAULT_PKI_CA_L\"" \
        province="\"$SMQ_VAULT_PKI_CA_ST\"" \
        street_address="\"$SMQ_VAULT_PKI_CA_ADDR\"" \
        postal_code="\"$SMQ_VAULT_PKI_CA_PO\"" \
        ttl=87600h | tee >(jq -r .data.certificate >"$scriptdir/data/${SMQ_VAULT_PKI_FILE_NAME}_ca.crt") \
                         >(jq -r .data.issuing_ca  >"$scriptdir/data/${SMQ_VAULT_PKI_FILE_NAME}_issuing_ca.crt") \
                         >(jq -r .data.private_key >"$scriptdir/data/${SMQ_VAULT_PKI_FILE_NAME}_ca.key")
}

vaultSetupRootCAIssuingURLs() {
    echo "Setup URLs for CRL and issuing"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_PATH}/config/urls \
        issuing_certificates="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_PATH}/ca" \
        crl_distribution_points="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_PATH}/crl" \
        ocsp_servers="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_PATH}/ocsp" \
        enable_templating=true
}

vaultGenerateIntermediateCAPKI() {
    echo "Generate Intermediate CA PKI"
    vault secrets enable -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR}  -path=${SMQ_VAULT_PKI_INT_PATH} pki
    vault secrets tune -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR}  -max-lease-ttl=43800h ${SMQ_VAULT_PKI_INT_PATH}
}

vaultConfigIntermediatePKIClusterPath() {
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/config/cluster aia_path=${SMQ_VAULT_PKI_INT_CLUSTER_AIA_PATH} path=${SMQ_VAULT_PKI_INT_CLUSTER_PATH}
}

vaultConfigIntermediatePKICrl() {
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/config/crl expiry="5m"  ocsp_disable=false ocsp_expiry=0 auto_rebuild=true auto_rebuild_grace_period="2m" enable_delta=true delta_rebuild_interval="1m"
}

vaultGenerateIntermediateCSR() {
    echo "Generate intermediate CSR"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -format=json  ${SMQ_VAULT_PKI_INT_PATH}/intermediate/generate/exported \
        common_name="\"$SMQ_VAULT_PKI_INT_CA_CN\"" \
        ou="\"$SMQ_VAULT_PKI_INT_CA_OU\""\
        organization="\"$SMQ_VAULT_PKI_INT_CA_O\"" \
        country="\"$SMQ_VAULT_PKI_INT_CA_C\"" \
        locality="\"$SMQ_VAULT_PKI_INT_CA_L\"" \
        province="\"$SMQ_VAULT_PKI_INT_CA_ST\"" \
        street_address="\"$SMQ_VAULT_PKI_INT_CA_ADDR\"" \
        postal_code="\"$SMQ_VAULT_PKI_INT_CA_PO\"" \
        | tee >(jq -r .data.csr         >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.csr") \
              >(jq -r .data.private_key >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.key")
}

vaultSignIntermediateCSR() {
    echo "Sign intermediate CSR"
    if is_container_running "supermq-vault"; then
        docker cp "$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.csr" supermq-vault:/vault/${SMQ_VAULT_PKI_INT_FILE_NAME}.csr
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -format=json  ${SMQ_VAULT_PKI_PATH}/root/sign-intermediate \
            csr=@/vault/${SMQ_VAULT_PKI_INT_FILE_NAME}.csr  ttl="8760h" \
            ou="\"$SMQ_VAULT_PKI_INT_CA_OU\""\
            organization="\"$SMQ_VAULT_PKI_INT_CA_O\"" \
            country="\"$SMQ_VAULT_PKI_INT_CA_C\"" \
            locality="\"$SMQ_VAULT_PKI_INT_CA_L\"" \
            province="\"$SMQ_VAULT_PKI_INT_CA_ST\"" \
            street_address="\"$SMQ_VAULT_PKI_INT_CA_ADDR\"" \
            postal_code="\"$SMQ_VAULT_PKI_INT_CA_PO\"" \
            | tee >(jq -r .data.certificate >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt") \
                >(jq -r .data.issuing_ca >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}_issuing_ca.crt")
    else
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -format=json  ${SMQ_VAULT_PKI_PATH}/root/sign-intermediate \
            csr=@"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.csr"  ttl="8760h" \
            ou="\"$SMQ_VAULT_PKI_INT_CA_OU\""\
            organization="\"$SMQ_VAULT_PKI_INT_CA_O\"" \
            country="\"$SMQ_VAULT_PKI_INT_CA_C\"" \
            locality="\"$SMQ_VAULT_PKI_INT_CA_L\"" \
            province="\"$SMQ_VAULT_PKI_INT_CA_ST\"" \
            street_address="\"$SMQ_VAULT_PKI_INT_CA_ADDR\"" \
            postal_code="\"$SMQ_VAULT_PKI_INT_CA_PO\"" \
            | tee >(jq -r .data.certificate >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt") \
                >(jq -r .data.issuing_ca >"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}_issuing_ca.crt")
    fi
}

vaultInjectIntermediateCertificate() {
    echo "Inject Intermediate Certificate"
    if is_container_running "supermq-vault"; then
        docker cp "$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt" supermq-vault:/vault/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/intermediate/set-signed certificate=@/vault/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt
    else
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/intermediate/set-signed certificate=@"$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt"
    fi
}

vaultGenerateIntermediateCertificateBundle() {
    echo "Generate intermediate certificate bundle"
    cat "$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}.crt" "$scriptdir/data/${SMQ_VAULT_PKI_FILE_NAME}_ca.crt" \
       > "$scriptdir/data/${SMQ_VAULT_PKI_INT_FILE_NAME}_bundle.crt"
}

vaultSetupIntermediateIssuingURLs() {
    echo "Setup URLs for CRL and issuing"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/config/urls \
        issuing_certificates="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_INT_PATH}/ca" \
        crl_distribution_points="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_INT_PATH}/crl" \
        ocsp_servers="{{cluster_aia_path}}/v1/${SMQ_VAULT_PKI_INT_PATH}/ocsp" \
        enable_templating=true
}

vaultSetupServerCertsRole() {
    if [ "$SKIP_SERVER_CERT" == "--skip-server-cert" ]; then
        echo "Skipping server certificate role"
    else
        echo "Setup Server certificate role"
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/roles/${SMQ_VAULT_PKI_INT_SERVER_CERTS_ROLE_NAME} \
            allow_subdomains=true \
            allow_any_name=true \
            max_ttl="4320h"
    fi
}

vaultGenerateServerCertificate() {
    if [ "$SKIP_SERVER_CERT" == "--skip-server-cert" ]; then
        echo "Skipping generate server certificate"
    else
        echo "Generate server certificate"
        vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} -format=json ${SMQ_VAULT_PKI_INT_PATH}/issue/${SMQ_VAULT_PKI_INT_SERVER_CERTS_ROLE_NAME} \
            common_name="$server_name" ttl="4320h" \
            | tee >(jq -r .data.certificate >"$scriptdir/data/${server_name}.crt") \
                >(jq -r .data.private_key >"$scriptdir/data/${server_name}.key")
    fi
}

vaultSetupClientCertsRole() {
    echo "Setup Client Certs role"
    vault write -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_PKI_INT_PATH}/roles/${SMQ_VAULT_PKI_INT_CLIENTS_CERTS_ROLE_NAME} \
        allow_subdomains=true \
        allow_any_name=true \
        max_ttl="2160h"
}

vaultCleanupFiles() {
    if is_container_running "supermq-vault"; then
        docker exec supermq-vault sh -c 'rm -rf /vault/*.{crt,csr}'
    fi
}

if ! command -v jq &> /dev/null; then
    echo "jq command could not be found, please install it and try again."
    exit 1
fi

mkdir -p "$scriptdir/data"

vault login -namespace=${SMQ_VAULT_NAMESPACE} -address=${SMQ_VAULT_ADDR} ${SMQ_VAULT_TOKEN}

vaultEnablePKI
vaultConfigPKIClusterPath
vaultConfigPKICrl
vaultAddRoleToSecret
vaultGenerateRootCACertificate
vaultSetupRootCAIssuingURLs
vaultGenerateIntermediateCAPKI
vaultConfigIntermediatePKIClusterPath
vaultConfigIntermediatePKICrl
vaultGenerateIntermediateCSR
vaultSignIntermediateCSR
vaultInjectIntermediateCertificate
vaultGenerateIntermediateCertificateBundle
vaultSetupIntermediateIssuingURLs
vaultSetupServerCertsRole
vaultGenerateServerCertificate
vaultSetupClientCertsRole
vaultCleanupFiles

exit 0
