# BOOTSTRAP SERVICE

New devices need to be configured properly and connected to the Magistrala. Bootstrap service is used in order to accomplish that. This service provides the following features:

1. Creating new Magistrala Things
2. Providing basic configuration for the newly created Things
3. Enabling/disabling Things

Pre-provisioning a new Thing is as simple as sending Configuration data to the Bootstrap service. Once the Thing is online, it sends a request for initial config to Bootstrap service. Bootstrap service provides an API for enabling and disabling Things. Only enabled Things can exchange messages over Magistrala. Bootstrapping does not implicitly enable Things, it has to be done manually.

In order to bootstrap successfully, the Thing needs to send bootstrapping request to the specific URL, as well as a secret key. This key and URL are pre-provisioned during the manufacturing process. If the Thing is provisioned on the Bootstrap service side, the corresponding configuration will be sent as a response. Otherwise, the Thing will be saved so that it can be provisioned later.

## Thing Configuration Entity

Thing Configuration consists of two logical parts: the custom configuration that can be interpreted by the Thing itself and Magistrala-related configuration. Magistrala config contains:

1. corresponding Magistrala Thing ID
2. corresponding Magistrala Thing key
3. list of the Magistrala channels the Thing is connected to

> Note: list of channels contains IDs of the Magistrala channels. These channels are _pre-provisioned_ on the Magistrala side and, unlike corresponding Magistrala Thing, Bootstrap service is not able to create Magistrala Channels.

Enabling and disabling Thing (adding Thing to/from whitelist) is as simple as connecting corresponding Magistrala Thing to the given list of Channels. Configuration keeps _state_ of the Thing:

| State    | What it means                                 |
| -------- | --------------------------------------------- |
| Inactive | Thing is created, but isn't enabled           |
| Active   | Thing is able to communicate using Magistrala |

Switching between states `Active` and `Inactive` enables and disables Thing, respectively.

Thing configuration also contains the so-called `external ID` and `external key`. An external ID is a unique identifier of corresponding Thing. For example, a device MAC address is a good choice for external ID. External key is a secret key that is used for authentication during the bootstrapping procedure.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                                      | Default                             |
| ----------------------------- | -------------------------------------------------------------------------------- | ----------------------------------- |
| MG_BOOTSTRAP_LOG_LEVEL        | Log level for Bootstrap (debug, info, warn, error)                               | info                                |
| MG_BOOTSTRAP_DB_HOST          | Database host address                                                            | localhost                           |
| MG_BOOTSTRAP_DB_PORT          | Database host port                                                               | 5432                                |
| MG_BOOTSTRAP_DB_USER          | Database user                                                                    | magistrala                          |
| MG_BOOTSTRAP_DB_PASS          | Database password                                                                | magistrala                          |
| MG_BOOTSTRAP_DB_NAME          | Name of the database used by the service                                         | bootstrap                           |
| MG_BOOTSTRAP_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full)          | disable                             |
| MG_BOOTSTRAP_DB_SSL_CERT      | Path to the PEM encoded certificate file                                         | ""                                  |
| MG_BOOTSTRAP_DB_SSL_KEY       | Path to the PEM encoded key file                                                 | ""                                  |
| MG_BOOTSTRAP_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                                    | ""                                  |
| MG_BOOTSTRAP_ENCRYPT_KEY      | Secret key for secure bootstrapping encryption                                   | 12345678910111213141516171819202    |
| MG_BOOTSTRAP_HTTP_HOST        | Bootstrap service HTTP host                                                      | ""                                  |
| MG_BOOTSTRAP_HTTP_PORT        | Bootstrap service HTTP port                                                      | 9013                                |
| MG_BOOTSTRAP_HTTP_SERVER_CERT | Path to server certificate in pem format                                         | ""                                  |
| MG_BOOTSTRAP_HTTP_SERVER_KEY  | Path to server key in pem format                                                 | ""                                  |
| MG_BOOTSTRAP_EVENT_CONSUMER   | Bootstrap service event source consumer name                                     | bootstrap                           |
| MG_ES_URL                     | Event store URL                                                                  | <nats://localhost:4222>             |
| MG_AUTH_GRPC_URL              | Auth service Auth gRPC URL                                                       | <localhost:8181>                    |
| MG_AUTH_GRPC_TIMEOUT          | Auth service Auth gRPC request timeout in seconds                                | 1s                                  |
| MG_AUTH_GRPC_CLIENT_CERT      | Path to the PEM encoded auth service Auth gRPC client certificate file           | ""                                  |
| MG_AUTH_GRPC_CLIENT_KEY       | Path to the PEM encoded auth service Auth gRPC client key file                   | ""                                  |
| MG_AUTH_GRPC_SERVER_CERTS     | Path to the PEM encoded auth server Auth gRPC server trusted CA certificate file | ""                                  |
| MG_THINGS_URL                 | Base url for Magistrala Things                                                   | <http://localhost:9000>             |
| MG_JAEGER_URL                 | Jaeger server URL                                                                | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO         | Jaeger sampling ratio                                                            | 1.0                                 |
| MG_SEND_TELEMETRY             | Send telemetry to magistrala call home server                                    | true                                |
| MG_BOOTSTRAP_INSTANCE_ID      | Bootstrap service instance ID                                                    | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`boostrap`](https://github.com/absmach/magistrala/blob/master/docker/addons/bootstrap/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the service
make bootstrap

# copy binary to bin
make install

# set the environment variables and run the service
MG_BOOTSTRAP_LOG_LEVEL=info \
MG_BOOTSTRAP_DB_HOST=localhost \
MG_BOOTSTRAP_DB_PORT=5432 \
MG_BOOTSTRAP_DB_USER=magistrala \
MG_BOOTSTRAP_DB_PASS=magistrala \
MG_BOOTSTRAP_DB_NAME=bootstrap \
MG_BOOTSTRAP_DB_SSL_MODE=disable \
MG_BOOTSTRAP_DB_SSL_CERT="" \
MG_BOOTSTRAP_DB_SSL_KEY="" \
MG_BOOTSTRAP_DB_SSL_ROOT_CERT="" \
MG_BOOTSTRAP_HTTP_HOST=localhost \
MG_BOOTSTRAP_HTTP_PORT=9013 \
MG_BOOTSTRAP_HTTP_SERVER_CERT="" \
MG_BOOTSTRAP_HTTP_SERVER_KEY="" \
MG_BOOTSTRAP_EVENT_CONSUMER=bootstrap \
MG_ES_URL=nats://localhost:4222 \
MG_AUTH_GRPC_URL=localhost:8181 \
MG_AUTH_GRPC_TIMEOUT=1s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_SERVER_CERTS="" \
MG_THINGS_URL=http://localhost:9000 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_BOOTSTRAP_INSTANCE_ID="" \
$GOBIN/magistrala-bootstrap
```

Setting `MG_BOOTSTRAP_HTTP_SERVER_CERT` and `MG_BOOTSTRAP_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Setting `MG_AUTH_GRPC_CLIENT_CERT` and `MG_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_AUTH_GRPC_SERVER_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://api.mainflux.io/?urls.primaryName=bootstrap.yml).
