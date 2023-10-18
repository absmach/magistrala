# Certs Service

Issues certificates for things. `Certs` service can create certificates to be used when `Mainflux` is deployed to support mTLS.
Certificate service can create certificates using PKI mode - where certificates issued by PKI, when you deploy `Vault` as PKI certificate management `cert` service will proxy requests to `Vault` previously checking access rights and saving info on successfully created certificate.

## PKI mode

When `MF_CERTS_VAULT_HOST` is set it is presumed that `Vault` is installed and `certs` service will issue certificates using `Vault` API.
First you'll need to set up `Vault`.
To setup `Vault` follow steps in [Build Your Own Certificate Authority (CA)](https://learn.hashicorp.com/tutorials/vault/pki-engine).

To setup certs service with `Vault` following environment variables must be set:

```bash
MF_CERTS_VAULT_HOST=vault-domain.com
MF_CERTS_VAULT_PKI_PATH=<vault_pki_path>
MF_CERTS_VAULT_ROLE=<vault_role>
MF_CERTS_VAULT_TOKEN=<vault_acces_token>
```

For lab purposes you can use docker-compose and script for setting up PKI in [https://github.com/mteodor/vault](https://github.com/mteodor/vault)

The certificates can also be revoked using `certs` service. To revoke a certificate you need to provide `thing_id` of the thing for which the certificate was issued.

```bash
curl -s -S -X DELETE http://localhost:9019/certs/revoke -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json'   -d '{"thing_id":"c30b8842-507c-4bcd-973c-74008cef3be5"}'
```
