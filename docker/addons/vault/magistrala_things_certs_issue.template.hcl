
# Allow issue certificate with role with default issuer from Intermediate PKI
path "${MG_VAULT_PKI_INT_PATH}/issue/${MG_VAULT_PKI_INT_THINGS_CERTS_ROLE_NAME}" {
   capabilities = ["create",  "update"]
}

## Revole certificate from Intermediate PKI
path "${MG_VAULT_PKI_INT_PATH}/revoke" {
  capabilities = ["create",  "update"]
}

## List Revoked Certificates from Intermediate PKI
path "${MG_VAULT_PKI_INT_PATH}/certs/revoked" {
  capabilities = ["list"]
}


## List Certificates from Intermediate PKI
path "${MG_VAULT_PKI_INT_PATH}/certs" {
  capabilities = ["list"]
}

## Read Certificate from Intermediate PKI
path "${MG_VAULT_PKI_INT_PATH}/cert/+" {
  capabilities = ["read"]
}
path "${MG_VAULT_PKI_INT_PATH}/cert/+/raw" {
  capabilities = ["read"]
}
path "${MG_VAULT_PKI_INT_PATH}/cert/+/raw/pem" {
  capabilities = ["read"]
}
