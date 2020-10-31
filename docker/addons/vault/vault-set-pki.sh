#!/bin/bash
set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAINFLUX_DIR=$scriptdir/../../../

cd $scriptdir

readDotEnv() {
    set -o allexport
    source $MAINFLUX_DIR/.env
    set +o allexport
}

vault() {
    docker exec -it mainflux-vault vault "$@"
}

vaultEnablePKI() {
    vault secrets enable -path pki_${MF_VAULT_CA_NAME} pki
    vault secrets tune -max-lease-ttl=87600h pki_${MF_VAULT_CA_NAME}
}

vaultAddRoleToSecret() {
    vault write pki_${MF_VAULT_CA_NAME}/roles/${MF_VAULT_CA_NAME} \
        allow_any_name=true \
        max_ttl="4300h" \
        default_ttl="4300h" \
        generate_lease=true
}

vaultGenerateRootCACertificate() {
    echo "Generate root CA certificate"
    vault write -format=json pki_${MF_VAULT_CA_NAME}/root/generate/exported \
        common_name="\"$MF_VAULT_CA_DOMAIN_NAME CA Root\"" \
        ou="\"$MF_VAULT_CA_OU\""\
        organization="\"$MF_VAULT_CA_ORG\"" \
        country="\"$MF_VAULT_CA_COUNTRY\"" \
        locality="\"$MF_VAULT_CA_LOC\"" \
        ttl=87600h | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_NAME}_ca.crt) \
                         >(jq -r .data.issuing_ca  >data/${MF_VAULT_CA_NAME}_issuing_ca.crt) \
                         >(jq -r .data.private_key >data/${MF_VAULT_CA_NAME}_ca.key)
}

vaultGenerateIntermediateCAPKI() {
    echo "Generate Intermediate CA PKI"
    export NAME_PKI_INT_PATH="pki_int_$MF_VAULT_CA_NAME"
    vault secrets enable -path=${NAME_PKI_INT_PATH} pki
    vault secrets tune -max-lease-ttl=43800h ${NAME_PKI_INT_PATH}
}

vaultGenerateIntermediateCSR() {
    echo "Generate intermediate CSR"
    vault write -format=json ${NAME_PKI_INT_PATH}/intermediate/generate/exported \
        common_name="$MF_VAULT_CA_DOMAIN_NAME Intermediate Authority" \
        | tee >(jq -r .data.csr         >data/${MF_VAULT_CA_NAME}_int.csr) \
              >(jq -r .data.private_key >data/${MF_VAULT_CA_NAME}_int.key)
}

vaultSignIntermediateCSR() {
    echo "Sign intermediate CSR"
    docker cp data/${MF_VAULT_CA_NAME}_int.csr mainflux-vault:/vault/${MF_VAULT_CA_NAME}_int.csr
    vault write -format=json pki_${MF_VAULT_CA_NAME}/root/sign-intermediate \
        csr=@/vault/${MF_VAULT_CA_NAME}_int.csr \
        | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_NAME}_int.crt) \
              >(jq -r .data.issuing_ca >data/${MF_VAULT_CA_NAME}_int_issuing_ca.crt)
}

vaultInjectIntermediateCertificate() {
    echo "Inject Intermediate Certificate"
    docker cp data/${MF_VAULT_CA_NAME}_int.crt mainflux-vault:/vault/${MF_VAULT_CA_NAME}_int.crt
    vault write ${NAME_PKI_INT_PATH}/intermediate/set-signed certificate=@/vault/${MF_VAULT_CA_NAME}_int.crt
}

vaultGenerateIntermediateCertificateBundle() {
    echo "Generate intermediate certificate bundle"
    cat data/${MF_VAULT_CA_NAME}_int.crt data/${MF_VAULT_CA_NAME}_ca.crt \
       > data/${MF_VAULT_CA_NAME}_int_bundle.crt
}

vaultSetupIssuingURLs() {
    echo "Setup URLs for CRL and issuing"
    VAULT_ADDR=http://$MF_VAULT_HOST:$MF_VAULT_PORT
    vault write ${NAME_PKI_INT_PATH}/config/urls \
        issuing_certificates="$VAULT_ADDR/v1/${NAME_PKI_INT_PATH}/ca" \
        crl_distribution_points="$VAULT_ADDR/v1/${NAME_PKI_INT_PATH}/crl"
}

vaultSetupCARole() {
    echo "Setup CA role"
    vault write ${NAME_PKI_INT_PATH}/roles/${MF_VAULT_CA_ROLE_NAME} \
        allow_subdomains=true \
        allow_any_name=true \
        max_ttl="720h"
}

vaultGenerateServerCertificate() {
    echo "Generate server certificate"
    vault write -format=json ${NAME_PKI_INT_PATH}/issue/${MF_VAULT_CA_ROLE_NAME} \
        common_name="$MF_VAULT_CA_DOMAIN_NAME" ttl="8670h" \
        | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_DOMAIN_NAME}.crt) \
              >(jq -r .data.private_key >data/${MF_VAULT_CA_DOMAIN_NAME}.key)
}

vaultCleanupFiles() {
    docker exec mainflux-vault sh -c 'rm -rf /vault/*.{crt,csr}'
}

if ! command -v jq &> /dev/null
then
    echo "jq command could not be found, please install it and try again."
    exit
fi

readDotEnv

mkdir -p data

vault login ${MF_VAULT_TOKEN}

vaultEnablePKI
vaultAddRoleToSecret
vaultGenerateRootCACertificate
vaultGenerateIntermediateCAPKI
vaultGenerateIntermediateCSR
vaultSignIntermediateCSR
vaultInjectIntermediateCertificate
vaultGenerateIntermediateCertificateBundle
vaultSetupIssuingURLs
vaultSetupCARole
vaultGenerateServerCertificate
vaultCleanupFiles

echo "Copying certificate files"

cp -v data/${MF_VAULT_CA_DOMAIN_NAME}.crt     ${MAINFLUX_DIR}/docker/ssl/certs/mainflux-server.crt
cp -v data/${MF_VAULT_CA_DOMAIN_NAME}.key     ${MAINFLUX_DIR}/docker/ssl/certs/mainflux-server.key
cp -v data/${MF_VAULT_CA_NAME}_int.key        ${MAINFLUX_DIR}/docker/ssl/certs/ca.key
cp -v data/${MF_VAULT_CA_NAME}_int.crt        ${MAINFLUX_DIR}/docker/ssl/certs/ca.crt
cp -v data/${MF_VAULT_CA_NAME}_int_bundle.crt ${MAINFLUX_DIR}/docker/ssl/bundle.pem

exit 0
