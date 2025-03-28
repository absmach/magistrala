# BOOTSTRAP SERVICE

New devices need to be configured properly and connected to the Magistrala. Bootstrap service is used in order to accomplish that. This service provides the following features:

1. Creating new Magistrala Clients
2. Providing basic configuration for the newly created Clients
3. Enabling/disabling Clients

Pre-provisioning a new Client is as simple as sending Configuration data to the Bootstrap service. Once the Client is online, it sends a request for initial config to Bootstrap service. Bootstrap service provides an API for enabling and disabling Clients. Only enabled Clients can exchange messages over Magistrala. Bootstrapping does not implicitly enable Clients, it has to be done manually.

In order to bootstrap successfully, the Client needs to send bootstrapping request to the specific URL, as well as a secret key. This key and URL are pre-provisioned during the manufacturing process. If the Client is provisioned on the Bootstrap service side, the corresponding configuration will be sent as a response. Otherwise, the Client will be saved so that it can be provisioned later.

## Client Configuration Entity

Client Configuration consists of two logical parts: the custom configuration that can be interpreted by the Client itself and Magistrala-related configuration. Magistrala config contains:

1. corresponding Magistrala Client ID
2. corresponding Magistrala Client key
3. list of the Magistrala channels the Client is connected to

> Note: list of channels contains IDs of the Magistrala channels. These channels are _pre-provisioned_ on the Magistrala side and, unlike corresponding Magistrala Client, Bootstrap service is not able to create Magistrala Channels.

Enabling and disabling Client (adding Client to/from whitelist) is as simple as connecting corresponding Magistrala Client to the given list of Channels. Configuration keeps _state_ of the Client:

| State    | What it means                                  |
| -------- | ---------------------------------------------- |
| Inactive | Client is created, but isn't enabled           |
| Active   | Client is able to communicate using Magistrala |

Switching between states `Active` and `Inactive` enables and disables Client, respectively.

Client configuration also contains the so-called `external ID` and `external key`. An external ID is a unique identifier of corresponding Client. For example, a device MAC address is a good choice for external ID. External key is a secret key that is used for authentication during the bootstrapping procedure.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                       | Description                                                                      | Default                           |
| ------------------------------ | -------------------------------------------------------------------------------- | --------------------------------- |
| SMQ_BOOTSTRAP_LOG_LEVEL        | Log level for Bootstrap (debug, info, warn, error)                               | info                              |
| SMQ_BOOTSTRAP_DB_HOST          | Database host address                                                            | localhost                         |
| SMQ_BOOTSTRAP_DB_PORT          | Database host port                                                               | 5432                              |
| SMQ_BOOTSTRAP_DB_USER          | Database user                                                                    | magistrala                        |
| SMQ_BOOTSTRAP_DB_PASS          | Database password                                                                | magistrala                        |
| SMQ_BOOTSTRAP_DB_NAME          | Name of the database used by the service                                         | bootstrap                         |
| SMQ_BOOTSTRAP_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full)          | disable                           |
| SMQ_BOOTSTRAP_DB_SSL_CERT      | Path to the PEM encoded certificate file                                         | ""                                |
| SMQ_BOOTSTRAP_DB_SSL_KEY       | Path to the PEM encoded key file                                                 | ""                                |
| SMQ_BOOTSTRAP_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                                    | ""                                |
| SMQ_BOOTSTRAP_ENCRYPT_KEY      | Secret key for secure bootstrapping encryption                                   | 12345678910111213141516171819202  |
| SMQ_BOOTSTRAP_HTTP_HOST        | Bootstrap service HTTP host                                                      | ""                                |
| SMQ_BOOTSTRAP_HTTP_PORT        | Bootstrap service HTTP port                                                      | 9013                              |
| SMQ_BOOTSTRAP_HTTP_SERVER_CERT | Path to server certificate in pem format                                         | ""                                |
| SMQ_BOOTSTRAP_HTTP_SERVER_KEY  | Path to server key in pem format                                                 | ""                                |
| SMQ_BOOTSTRAP_EVENT_CONSUMER   | Bootstrap service event source consumer name                                     | bootstrap                         |
| SMQ_ES_URL                     | Event store URL                                                                  | <nats://localhost:4222>           |
| SMQ_AUTH_GRPC_URL              | Auth service Auth gRPC URL                                                       | <localhost:8181>                  |
| SMQ_AUTH_GRPC_TIMEOUT          | Auth service Auth gRPC request timeout in seconds                                | 1s                                |
| SMQ_AUTH_GRPC_CLIENT_CERT      | Path to the PEM encoded auth service Auth gRPC client certificate file           | ""                                |
| SMQ_AUTH_GRPC_CLIENT_KEY       | Path to the PEM encoded auth service Auth gRPC client key file                   | ""                                |
| SMQ_AUTH_GRPC_SERVER_CERTS     | Path to the PEM encoded auth server Auth gRPC server trusted CA certificate file | ""                                |
| SMQ_CLIENTS_URL                | Base URL for Magistrala Clients                                                  | <http://localhost:9000>           |
| SMQ_JAEGER_URL                 | Jaeger server URL                                                                | <http://localhost:4318/v1/traces> |
| SMQ_JAEGER_TRACE_RATIO         | Jaeger sampling ratio                                                            | 1.0                               |
| SMQ_SEND_TELEMETRY             | Send telemetry to magistrala call home server                                    | true                              |
| SMQ_BOOTSTRAP_INSTANCE_ID      | Bootstrap service instance ID                                                    | ""                                |

