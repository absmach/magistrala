# Users

Users service provides an HTTP API for managing users. Through this API clients are able to do the following actions:

- register new accounts
- login
- manage account(s) (list, update, delete)

For in-depth explanation of the aforementioned scenarios, as well as thorough understanding of SuperMQ, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                       | Description                                                             | Default                           |
| ------------------------------ | ----------------------------------------------------------------------- | --------------------------------- |
| SMQ_USERS_LOG_LEVEL            | Log level for users service (debug, info, warn, error)                  | info                              |
| SMQ_USERS_ADMIN_EMAIL          | Default user, created on startup                                        | <admin@example.com>               |
| SMQ_USERS_ADMIN_PASSWORD       | Default user password, created on startup                               | 12345678                          |
| SMQ_USERS_PASS_REGEX           | Password regex                                                          | ^.{8,}$                           |
| SMQ_TOKEN_RESET_ENDPOINT       | Password request reset endpoint, for constructing link                  | /reset-request                    |
| SMQ_USERS_HTTP_HOST            | Users service HTTP host                                                 | localhost                         |
| SMQ_USERS_HTTP_PORT            | Users service HTTP port                                                 | 9002                              |
| SMQ_USERS_HTTP_SERVER_CERT     | Path to the PEM encoded server certificate file                         | ""                                |
| SMQ_USERS_HTTP_SERVER_KEY      | Path to the PEM encoded server key file                                 | ""                                |
| SMQ_USERS_HTTP_SERVER_CA_CERTS | Path to the PEM encoded server CA certificate file                      | ""                                |
| SMQ_USERS_HTTP_CLIENT_CA_CERTS | Path to the PEM encoded client CA certificate file                      | ""                                |
| SMQ_AUTH_GRPC_URL              | Auth service GRPC URL                                                   | localhost:8181                    |
| SMQ_AUTH_GRPC_TIMEOUT          | Auth service GRPC timeout                                               | 1s                                |
| SMQ_AUTH_GRPC_CLIENT_CERT      | Path to the PEM encoded client certificate file                         | ""                                |
| SMQ_AUTH_GRPC_CLIENT_KEY       | Path to the PEM encoded client key file                                 | ""                                |
| SMQ_AUTH_GRPC_SERVER_CA_CERTS  | Path to the PEM encoded server CA certificate file                      | ""                                |
| SMQ_USERS_DB_HOST              | Database host address                                                   | localhost                         |
| SMQ_USERS_DB_PORT              | Database host port                                                      | 5432                              |
| SMQ_USERS_DB_USER              | Database user                                                           | supermq                           |
| SMQ_USERS_DB_PASS              | Database password                                                       | supermq                           |
| SMQ_USERS_DB_NAME              | Name of the database used by the service                                | users                             |
| SMQ_USERS_DB_SSL_MODE          | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                           |
| SMQ_USERS_DB_SSL_CERT          | Path to the PEM encoded certificate file                                | ""                                |
| SMQ_USERS_DB_SSL_KEY           | Path to the PEM encoded key file                                        | ""                                |
| SMQ_USERS_DB_SSL_ROOT_CERT     | Path to the PEM encoded root certificate file                           | ""                                |
| SMQ_EMAIL_HOST                 | Mail server host                                                        | localhost                         |
| SMQ_EMAIL_PORT                 | Mail server port                                                        | 25                                |
| SMQ_EMAIL_USERNAME             | Mail server username                                                    | ""                                |
| SMQ_EMAIL_PASSWORD             | Mail server password                                                    | ""                                |
| SMQ_EMAIL_FROM_ADDRESS         | Email "from" address                                                    | ""                                |
| SMQ_EMAIL_FROM_NAME            | Email "from" name                                                       | ""                                |
| SMQ_EMAIL_TEMPLATE             | Email template for sending emails with password reset link              | email.tmpl                        |
| SMQ_USERS_ES_URL               | Event store URL                                                         | <nats://localhost:4222>           |
| SMQ_JAEGER_URL                 | Jaeger server URL                                                       | <http://localhost:4318/v1/traces> |
| SMQ_OAUTH_UI_REDIRECT_URL      | OAuth UI redirect URL                                                   | <http://localhost:9095/domains>   |
| SMQ_OAUTH_UI_ERROR_URL         | OAuth UI error URL                                                      | <http://localhost:9095/error>     |
| SMQ_USERS_DELETE_INTERVAL      | Interval for deleting users                                             | 24h                               |
| SMQ_USERS_DELETE_AFTER         | Time after which users are deleted                                      | 720h                              |
| SMQ_JAEGER_TRACE_RATIO         | Jaeger sampling ratio                                                   | 1.0                               |
| SMQ_SEND_TELEMETRY             | Send telemetry to supermq call home server.                             | true                              |
| SMQ_USERS_INSTANCE_ID          | SuperMQ instance ID                                                     | ""                                |

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yml) service section in docker-compose file to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the service
make users

