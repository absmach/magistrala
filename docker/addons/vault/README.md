This is Vault service deployment to be used with Mainflux.

When the Vault service is started, some initialization steps need to be done to set things up.

## Configuration

| Variable                  | Description                                                             | Default        |
| ------------------------- | ----------------------------------------------------------------------- | -------------- |
| MF_VAULT_HOST             | Vault service address                                                   | vault          |
| MF_VAULT_PORT             | Vault service port                                                      | 8200           |
| MF_VAULT_UNSEAL_KEY_1     | Vault unseal key                                                        | ""             |
| MF_VAULT_UNSEAL_KEY_2     | Vault unseal key                                                        | ""             |
| MF_VAULT_UNSEAL_KEY_3     | Vault unseal key                                                        | ""             |
| MF_VAULT_TOKEN            | Vault cli access token                                                  | ""             |
| MF_VAULT_PKI_PATH         | Vault secrets engine path for CA                                        | pki            |
| MF_VAULT_PKI_INT_PATH     | Vault secrets engine path for intermediate CA                           | pki_int        |
| MF_VAULT_CA_ROLE_NAME     | Vault secrets engine role                                               | mainflux       |
| MF_VAULT_CA_NAME          | Certificates name used by `vault-set-pki.sh`                            | mainflux       |
| MF_VAULT_CA_CN            | Common name used for CA creation by `vault-set-pki.sh`                  | mainflux.com   |
| MF_VAULT_CA_OU            | Org unit used for CA creation by `vault-set-pki.sh`                     | Mainflux Cloud |
| MF_VAULT_CA_O             | Organization used for CA creation by `vault-set-pki.sh`                 | Mainflux Labs  |
| MF_VAULT_CA_C             | Country used for CA creation by `vault-set-pki.sh`                      | Serbia         |
| MF_VAULT_CA_L             | Location used for CA creation by `vault-set-pki.sh`                     | Belgrade       |


## Setup

The following scripts are provided, which work on the running Vault service in Docker.

1. `vault-init.sh`

Calls `vault operator init` to perform the initial vault initialization and generates
a `data/secrets` file which contains the Vault unseal keys and root tokens.

After this step, the corresponding Vault environment variables (`MF_VAULT_TOKEN`, `MF_VAULT_UNSEAL_KEY_1`,
`MF_VAULT_UNSEAL_KEY_2`, `MF_VAULT_UNSEAL_KEY_3`) should be updated in `.env` file.

Example contents for `data/secrets`:

```
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

2. `vault-unseal.sh`

This can be run after the initialization to unseal Vault, which is necessary for it to be used to store and/or get secrets.
This can be used if you don't want to restart the service.

The unseal environment variables need to be set in `.env` for the script to work (`MF_VAULT_TOKEN`, `MF_VAULT_UNSEAL_KEY_1`,
`MF_VAULT_UNSEAL_KEY_2`, `MF_VAULT_UNSEAL_KEY_3`).

This script should not be necessary to run after the initial setup, since the Vault service unseals itself when
starting the container.

3. `vault-set-pki.sh`

This script is used to generate the root certificate, intermediate certificate and HTTPS server certificate.
After it runs, it copies the necessary certificates and keys to the `docker/ssl/certs` folder.

The CA parameters are obtained from the environment variables starting with `MF_VAULT_CA` in `.env` file.

## Vault CLI 

It can also be useful to run the Vault CLI for inspection and administration work.

This can be done directly using the Vault image in Docker: `docker run -it mainflux/vault:latest vault`

```
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
