# Vault

This is Vault service deployment to be used with Magistrala.

When the Vault service is started, some initialization steps need to be done to set clients up.

## Configuration

| Variable                                | Description                                                                   | Default                               |
| :-------------------------------------- | ----------------------------------------------------------------------------- | ------------------------------------- |
| MG_VAULT_ADDR                           | Vault Address                                                                 | http://vault:8200                     |
| MG_VAULT_UNSEAL_KEY_1                   | Vault unseal key                                                              | ""                                    |
| MG_VAULT_UNSEAL_KEY_2                   | Vault unseal key                                                              | ""                                    |
| MG_VAULT_UNSEAL_KEY_3                   | Vault unseal key                                                              | ""                                    |
| MG_VAULT_TOKEN                          | Vault cli access token                                                        | ""                                    |
| MG_VAULT_PKI_PATH                       | Vault secrets engine path for Root CA                                         | pki                                   |
| MG_VAULT_PKI_ROLE_NAME                  | Vault Root CA role name to issue intermediate CA                              | magistrala_int_ca                     |
| MG_VAULT_PKI_FILE_NAME                  | Root CA Certificates name used by`vault_set_pki.sh`                           | mg_root                               |
| MG_VAULT_PKI_CA_CN                      | Common name used for Root CA creation by`vault_set_pki.sh`                    | Magistrala Root Certificate Authority |
| MG_VAULT_PKI_CA_OU                      | Organization unit used for Root CA creation by`vault_set_pki.sh`              | Magistrala                            |
| MG_VAULT_PKI_CA_O                       | Organization used for Root CA creation by`vault_set_pki.sh`                   | Magistrala                            |
| MG_VAULT_PKI_CA_C                       | Country used for Root CA creation by`vault_set_pki.sh`                        | FRANCE                                |
| MG_VAULT_PKI_CA_L                       | Location used for Root CA creation by`vault_set_pki.sh`                       | PARIS                                 |
| MG_VAULT_PKI_CA_ST                      | State or Provisions used for Root CA creation by`vault_set_pki.sh`            | PARIS                                 |
| MG_VAULT_PKI_CA_ADDR                    | Address used for Root CA creation by`vault_set_pki.sh`                        | 5 Av. Anatole                         |
| MG_VAULT_PKI_CA_PO                      | Postal code used for Root CA creation by`vault_set_pki.sh`                    | 75007                                 |
| MG_VAULT_PKI_CLUSTER_PATH               | Vault Root CA Cluster Path                                                    | http://localhost                      |
| MG_VAULT_PKI_CLUSTER_AIA_PATH           | Vault Root CA Cluster AIA Path                                                | http://localhost                      |
| MG_VAULT_PKI_INT_PATH                   | Vault secrets engine path for Intermediate CA                                 | pki_int                               |
| MG_VAULT_PKI_INT_SERVER_CERTS_ROLE_NAME | Vault Intermediate CA role name to issue server certificate                   | magistrala_server_certs               |
| MG_VAULT_PKI_INT_CLIENTS_CERTS_ROLE_NAME | Vault Intermediate CA role name to issue Clients certificates                  | magistrala_clients_certs               |
| MG_VAULT_PKI_INT_FILE_NAME              | Intermediate CA Certificates name used by`vault_set_pki.sh`                   | mg_root                               |
| MG_VAULT_PKI_INT_CA_CN                  | Common name used for Intermediate CA creation by`vault_set_pki.sh`            | Magistrala Root Certificate Authority |
| MG_VAULT_PKI_INT_CA_OU                  | Organization unit used for Root CA creation by`vault_set_pki.sh`              | Magistrala                            |
| MG_VAULT_PKI_INT_CA_O                   | Organization used for Intermediate CA creation by`vault_set_pki.sh`           | Magistrala                            |
| MG_VAULT_PKI_INT_CA_C                   | Country used for Intermediate CA creation by`vault_set_pki.sh`                | FRANCE                                |
| MG_VAULT_PKI_INT_CA_L                   | Location used for Intermediate CA creation by`vault_set_pki.sh`               | PARIS                                 |
| MG_VAULT_PKI_INT_CA_ST                  | State or Provisions used for Intermediate CA creation by`vault_set_pki.sh`    | PARIS                                 |
| MG_VAULT_PKI_INT_CA_ADDR                | Address used for Intermediate CA creation by`vault_set_pki.sh`                | 5 Av. Anatole                         |
| MG_VAULT_PKI_INT_CA_PO                  | Postal code used for Intermediate CA creation by`vault_set_pki.sh`            | 75007                                 |
| MG_VAULT_PKI_INT_CLUSTER_PATH           | Vault Intermediate CA Cluster Path                                            | http://localhost                      |
| MG_VAULT_PKI_INT_CLUSTER_AIA_PATH       | Vault Intermediate CA Cluster AIA Path                                        | http://localhost                      |
| MG_VAULT_CLIENTS_CERTS_ISSUER_ROLEID     | Vault Intermediate CA Clients Certificate issuer AppRole authentication RoleID | magistrala                            |
| MG_VAULT_CLIENTS_CERTS_ISSUER_SECRET     | Vault Intermediate CA Clients Certificate issuer AppRole authentication Secret | magistrala                            |

