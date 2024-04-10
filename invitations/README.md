# Invitation Service

Invitation service is responsible for sending invitations to users to join a domain.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                        | Description                                      | Default                 |
| ------------------------------- | ------------------------------------------------ | ----------------------- |
| MG_INVITATION_LOG_LEVEL         | Log level for the Invitation service             | debug                   |
| MG_USERS_URL                    | Users service URL                                | <http://localhost:9002> |
| MG_DOMAINS_URL                  | Domains service URL                              | <http://localhost:8189> |
| MG_INVITATIONS_HTTP_HOST        | Invitation service HTTP listening host           | localhost               |
| MG_INVITATIONS_HTTP_PORT        | Invitation service HTTP listening port           | 9020                    |
| MG_INVITATIONS_HTTP_SERVER_CERT | Invitation service server certificate            | ""                      |
| MG_INVITATIONS_HTTP_SERVER_KEY  | Invitation service server key                    | ""                      |
| MG_AUTH_GRPC_URL                | Auth service gRPC URL                            | localhost:8181          |
| MG_AUTH_GRPC_TIMEOUT            | Auth service gRPC request timeout in seconds     | 1s                      |
| MG_AUTH_GRPC_CLIENT_CERT        | Path to client certificate in PEM format         | ""                      |
| MG_AUTH_GRPC_CLIENT_KEY         | Path to client key in PEM format                 | ""                      |
| MG_AUTH_GRPC_CLIENT_CA_CERTS    | Path to trusted CAs in PEM format                | ""                      |
| MG_INVITATIONS_DB_HOST          | Invitation service database host                 | localhost               |
| MG_INVITATIONS_DB_USER          | Invitation service database user                 | magistrala              |
| MG_INVITATIONS_DB_PASS          | Invitation service database password             | magistrala              |
| MG_INVITATIONS_DB_PORT          | Invitation service database port                 | 5432                    |
| MG_INVITATIONS_DB_NAME          | Invitation service database name                 | invitations             |
| MG_INVITATIONS_DB_SSL_MODE      | Invitation service database SSL mode             | disable                 |
| MG_INVITATIONS_DB_SSL_CERT      | Invitation service database SSL certificate      | ""                      |
| MG_INVITATIONS_DB_SSL_KEY       | Invitation service database SSL key              | ""                      |
| MG_INVITATIONS_DB_SSL_ROOT_CERT | Invitation service database SSL root certificate | ""                      |
| MG_INVITATIONS_INSTANCE_ID      | Invitation service instance ID                   |                         |

## Deployment

The service itself is distributed as Docker container. Check the [`invitation`](https://github.com/absmach/amdm/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the http
make invitation

# copy binary to bin
make install

# set the environment variables and run the service
MG_INVITATION_LOG_LEVEL=info \
MG_INVITATIONS_ENDPOINT=/invitations \
MG_USERS_URL="http://localhost:9002" \
MG_DOMAINS_URL="http://localhost:8189" \
MG_INVITATIONS_HTTP_HOST=localhost \
MG_INVITATIONS_HTTP_PORT=9020 \
MG_INVITATIONS_HTTP_SERVER_CERT="" \
MG_INVITATIONS_HTTP_SERVER_KEY="" \
MG_AUTH_GRPC_URL=localhost:8181 \
MG_AUTH_GRPC_TIMEOUT=1s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_CLIENT_CA_CERTS="" \
MG_INVITATIONS_DB_HOST=localhost \
MG_INVITATIONS_DB_USER=magistrala \
MG_INVITATIONS_DB_PASS=magistrala \
MG_INVITATIONS_DB_PORT=5432 \
MG_INVITATIONS_DB_NAME=invitations \
MG_INVITATIONS_DB_SSL_MODE=disable \
MG_INVITATIONS_DB_SSL_CERT="" \
MG_INVITATIONS_DB_SSL_KEY="" \
MG_INVITATIONS_DB_SSL_ROOT_CERT="" \
$GOBIN/magistrala-invitation
```

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.abstractmachines.fr/?urls.primaryName=invitations.yml).
