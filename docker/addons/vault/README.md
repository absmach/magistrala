# Vault

This is Vault service deployment to be used with Magistrala.

When the Vault service is started, some initialization steps need to be done to set things up.

## Configuration


| Variable                                | Description                                                                   | Default                               |
| :---------------------------------------- | ------------------------------------------------------------------------------- | --------------------------------------- |
| MG_VAULT_HOST                           | Vault service address                                                         | vault                                 |
| MG_VAULT_PORT                           | Vault service port                                                            | 8200                                  |
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
| MG_VAULT_PKI_INT_THINGS_CERTS_ROLE_NAME | Vault Intermediate CA role name to issue Things certificates                  | magistrala_things_certs               |
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
| MG_VAULT_THINGS_CERTS_ISSUER_ROLEID     | Vault Intermediate CA Things Certificate issuer AppRole authentication RoleID | magistrala                            |
| MG_VAULT_THINGS_CERTS_ISSUER_SECRET     | Vault Intermediate CA Things Certificate issuer AppRole authentication Secret | magistrala                            |

## Setup

The following scripts are provided, which work on the running Vault service in Docker.

### 1. `vault_init.sh`

Calls `vault operator init` to perform the initial vault initialization and generates a `docker/addons/vault/data/secrets` file which contains the Vault unseal keys and root tokens.

Example contents for `data/secrets`:

```bash
Unseal Key 1: Ay0YZecYJ2HVtNtXfPootXK5LtF+JZoDmBb7IbbYdLBI
Unseal Key 2: P6hb7x2cglv0p61jdLyNE3+d44cJUOFaDt9jHFDfr8Df
Unseal Key 3: zSBfDHzUiWoOzXKY1pnnBqKO8UD2MDLuy8DNTxNtEBFy
Unseal Key 4: 5oJuDDuMI0I8snaw/n4VLNpvndvvKi6JlkgOxuWXqMSz
Unseal Key 5: ZhsUkk2tXBYEcWgz4WUCHH9rocoW6qZoiARWlkE5Epi5

Initial Root Token: s.V2hdd00P4bHtUQnoWZK2hSaS

Vault initialized with 5 key shares and a key threshold of 3. Please securely
distribute the key shares printed above. When the Vault is re-sealed,
restarted, or stopped, you must supply at least 3 of these keys to unseal it
before it can start servicing requests.

Vault does not store the generated master key. Without at least 3 key to
reconstruct the master key, Vault will remain permanently sealed!

It is possible to generate new unseal keys, provided you have a quorum of
existing unseal keys shares. See "vault operator rekey" for more information.
bash-4.4

Use 3 out of five keys presented and put it into .env file and than start the composition again Vault should be in unsealed state ( take a note that this is not recommended in terms of security, this is deployment for development) A real production deployment can use Vault auto unseal mode where vault gets unseal keys from some 3rd party KMS ( on AWS for example)
```

### 2. `vault_copy_env.sh`

After first step, the corresponding Vault environment variables (`MG_VAULT_TOKEN`, `MG_VAULT_UNSEAL_KEY_1`, `MG_VAULT_UNSEAL_KEY_2`, `MG_VAULT_UNSEAL_KEY_3`) should be updated in `.env` file.

`vault_copy_env.sh` scripts copies values from `docker/addons/vault/data/secrets` file and update environmental variables `MG_VAULT_TOKEN`, `MG_VAULT_UNSEAL_KEY_1`, `MG_VAULT_UNSEAL_KEY_2`, `MG_VAULT_UNSEAL_KEY_3` present in `.env` file.

### 3. `vault_unseal.sh`

This can be run after the initialization to unseal Vault, which is necessary for it to be used to store and/or get secrets.

This can be used if you don't want to restart the service.

The unseal environment variables need to be set in `.env` for the script to work (`MG_VAULT_TOKEN`,`MG_VAULT_UNSEAL_KEY_1`, `MG_VAULT_UNSEAL_KEY_2`, `MG_VAULT_UNSEAL_KEY_3`).

This script should not be necessary to run after the initial setup, since the Vault service unseals itself when starting the container.

### 4. `vault_set_pki.sh`

This script is used to generate the root certificate, intermediate certificate and HTTPS server certificate.  
All generate certificates, keys and CSR by `vault_set_pki.sh` will be present at `docker/addons/vault/data`.  

The parameters required for generating certificate are obtained from the environment variables which are loaded from `docker/.env`.  

Environmental variables starting with `MG_VAULT_PKI` in `docker/.env` file are used by `vault_set_pki.sh` to generate root CA.  
Environmental variables starting with`MG_VAULT_PKI_INT` in `docker/.env` file are used by `vault_set_pki.sh` to generate intermediate CA.  

### 5. `vault_create_approle.sh`  

This script is used to enable app role authorization in Vault. Certs service used the approle credentials to issue, revoke things certificate from vault intermedate CA.  

`vault_create_approle.sh` script by default tries to enable auth approle.  
If approle is already enabled in vault, then use args `skip_enable_app_role` to skip enable auth approle step.  
To skip enable auth approle step use the following  `vault_create_approle.sh   skip_enable_app_role`

### 6. `vault_copy_certs.sh`

This scripts copies the necessary certificates and keys from `docker/addons/vault/data` to the `docker/ssl/certs` folder.

## Vault CLI

It can also be useful to run the Vault CLI for inspection and administration work.

This can be done directly using the Vault image in Docker: `docker run -it magistrala/vault:latest vault`

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

### Vault Web UI

The Vault Web UI is accessible by default on `http://localhost:8200/ui`.