## Setup

The following scripts are provided, which work on the running Vault service from within the `docker/addons/vault/scripts` directory.

### 1. `vault_init.sh`

Calls `vault operator init` to perform the initial vault initialization and generates a `docker/addons/vault/scripts/data/secrets` file which contains the Vault unseal keys and root tokens.

### 2. `vault_copy_env.sh`

After the initial setup, the Vault-related environment variables (`MG_VAULT_TOKEN`, `MG_VAULT_UNSEAL_KEY_1`, `MG_VAULT_UNSEAL_KEY_2`, `MG_VAULT_UNSEAL_KEY_3`) need to be updated in the `.env` file.

The `vault_copy_env.sh` script automatically retrieves these values from the `docker/addons/vault/scripts/data/secrets` file and updates the corresponding environment variables in your `.env` file.

Example:

```sh
Vault environment variables have been successfully set in ~/magistrala/docker/.env
```

### 3. `vault_unseal.sh`

This can be run after the initialization to unseal Vault, which is necessary for it to be used to store and/or get secrets.

This can be used if you don't want to restart the service.

The unseal environment variables need to be set in `.env` for the script to work (`MG_VAULT_TOKEN`,`MG_VAULT_UNSEAL_KEY_1`, `MG_VAULT_UNSEAL_KEY_2`, `MG_VAULT_UNSEAL_KEY_3`).

This script should not be necessary to run after the initial setup, since the Vault service unseals itself when starting the container.

Example output:

```bash
Key                Value
---                -----
Seal Type          shamir
Initialized        true
Sealed             true
Total Shares       5
Threshold          3
Unseal Progress    1/3
Unseal Nonce       4c248cc8-e9f5-055e-319b-00ee06f998a0
Version            1.15.4
Build Date         2023-12-04T17:45:28Z
Storage Type       file
HA Enabled         false
Key                Value
---                -----
Seal Type          shamir
Initialized        true
Sealed             true
Total Shares       5
Threshold          3
Unseal Progress    2/3
Unseal Nonce       4c248cc8-e9f5-055e-319b-00ee06f998a0
Version            1.15.4
Build Date         2023-12-04T17:45:28Z
Storage Type       file
HA Enabled         false
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    5
Threshold       3
Unseal Progress 3/3
Unseal Nonce    4c248cc8-e9f5-055e-319b-00ee06f998a0
Version         1.15.4
Build Date      2023-12-04T17:45:28Z
Storage Type    file
HA Enabled      false
```

### 4. vault_set_pki.sh

The `vault_set_pki.sh` script is responsible for generating the root certificate, intermediate certificate, and HTTPS server certificate. All generated certificates, keys, and CSR files are stored in the `docker/addons/vault/scripts/data` directory.

The script pulls necessary parameters for certificate generation from environment variables, which are, by default, loaded from `docker/.env`.

- Environment variables prefixed with `MG_VAULT_PKI` in the `docker/.env` file are used for generating the root CA.
- Environment variables prefixed with `MG_VAULT_PKI_INT` are used for generating the intermediate CA.

To skip generating the server certificate and key, you can pass the `--skip-server-cert` option to the script:

```sh
./vault_set_pki.sh --skip-server-cert
```

#### Troubleshooting:

If you encounter the following error:

```sh
jq command could not be found, please install it and try again.
```

Install `jq` using:

```sh
sudo apt-get update && sudo apt-get install -y jq
```

After installing `jq`, rerun the script.

### 5. `vault_create_approle.sh`

This script enables AppRole authorization in Vault. The certs service uses these AppRole credentials to issue and revoke certificates from the Vault intermediate CA.

Example output:

