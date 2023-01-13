#!/usr/bin/bash
set -euo pipefail

scriptdir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export MAINFLUX_DIR=$scriptdir/../../../

cd $scriptdir

readDotEnv() {
    set -o allexport
    source $MAINFLUX_DIR/docker/.env
    set +o allexport
}

vault() {
    docker exec -it mainflux-vault vault "$@"
}

vaultEnablePKI() {
    vault secrets enable -path ${MF_VAULT_PKI_PATH} pki
    vault secrets tune -max-lease-ttl=87600h ${MF_VAULT_PKI_PATH}
}

vaultAddRoleToSecret() {
    vault write ${MF_VAULT_PKI_PATH}/roles/${MF_VAULT_CA_NAME} \
        allow_any_name=true \
        max_ttl="4300h" \
        default_ttl="4300h" \
        generate_lease=true
}

vaultGenerateRootCACertificate() {
    echo "Generate root CA certificate"
    vault write -format=json ${MF_VAULT_PKI_PATH}/root/generate/exported \
        common_name="\"$MF_VAULT_CA_CN CA Root\"" \
        ou="\"$MF_VAULT_CA_OU\""\
        organization="\"$MF_VAULT_CA_O\"" \
        country="\"$MF_VAULT_CA_C\"" \
        locality="\"$MF_VAULT_CA_L\"" \
        ttl=87600h | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_NAME}_ca.crt) \
                         >(jq -r .data.issuing_ca  >data/${MF_VAULT_CA_NAME}_issuing_ca.crt) \
                         >(jq -r .data.private_key >data/${MF_VAULT_CA_NAME}_ca.key)
}

vaultGenerateIntermediateCAPKI() {
    echo "Generate Intermediate CA PKI"
    vault secrets enable -path=${MF_VAULT_PKI_INT_PATH} pki
    vault secrets tune -max-lease-ttl=43800h ${MF_VAULT_PKI_INT_PATH}
}

vaultGenerateIntermediateCSR() {
    echo "Generate intermediate CSR"
    vault write -format=json ${MF_VAULT_PKI_INT_PATH}/intermediate/generate/exported \
        common_name="$MF_VAULT_CA_CN Intermediate Authority" \
        | tee >(jq -r .data.csr         >data/${MF_VAULT_CA_NAME}_int.csr) \
              >(jq -r .data.private_key >data/${MF_VAULT_CA_NAME}_int.key)
}

vaultSignIntermediateCSR() {
    echo "Sign intermediate CSR"
    docker cp data/${MF_VAULT_CA_NAME}_int.csr mainflux-vault:/vault/${MF_VAULT_CA_NAME}_int.csr
    vault write -format=json ${MF_VAULT_PKI_PATH}/root/sign-intermediate \
        csr=@/vault/${MF_VAULT_CA_NAME}_int.csr \
        | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_NAME}_int.crt) \
              >(jq -r .data.issuing_ca >data/${MF_VAULT_CA_NAME}_int_issuing_ca.crt)
}

vaultInjectIntermediateCertificate() {
    echo "Inject Intermediate Certificate"
    docker cp data/${MF_VAULT_CA_NAME}_int.crt mainflux-vault:/vault/${MF_VAULT_CA_NAME}_int.crt
    vault write ${MF_VAULT_PKI_INT_PATH}/intermediate/set-signed certificate=@/vault/${MF_VAULT_CA_NAME}_int.crt
}

vaultGenerateIntermediateCertificateBundle() {
    echo "Generate intermediate certificate bundle"
    cat data/${MF_VAULT_CA_NAME}_int.crt data/${MF_VAULT_CA_NAME}_ca.crt \
       > data/${MF_VAULT_CA_NAME}_int_bundle.crt
}

vaultSetupIssuingURLs() {
    echo "Setup URLs for CRL and issuing"
    VAULT_ADDR=http://$MF_VAULT_HOST:$MF_VAULT_PORT
    vault write ${MF_VAULT_PKI_INT_PATH}/config/urls \
        issuing_certificates="$VAULT_ADDR/v1/${MF_VAULT_PKI_INT_PATH}/ca" \
        crl_distribution_points="$VAULT_ADDR/v1/${MF_VAULT_PKI_INT_PATH}/crl"
}

vaultSetupCARole() {
    echo "Setup CA role"
    vault write ${MF_VAULT_PKI_INT_PATH}/roles/${MF_VAULT_CA_ROLE_NAME} \
        allow_subdomains=true \
        allow_any_name=true \
        max_ttl="720h"
}

vaultGenerateServerCertificate() {
    echo "Generate server certificate"
    vault write -format=json ${MF_VAULT_PKI_INT_PATH}/issue/${MF_VAULT_CA_ROLE_NAME} \
        common_name="$MF_VAULT_CA_CN" ttl="8670h" \
        | tee >(jq -r .data.certificate >data/${MF_VAULT_CA_CN}.crt) \
              >(jq -r .data.private_key >data/${MF_VAULT_CA_CN}.key)
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

cp -v data/${MF_VAULT_CA_CN}.crt     ${MAINFLUX_DIR}/docker/ssl/certs/mainflux-server.crt
cp -v data/${MF_VAULT_CA_CN}.key     ${MAINFLUX_DIR}/docker/ssl/certs/mainflux-server.key
cp -v data/${MF_VAULT_CA_NAME}_int.key        ${MAINFLUX_DIR}/docker/ssl/certs/ca.key
cp -v data/${MF_VAULT_CA_NAME}_int.crt        ${MAINFLUX_DIR}/docker/ssl/certs/ca.crt
cp -v data/${MF_VAULT_CA_NAME}_int_bundle.crt ${MAINFLUX_DIR}/docker/ssl/bundle.pem

exit 0
