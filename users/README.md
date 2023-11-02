# Clients

Users service provides an HTTP API for managing users. Through this API clients
are able to do the following actions:

- register new accounts
- obtain access tokens
- verify access tokens

For in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Magistrala, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                             | Default                          |
| ------------------------------- | ----------------------------------------------------------------------- | -------------------------------- |
| MG_USERS_LOG_LEVEL              | Log level for Users (debug, info, warn, error)                          | info                             |
| MG_USERS_SECRET_KEY             | Default secret key used to generate tokens                              | magistrala                       |
| MG_USERS_ADMIN_EMAIL            | Default user, created on startup                                        | <admin@example.com>              |
| MG_USERS_ADMIN_PASSWORD         | Default user password, created on startup                               | 12345678                         |
| MG_USERS_PASS_REGEX             | Password regex                                                          | `^.{8,}$`                        |
| MG_USERS_ACCESS_TOKEN_DURATION  | Duration for an access token to be valid                                | 15m                              |
| MG_USERS_REFRESH_TOKEN_DURATION | Duration for a refresh token to be valid                                | 24h                              |
| MG_TOKEN_RESET_ENDPOINT         | Password request reset endpoint, for constructing link                  | /reset-request                   |
| MG_USERS_HTTP_HOST              | Users service HTTP host                                                 | localhost                        |
| MG_USERS_HTTP_PORT              | Users service HTTP port                                                 | 9002                             |
| MG_USERS_HTTP_SERVER_CERT       | Path to server certificate in pem format                                | ""                               |
| MG_USERS_HTTP_SERVER_KEY        | Path to server key in pem format                                        | ""                               |
| MG_USERS_GRPC_HOST              | Users service GRPC host                                                 | localhost                        |
| MG_USERS_GRPC_PORT              | Users service GRPC port                                                 | 7001                             |
| MG_USERS_GRPC_SERVER_CERT       | Path to server certificate in pem format                                | ""                               |
| MG_USERS_GRPC_SERVER_KEY        | Path to server key in pem format                                        | ""                               |
| MG_USERS_DB_HOST                | Database host address                                                   | localhost                        |
| MG_USERS_DB_PORT                | Database host port                                                      | 5432                             |
| MG_USERS_DB_USER                | Database user                                                           | magistrala                       |
| MG_USERS_DB_PASS                | Database password                                                       | magistrala                       |
| MG_USERS_DB_NAME                | Name of the database used by the service                                | users                            |
| MG_USERS_DB_SSL_MODE            | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                          |
| MG_USERS_DB_SSL_CERT            | Path to the PEM encoded certificate file                                | ""                               |
| MG_USERS_DB_SSL_KEY             | Path to the PEM encoded key file                                        | ""                               |
| MG_USERS_DB_SSL_ROOT_CERT       | Path to the PEM encoded root certificate file                           | ""                               |
| MG_EMAIL_HOST                   | Mail server host                                                        | localhost                        |
| MG_EMAIL_PORT                   | Mail server port                                                        | 25                               |
| MG_EMAIL_USERNAME               | Mail server username                                                    |                                  |
| MG_EMAIL_PASSWORD               | Mail server password                                                    |                                  |
| MG_EMAIL_FROM_ADDRESS           | Email "from" address                                                    |                                  |
| MG_EMAIL_FROM_NAME              | Email "from" name                                                       |                                  |
| MG_EMAIL_TEMPLATE               | Email template for sending emails with password reset link              | email.tmpl                       |
| MG_JAEGER_URL                   | Jaeger server URL                                                       | <http://jaeger:14268/api/traces> |
| MG_SEND_TELEMETRY               | Send telemetry to magistrala call home server.                          | true                             |
| MG_INSTANCE_ID                  | Magistrala instance ID                                                  | ""                               |

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/absmach/magistrala/blob/master/docker/docker-compose.yml#L109-L143) service section in docker-compose to see how service is deployed.

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
MG_USERS_LOG_LEVEL=[Users log level] \
MG_USERS_SECRET_KEY=[Secret key used to generate tokens] \
MG_USERS_ADMIN_EMAIL=[Default user, created on startup] \
MG_USERS_ADMIN_PASSWORD=[Default user password, created on startup] \
MG_USERS_PASS_REGEX=[Password regex] \
MG_USERS_ACCESS_TOKEN_DURATION=[Duration for an access token to be valid] \
MG_USERS_REFRESH_TOKEN_DURATION=[Duration for a refresh token to be valid] \
MG_TOKEN_RESET_ENDPOINT=[Password reset token endpoint] \
MG_USERS_HTTP_HOST=[Service HTTP host] \
MG_USERS_HTTP_PORT=[Service HTTP port] \
MG_USERS_HTTP_SERVER_CERT=[Path to server certificate] \
MG_USERS_HTTP_SERVER_KEY=[Path to server key] \
MG_USERS_GRPC_HOST=[Service GRPC host] \
MG_USERS_GRPC_PORT=[Service GRPC port] \
MG_USERS_GRPC_SERVER_CERT=[Path to server certificate] \
MG_USERS_GRPC_SERVER_KEY=[Path to server key] \
MG_USERS_DB_HOST=[Database host address] \
MG_USERS_DB_PORT=[Database host port] \
MG_USERS_DB_USER=[Database user] \
MG_USERS_DB_PASS=[Database password] \
MG_USERS_DB_NAME=[Name of the database used by the service] \
MG_USERS_DB_SSL_MODE=[SSL mode to connect to the database with] \
MG_USERS_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
MG_USERS_DB_SSL_KEY=[Path to the PEM encoded key file] \
MG_USERS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
MG_EMAIL_HOST=[Mail server host] \
MG_EMAIL_PORT=[Mail server port] \
MG_EMAIL_USERNAME=[Mail server username] \
MG_EMAIL_PASSWORD=[Mail server password] \
MG_EMAIL_FROM_ADDRESS=[Email from address] \
MG_EMAIL_FROM_NAME=[Email from name] \
MG_EMAIL_TEMPLATE=[Email template file] \
MG_JAEGER_URL=[Jaeger server URL] \
MG_SEND_TELEMETRY=[Send telemetry to Jaeger (true/false)] \
MG_USERS_INSTANCE_ID=[Instance ID] \
$GOBIN/magistrala-users
```

If `MG_EMAIL_TEMPLATE` doesn't point to any file service will function but password reset functionality will not work.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=users-openapi.yml).

[doc]: https://docs.mainflux.io
