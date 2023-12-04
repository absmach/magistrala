# Certs Service

Issues certificates for things. `Certs` service can create certificates to be used when `Magistrala` is deployed to support mTLS.
Certificate service can create certificates using PKI mode - where certificates issued by PKI, when you deploy `Vault` as PKI certificate management `cert` service will proxy requests to `Vault` previously checking access rights and saving info on successfully created certificate.

## PKI mode

When `MG_CERTS_VAULT_HOST` is set it is presumed that `Vault` is installed and `certs` service will issue certificates using `Vault` API.
First you'll need to set up `Vault`.
To setup `Vault` follow steps in [Build Your Own Certificate Authority (CA)](https://learn.hashicorp.com/tutorials/vault/pki-engine).

To setup certs service with `Vault` following environment variables must be set:

```bash
MG_CERTS_VAULT_HOST=vault-domain.com
MG_CERTS_VAULT_PKI_PATH=<vault_pki_path>
MG_CERTS_VAULT_ROLE=<vault_role>
MG_CERTS_VAULT_TOKEN=<vault_acces_token>
```

For lab purposes you can use docker-compose and script for setting up PKI in [https://github.com/mteodor/vault](https://github.com/mteodor/vault)

The certificates can also be revoked using `certs` service. To revoke a certificate you need to provide `thing_id` of the thing for which the certificate was issued.

```bash
curl -s -S -X DELETE http://localhost:9019/certs/revoke -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json'   -d '{"thing_id":"c30b8842-507c-4bcd-973c-74008cef3be5"}'
```

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                  | Description                                                                 | Default                             |
| ------------------------- | --------------------------------------------------------------------------- | ----------------------------------- |
| MG_CERTS_LOG_LEVEL        | Log level for the Certs (debug, info, warn, error)                          | info                                |
| MG_CERTS_HTTP_HOST        | Service Certs host                                                          | ""                                  |
| MG_CERTS_HTTP_PORT        | Service Certs port                                                          | 9019                                |
| MG_CERTS_HTTP_SERVER_CERT | Path to the PEM encoded server certificate file                             | ""                                  |
| MG_CERTS_HTTP_SERVER_KEY  | Path to the PEM encoded server key file                                     | ""                                  |
| MG_AUTH_GRPC_URL          | Auth service gRPC URL                                                       | <localhost:8181>                    |
| MG_AUTH_GRPC_TIMEOUT      | Auth service gRPC request timeout in seconds                                | 1s                                  |
| MG_AUTH_GRPC_CLIENT_CERT  | Path to the PEM encoded auth service gRPC client certificate file           | ""                                  |
| MG_AUTH_GRPC_CLIENT_KEY   | Path to the PEM encoded auth service gRPC client key file                   | ""                                  |
| MG_AUTH_GRPC_SERVER_CERTS | Path to the PEM encoded auth server gRPC server trusted CA certificate file | ""                                  |
| MG_CERTS_SIGN_CA_PATH     | Path to the PEM encoded CA certificate file                                 | ca.crt                              |
| MG_CERTS_SIGN_CA_KEY_PATH | Path to the PEM encoded CA key file                                         | ca.key                              |
| MG_CERTS_VAULT_HOST       | Vault host                                                                  | ""                                  |
| MG_VAULT_PKI_INT_PATH     | Vault PKI intermediate path                                                 | pki_int                             |
| MG_VAULT_CA_ROLE_NAME     | Vault PKI role name                                                         | magistrala                          |
| MG_VAULT_TOKEN            | Vault token                                                                 | ""                                  |
| MG_CERTS_DB_HOST          | Database host                                                               | localhost                           |
| MG_CERTS_DB_PORT          | Database port                                                               | 5432                                |
| MG_CERTS_DB_PASS          | Database password                                                           | magistrala                          |
| MG_CERTS_DB_USER          | Database user                                                               | magistrala                          |
| MG_CERTS_DB_NAME          | Database name                                                               | certs                               |
| MG_CERTS_DB_SSL_MODE      | Database SSL mode                                                           | disable                             |
| MG_CERTS_DB_SSL_CERT      | Database SSL certificate                                                    | ""                                  |
| MG_CERTS_DB_SSL_KEY       | Database SSL key                                                            | ""                                  |
| MG_CERTS_DB_SSL_ROOT_CERT | Database SSL root certificate                                               | ""                                  |
| MG_THINGS_URL             | Things service URL                                                          | <localhost:9000>                    |
| MG_JAEGER_URL             | Jaeger server URL                                                           | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO     | Jaeger sampling ratio                                                       | 1.0                                 |
| MG_SEND_TELEMETRY         | Send telemetry to magistrala call home server                               | true                                |
| MG_CERTS_INSTANCE_ID      | Service instance ID                                                         | ""                                  |

## Deployment

The service is distributed as Docker container. Check the [`certs`](https://github.com/absmach/magistrala/blob/main/docker/addons/bootstrap/docker-compose.yml) service section in docker-compose to see how the service is deployed.

Running this service outside of container requires working instance of the auth service, things service, postgres database, vault and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the certs
make certs

# copy binary to bin
make install

# set the environment variables and run the service
MG_CERTS_LOG_LEVEL=info \
MG_CERTS_HTTP_HOST=localhost \
MG_CERTS_HTTP_PORT=9019 \
MG_CERTS_HTTP_SERVER_CERT="" \
MG_CERTS_HTTP_SERVER_KEY="" \
MG_AUTH_GRPC_URL=localhost:8181 \
MG_AUTH_GRPC_TIMEOUT=1s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_SERVER_CERTS="" \
MG_CERTS_SIGN_CA_PATH=ca.crt \
MG_CERTS_SIGN_CA_KEY_PATH=ca.key \
MG_CERTS_VAULT_HOST="" \
MG_VAULT_PKI_INT_PATH=pki_int \
MG_VAULT_CA_ROLE_NAME=magistrala \
MG_VAULT_TOKEN="" \
MG_CERTS_DB_HOST=localhost \
MG_CERTS_DB_PORT=5432 \
MG_CERTS_DB_PASS=magistrala \
MG_CERTS_DB_USER=magistrala \
MG_CERTS_DB_NAME=certs \
MG_CERTS_DB_SSL_MODE=disable \
MG_CERTS_DB_SSL_CERT="" \
MG_CERTS_DB_SSL_KEY="" \
MG_CERTS_DB_SSL_ROOT_CERT="" \
MG_THINGS_URL=localhost:9000 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_CERTS_INSTANCE_ID="" \
$GOBIN/magistrala-certs
```

Setting `MG_CERTS_HTTP_SERVER_CERT` and `MG_CERTS_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_AUTH_GRPC_CLIENT_CERT` and `MG_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_AUTH_GRPC_SERVER_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [Certs section](https://docs.mainflux.io/certs/).