```sh
Success! You are now authenticated. The token information displayed below
is already stored in the token helper. You do NOT need to run "vault login"
again. Future Vault requests will automatically use this token.

Key                  Value
---                  -----
token                <token_value>
token_accessor       i6YVeKh4wQ4e0Aj0ONiyGw1Z
token_duration       ∞
token_renewable      false
token_policies       ["root"]
identity_policies    []
policies             ["root"]
Creating new policy for AppRole
Successfully copied 2.56kB to magistrala-vault:/vault/magistrala_clients_certs_issue.hcl
Success! Uploaded policy: magistrala_clients_certs_issue
Enabling AppRole
Success! Enabled approle auth method at: approle/
Deleting old AppRole
Success! Data deleted (if it existed) at: auth/approle/role/magistrala_clients_certs_issuer
Creating new AppRole
Success! Data written to: auth/approle/role/magistrala_clients_certs_issuer
Writing custom role ID
Key        Value
---        -----
role_id    f23942b3-62b9-7456-784f-220ca3f703b9
Success! Data written to: auth/approle/role/magistrala_clients_certs_issuer/role-id
Writing custom secret
Key                   Value
---                   -----
secret_id             61d5a30f-634c-6027-f5b6-4934e6fc49b2
secret_id_accessor    1d744f6e-e0c2-5431-a87a-2b23fde584a7
secret_id_num_uses    0
secret_id_ttl         0s
Testing custom role ID and secret by logging in
Key                     Value
---                     -----
token                   <token_value>
token_accessor          9cuwS4mrLHKhJQMv0pl9Bbg9
token_duration          1h
token_renewable         true
token_policies          ["default" "magistrala_clients_certs_issue"]
identity_policies       []
policies                ["default" "magistrala_clients_certs_issue"]
token_meta_role_name    magistrala_clients_certs_issuer
```

By default, the `vault_create_approle.sh` script tries to enable the AppRole authentication method. Certs service uses the approle credentials to issue and revoke clients certificate from vault intermedate CA. If AppRole is already enabled, you can skip this step by passing the `--skip-enable-approle` argument:

```sh
./vault_create_approle.sh --skip-enable-approle
```

### 6. `vault_copy_certs.sh`

This script copies the required certificates and keys from `docker/addons/vault/scripts/data` to the `docker/ssl/certs` folder.

Example output:

```bash
Copying certificate files
'data/localhost.crt' -> '~/Documents/magistrala/docker/ssl/certs/magistrala-server.crt'
'data/localhost.key' -> '~/Documents/magistrala/docker/ssl/certs/magistrala-server.key'
'data/mg_int.key' -> '~/Documents/magistrala/docker/ssl/certs/ca.key'
'data/mg_int_bundle.crt' -> '~/Documents/magistrala/docker/ssl/certs/ca.crt'
```

## Custom `.env` Path Support

Vault scripts support specifying a custom `.env` file path using the `--env-file` argument. If this argument is not provided, the scripts will use the default `.env` file located at `docker/.env`.

To use a different `.env` file, include the `--env-file` argument followed by the path to your `.env` file when running the Vault scripts. Below are examples of how to execute each script with a custom `.env` file path:

```bash
./vault_init.sh --env-file /custom/path/.env
./vault_copy_env.sh --env-file /custom/path/.env
./vault_unseal.sh --env-file /custom/path/.env
./vault_set_pki.sh --env-file /custom/path/.env
./vault_create_approle.sh --env-file /custom/path/.env
./vault_copy_certs.sh --env-file /custom/path/.env
```

## Hashicorp Cloud Platform (HCP) Vault

To have the same PKI setup can done in Hashicorp Cloud Platform (HCP) Vault follow the below steps:
Requirement: [VAULT CLI](https://developer.hashicorp.com/vault/tutorials/getting-started/getting-started-install)

- Replace the environmental variable `MG_VAULT_ADDR` in `docker/.env` with HCP Vault address.
- Replace the environmental variable `MG_VAULT_TOKEN` in `docker/.env` with HCP Vault Admin token.
- Run script `vault_set_pki.sh` and `vault_create_approle.sh`.
- Optional step, run script `vault_copy_certs.sh` to copy certificates to magistrala default path.

## Vault CLI

It can also be useful to run the Vault CLI for inspection and administration work.

```bash
Usage: vault <command> [args]

Common commands:
    read        Read data and retrieves secrets
    write       Write data, configuration, and secrets
    delete      Delete secrets and configuration
    list        List data or secrets
    login       Authenticate locally
    agent       Start a Vault agent
    server      Start a Vault server
    status      Print seal and HA status
    unwrap      Unwrap a wrapped secret

Other commands:
    audit          Interact with audit devices
    auth           Interact with auth methods
    debug          Runs the debug command
    kv             Interact with Vault's Key-Value storage
    lease          Interact with leases
    monitor        Stream log messages from a Vault server
    namespace      Interact with namespaces
    operator       Perform operator-specific tasks
    path-help      Retrieve API help for paths
    plugin         Interact with Vault plugins and catalog
    policy         Interact with policies
    print          Prints runtime configurations
    secrets        Interact with secrets engines
    ssh            Initiate an SSH session
    token          Interact with tokens
```

If the Vault is setup through `docker/addons/vault`, then Vault CLI can be run directly using the Vault image in Docker: `docker run -it magistrala/vault:latest vault`

## Vault Web UI

If the Vault is setup through `docker/addons/vault`, Then Vault Web UI is accessible by default on `http://localhost:8200/ui`.