# copy binary to bin
make install

# set the environment variables and run the service
SMQ_USERS_LOG_LEVEL=info \
SMQ_USERS_ADMIN_EMAIL=admin@example.com \
SMQ_USERS_ADMIN_PASSWORD=12345678 \
SMQ_USERS_PASS_REGEX="^.{8,}$" \
SMQ_TOKEN_RESET_ENDPOINT="/reset-request" \
SMQ_USERS_HTTP_HOST=localhost \
SMQ_USERS_HTTP_PORT=9002 \
SMQ_USERS_HTTP_SERVER_CERT="" \
SMQ_USERS_HTTP_SERVER_KEY="" \
SMQ_USERS_HTTP_SERVER_CA_CERTS="" \
SMQ_USERS_HTTP_CLIENT_CA_CERTS="" \
SMQ_AUTH_GRPC_URL=localhost:8181 \
SMQ_AUTH_GRPC_TIMEOUT=1s \
SMQ_AUTH_GRPC_CLIENT_CERT="" \
SMQ_AUTH_GRPC_CLIENT_KEY="" \
SMQ_AUTH_GRPC_SERVER_CA_CERTS="" \
SMQ_USERS_DB_HOST=localhost \
SMQ_USERS_DB_PORT=5432 \
SMQ_USERS_DB_USER=supermq \
SMQ_USERS_DB_PASS=supermq \
SMQ_USERS_DB_NAME=users \
SMQ_USERS_DB_SSL_MODE=disable \
SMQ_USERS_DB_SSL_CERT="" \
SMQ_USERS_DB_SSL_KEY="" \
SMQ_USERS_DB_SSL_ROOT_CERT="" \
SMQ_EMAIL_HOST=smtp.mailtrap.io \
SMQ_EMAIL_PORT=2525 \
SMQ_EMAIL_USERNAME="18bf7f7070513" \
SMQ_EMAIL_PASSWORD="2b0d302e775b1e" \
SMQ_EMAIL_FROM_ADDRESS=from@example.com \
SMQ_EMAIL_FROM_NAME=Example \
SMQ_EMAIL_TEMPLATE="docker/templates/users.tmpl" \
SMQ_USERS_ES_URL=nats://localhost:4222 \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_OAUTH_UI_REDIRECT_URL=http://localhost:9095/domains \
SMQ_OAUTH_UI_ERROR_URL=http://localhost:9095/error \
SMQ_USERS_DELETE_INTERVAL=24h \
SMQ_USERS_DELETE_AFTER=720h \
SMQ_USERS_INSTANCE_ID="" \
$GOBIN/supermq-users
```

If `SMQ_EMAIL_TEMPLATE` doesn't point to any file service will function but password reset functionality will not work. The email environment variables are used to send emails with password reset link. The service expects a file in Go template format. The template should be something like [this](https://github.com/absmach/supermq/blob/main/docker/templates/users.tmpl).

Setting `SMQ_USERS_HTTP_SERVER_CERT` and `SMQ_USERS_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_USERS_HTTP_SERVER_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `SMQ_USERS_HTTP_CLIENT_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

Setting `SMQ_AUTH_GRPC_CLIENT_CERT` and `SMQ_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_AUTH_GRPC_SERVER_CA_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.supermq.abstractmachines.fr/?urls.primaryName=users-openapi.yml).

[doc]: https://docs.supermq.abstractmachines.fr