## Deployment

The service itself is distributed as Docker container. Check the [`bootstrap`](https://github.com/absmach/magistrala/blob/main/docker/addons/bootstrap/docker-compose.yaml) service section in docker-compose file to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the servic e
make bootstrap

# copy binary to bin
make install

# set the environment variables and run the service
SMQ_BOOTSTRAP_LOG_LEVEL=info \
SMQ_BOOTSTRAP_DB_HOST=localhost \
SMQ_BOOTSTRAP_DB_PORT=5432 \
SMQ_BOOTSTRAP_DB_USER=magistrala \
SMQ_BOOTSTRAP_DB_PASS=magistrala \
SMQ_BOOTSTRAP_DB_NAME=bootstrap \
SMQ_BOOTSTRAP_DB_SSL_MODE=disable \
SMQ_BOOTSTRAP_DB_SSL_CERT="" \
SMQ_BOOTSTRAP_DB_SSL_KEY="" \
SMQ_BOOTSTRAP_DB_SSL_ROOT_CERT="" \
SMQ_BOOTSTRAP_HTTP_HOST=localhost \
SMQ_BOOTSTRAP_HTTP_PORT=9013 \
SMQ_BOOTSTRAP_HTTP_SERVER_CERT="" \
SMQ_BOOTSTRAP_HTTP_SERVER_KEY="" \
SMQ_BOOTSTRAP_EVENT_CONSUMER=bootstrap \
SMQ_ES_URL=nats://localhost:4222 \
SMQ_AUTH_GRPC_URL=localhost:8181 \
SMQ_AUTH_GRPC_TIMEOUT=1s \
SMQ_AUTH_GRPC_CLIENT_CERT="" \
SMQ_AUTH_GRPC_CLIENT_KEY="" \
SMQ_AUTH_GRPC_SERVER_CERTS="" \
SMQ_CLIENTS_URL=http://localhost:9000 \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_BOOTSTRAP_INSTANCE_ID="" \
$GOBIN/magistrala-bootstrap
```

Setting `SMQ_BOOTSTRAP_HTTP_SERVER_CERT` and `SMQ_BOOTSTRAP_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `SMQ_AUTH_GRPC_CLIENT_CERT` and `SMQ_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_AUTH_GRPC_SERVER_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.abstractmachines.fr/?urls.primaryName=bootstrap.yaml).
