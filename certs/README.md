# Certs Service

Issues certificates for clients. `Certs` service can create certificates to be used when `SuperMQ` is deployed to support mTLS.
Certificate service can create certificates using PKI mode - where certificates issued by PKI, when you deploy `Vault` as PKI certificate management `cert` service will proxy requests to `Vault` previously checking access rights and saving info on successfully created certificate.

## PKI mode

When `SMQ_CERTS_VAULT_HOST` is set it is presumed that `Vault` is installed and `certs` service will issue certificates using `Vault` API.
First you'll need to set up `Vault`.
To setup `Vault` follow steps in [Build Your Own Certificate Authority (CA)](https://learn.hashicorp.com/tutorials/vault/pki-engine).

For lab purposes you can use docker-compose and script for setting up PKI in [https://github.com/absmach/supermq/blob/main/docker/addons/vault/README.md](https://github.com/absmach/supermq/blob/main/docker/addons/vault/README.md)

```bash
SMQ_CERTS_VAULT_HOST=<https://vault-domain:8200>
SMQ_CERTS_VAULT_NAMESPACE=<vault_namespace>
SMQ_CERTS_VAULT_APPROLE_ROLEID=<vault_approle_roleid>
SMQ_CERTS_VAULT_APPROLE_SECRET=<vault_approle_sceret>
SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_PATH=<vault_clients_certs_pki_path>
SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_ROLE_NAME=<vault_clients_certs_issue_role_name>
```

The certificates can also be revoked using `certs` service. To revoke a certificate you need to provide `client_id` of the client for which the certificate was issued.

```bash
curl -s -S -X DELETE http://localhost:9019/certs/revoke -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json'   -d '{"client_id":"c30b8842-507c-4bcd-973c-74008cef3be5"}'
```

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                                    | Description                                                                 | Default                                                             |
| :------------------------------------------ | --------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| SMQ_CERTS_LOG_LEVEL                         | Log level for the Certs (debug, info, warn, error)                          | info                                                                |
| SMQ_CERTS_HTTP_HOST                         | Service Certs host                                                          | ""                                                                  |
| SMQ_CERTS_HTTP_PORT                         | Service Certs port                                                          | 9019                                                                |
| SMQ_CERTS_HTTP_SERVER_CERT                  | Path to the PEM encoded server certificate file                             | ""                                                                  |
| SMQ_CERTS_HTTP_SERVER_KEY                   | Path to the PEM encoded server key file                                     | ""                                                                  |
| SMQ_AUTH_GRPC_URL                           | Auth service gRPC URL                                                       | [localhost:8181](localhost:8181)                                    |
| SMQ_AUTH_GRPC_TIMEOUT                       | Auth service gRPC request timeout in seconds                                | 1s                                                                  |
| SMQ_AUTH_GRPC_CLIENT_CERT                   | Path to the PEM encoded auth service gRPC client certificate file           | ""                                                                  |
| SMQ_AUTH_GRPC_CLIENT_KEY                    | Path to the PEM encoded auth service gRPC client key file                   | ""                                                                  |
| SMQ_AUTH_GRPC_SERVER_CERTS                  | Path to the PEM encoded auth server gRPC server trusted CA certificate file | ""                                                                  |
| SMQ_CERTS_SIGN_CA_PATH                      | Path to the PEM encoded CA certificate file                                 | ca.crt                                                              |
| SMQ_CERTS_SIGN_CA_KEY_PATH                  | Path to the PEM encoded CA key file                                         | ca.key                                                              |
| SMQ_CERTS_VAULT_HOST                        | Vault host                                                                  | http://vault:8200                                                   |
| SMQ_CERTS_VAULT_NAMESPACE                   | Vault namespace in which pki is present                                     | supermq                                                             |
| SMQ_CERTS_VAULT_APPROLE_ROLEID              | Vault AppRole auth RoleID                                                   | supermq                                                             |
| SMQ_CERTS_VAULT_APPROLE_SECRET              | Vault AppRole auth Secret                                                   | supermq                                                             |
| SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_PATH      | Vault PKI path for issuing Clients Certificates                             | pki_int                                                             |
| SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_ROLE_NAME | Vault PKI Role Name for issuing Clients Certificates                        | supermq_clients_certs                                               |
| SMQ_CERTS_DB_HOST                           | Database host                                                               | localhost                                                           |
| SMQ_CERTS_DB_PORT                           | Database port                                                               | 5432                                                                |
| SMQ_CERTS_DB_PASS                           | Database password                                                           | supermq                                                             |
| SMQ_CERTS_DB_USER                           | Database user                                                               | supermq                                                             |
| SMQ_CERTS_DB_NAME                           | Database name                                                               | certs                                                               |
| SMQ_CERTS_DB_SSL_MODE                       | Database SSL mode                                                           | disable                                                             |
| SMQ_CERTS_DB_SSL_CERT                       | Database SSL certificate                                                    | ""                                                                  |
| SMQ_CERTS_DB_SSL_KEY                        | Database SSL key                                                            | ""                                                                  |
| SMQ_CERTS_DB_SSL_ROOT_CERT                  | Database SSL root certificate                                               | ""                                                                  |
| SMQ_CLIENTS_URL                             | Clients service URL                                                         | [localhost:9000](localhost:9000)                                    |
| SMQ_JAEGER_URL                              | Jaeger server URL                                                           | [http://localhost:4318/v1/traces](http://localhost:4318//v1/traces) |
| SMQ_JAEGER_TRACE_RATIO                      | Jaeger sampling ratio                                                       | 1.0                                                                 |
| SMQ_SEND_TELEMETRY                          | Send telemetry to supermq call home server                                  | true                                                                |
| SMQ_CERTS_INSTANCE_ID                       | Service instance ID                                                         | ""                                                                  |

## Deployment

The service is distributed as Docker container. Check the [`certs`](https://github.com/absmach/supermq/blob/main/docker/addons/certs/docker-compose.yml) service section in docker-compose file to see how the service is deployed.

Running this service outside of container requires working instance of the auth service, clients service, postgres database, vault and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the certs
make certs

# copy binary to bin
make install

# set the environment variables and run the service
SMQ_CERTS_LOG_LEVEL=info \
SMQ_CERTS_HTTP_HOST=localhost \
SMQ_CERTS_HTTP_PORT=9019 \
SMQ_CERTS_HTTP_SERVER_CERT="" \
SMQ_CERTS_HTTP_SERVER_KEY="" \
SMQ_AUTH_GRPC_URL=localhost:8181 \
SMQ_AUTH_GRPC_TIMEOUT=1s \
SMQ_AUTH_GRPC_CLIENT_CERT="" \
SMQ_AUTH_GRPC_CLIENT_KEY="" \
SMQ_AUTH_GRPC_SERVER_CERTS="" \
SMQ_CERTS_SIGN_CA_PATH=ca.crt \
SMQ_CERTS_SIGN_CA_KEY_PATH=ca.key \
SMQ_CERTS_VAULT_HOST=http://vault:8200 \
SMQ_CERTS_VAULT_NAMESPACE=supermq \
SMQ_CERTS_VAULT_APPROLE_ROLEID=supermq \
SMQ_CERTS_VAULT_APPROLE_SECRET=supermq \
SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_PATH=pki_int \
SMQ_CERTS_VAULT_CLIENTS_CERTS_PKI_ROLE_NAME=supermq_clients_certs \
SMQ_CERTS_DB_HOST=localhost \
SMQ_CERTS_DB_PORT=5432 \
SMQ_CERTS_DB_PASS=supermq \
SMQ_CERTS_DB_USER=supermq \
SMQ_CERTS_DB_NAME=certs \
SMQ_CERTS_DB_SSL_MODE=disable \
SMQ_CERTS_DB_SSL_CERT="" \
SMQ_CERTS_DB_SSL_KEY="" \
SMQ_CERTS_DB_SSL_ROOT_CERT="" \
SMQ_CLIENTS_URL=localhost:9000 \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_CERTS_INSTANCE_ID="" \
$GOBIN/supermq-certs
```

Setting `SMQ_CERTS_HTTP_SERVER_CERT` and `SMQ_CERTS_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `SMQ_AUTH_GRPC_CLIENT_CERT` and `SMQ_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_AUTH_GRPC_SERVER_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [Certs section](https://docs.supermq.abstractmachines.fr/certs/).
