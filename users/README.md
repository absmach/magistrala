# Clients

Users service provides an HTTP API for managing users. Through this API clients are able to do the following actions:

- register new accounts
- login
- manage account(s) (list, update, delete)

For in-depth explanation of the aforementioned scenarios, as well as thorough understanding of Magistrala, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                             | Default                             |
| ----------------------------- | ----------------------------------------------------------------------- | ----------------------------------- |
| MG_USERS_LOG_LEVEL            | Log level for users service (debug, info, warn, error)                  | info                                |
| MG_USERS_ADMIN_EMAIL          | Default user, created on startup                                        | <admin@example.com>                 |
| MG_USERS_ADMIN_PASSWORD       | Default user password, created on startup                               | 12345678                            |
| MG_USERS_PASS_REGEX           | Password regex                                                          | ^.{8,}$                             |
| MG_TOKEN_RESET_ENDPOINT       | Password request reset endpoint, for constructing link                  | /reset-request                      |
| MG_USERS_HTTP_HOST            | Users service HTTP host                                                 | localhost                           |
| MG_USERS_HTTP_PORT            | Users service HTTP port                                                 | 9002                                |
| MG_USERS_HTTP_SERVER_CERT     | Path to the PEM encoded server certificate file                         | ""                                  |
| MG_USERS_HTTP_SERVER_KEY      | Path to the PEM encoded server key file                                 | ""                                  |
| MG_USERS_HTTP_SERVER_CA_CERTS | Path to the PEM encoded server CA certificate file                      | ""                                  |
| MG_USERS_HTTP_CLIENT_CA_CERTS | Path to the PEM encoded client CA certificate file                      | ""                                  |
| MG_AUTH_GRPC_URL              | Auth service GRPC URL                                                   | localhost:8181                      |
| MG_AUTH_GRPC_TIMEOUT          | Auth service GRPC timeout                                               | 1s                                  |
| MG_AUTH_GRPC_CLIENT_CERT      | Path to the PEM encoded client certificate file                         | ""                                  |
| MG_AUTH_GRPC_CLIENT_KEY       | Path to the PEM encoded client key file                                 | ""                                  |
| MG_AUTH_GRPC_SERVER_CA_CERTS  | Path to the PEM encoded server CA certificate file                      | ""                                  |
| MG_USERS_DB_HOST              | Database host address                                                   | localhost                           |
| MG_USERS_DB_PORT              | Database host port                                                      | 5432                                |
| MG_USERS_DB_USER              | Database user                                                           | magistrala                          |
| MG_USERS_DB_PASS              | Database password                                                       | magistrala                          |
| MG_USERS_DB_NAME              | Name of the database used by the service                                | users                               |
| MG_USERS_DB_SSL_MODE          | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                             |
| MG_USERS_DB_SSL_CERT          | Path to the PEM encoded certificate file                                | ""                                  |
| MG_USERS_DB_SSL_KEY           | Path to the PEM encoded key file                                        | ""                                  |
| MG_USERS_DB_SSL_ROOT_CERT     | Path to the PEM encoded root certificate file                           | ""                                  |
| MG_EMAIL_HOST                 | Mail server host                                                        | localhost                           |
| MG_EMAIL_PORT                 | Mail server port                                                        | 25                                  |
| MG_EMAIL_USERNAME             | Mail server username                                                    | ""                                  |
| MG_EMAIL_PASSWORD             | Mail server password                                                    | ""                                  |
| MG_EMAIL_FROM_ADDRESS         | Email "from" address                                                    | ""                                  |
| MG_EMAIL_FROM_NAME            | Email "from" name                                                       | ""                                  |
| MG_EMAIL_TEMPLATE             | Email template for sending emails with password reset link              | email.tmpl                          |
| MG_USERS_ES_URL               | Event store URL                                                         | <nats://localhost:4222>             |
| MG_JAEGER_URL                 | Jaeger server URL                                                       | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO         | Jaeger sampling ratio                                                   | 1.0                                 |
| MG_SEND_TELEMETRY             | Send telemetry to magistrala call home server.                          | true                                |
| MG_USERS_INSTANCE_ID          | Magistrala instance ID                                                  | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/absmach/magistrala/blob/main/docker/docker-compose.yml) service section in docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the service
make users

# copy binary to bin
make install

# set the environment variables and run the service
MG_USERS_LOG_LEVEL=info \
MG_USERS_ADMIN_EMAIL=admin@example.com \
MG_USERS_ADMIN_PASSWORD=12345678 \
MG_USERS_PASS_REGEX="^.{8,}$" \
MG_TOKEN_RESET_ENDPOINT="/reset-request" \
MG_USERS_HTTP_HOST=localhost \
MG_USERS_HTTP_PORT=9002 \
MG_USERS_HTTP_SERVER_CERT="" \
MG_USERS_HTTP_SERVER_KEY="" \
MG_USERS_HTTP_SERVER_CA_CERTS="" \
MG_USERS_HTTP_CLIENT_CA_CERTS="" \
MG_AUTH_GRPC_URL=localhost:8181 \
MG_AUTH_GRPC_TIMEOUT=1s \
MG_AUTH_GRPC_CLIENT_CERT="" \
MG_AUTH_GRPC_CLIENT_KEY="" \
MG_AUTH_GRPC_SERVER_CA_CERTS="" \
MG_USERS_DB_HOST=localhost \
MG_USERS_DB_PORT=5432 \
MG_USERS_DB_USER=magistrala \
MG_USERS_DB_PASS=magistrala \
MG_USERS_DB_NAME=users \
MG_USERS_DB_SSL_MODE=disable \
MG_USERS_DB_SSL_CERT="" \
MG_USERS_DB_SSL_KEY="" \
MG_USERS_DB_SSL_ROOT_CERT="" \
MG_EMAIL_HOST=smtp.mailtrap.io \
MG_EMAIL_PORT=2525 \
MG_EMAIL_USERNAME="18bf7f7070513" \
MG_EMAIL_PASSWORD="2b0d302e775b1e" \
MG_EMAIL_FROM_ADDRESS=from@example.com \
MG_EMAIL_FROM_NAME=Example \
MG_EMAIL_TEMPLATE="docker/templates/users.tmpl" \
MG_USERS_ES_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_USERS_INSTANCE_ID="" \
$GOBIN/magistrala-users
```

If `MG_EMAIL_TEMPLATE` doesn't point to any file service will function but password reset functionality will not work. The email environment variables are used to send emails with password reset link. The service expects a file in Go template format. The template should be something like [this](https://github.com/absmach/magistrala/blob/main/docker/templates/users.tmpl).

Setting `MG_USERS_HTTP_SERVER_CERT` and `MG_USERS_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_USERS_HTTP_SERVER_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `MG_USERS_HTTP_CLIENT_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

Setting `MG_AUTH_GRPC_CLIENT_CERT` and `MG_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_AUTH_GRPC_SERVER_CA_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://api.mainflux.io/?urls.primaryName=users-openapi.yml).

[doc]: https://docs.mainflux.io
