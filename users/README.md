# Clients

Users service provides an HTTP API for managing users. Through this API clients
are able to do the following actions:

- register new accounts
- obtain access tokens
- verify access tokens

For in-depth explanation of the aforementioned scenarios, as well as thorough
understanding of Mainflux, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                                             | Default                        |
| ------------------------------- | ----------------------------------------------------------------------- | ------------------------------ |
| MF_USERS_LOG_LEVEL              | Log level for Users (debug, info, warn, error)                          | info                           |
| MF_USERS_SECRET_KEY             | Default secret key used to generate tokens                              | mainflux                       |
| MF_USERS_ADMIN_EMAIL            | Default user, created on startup                                        | admin@example.com              |
| MF_USERS_ADMIN_PASSWORD         | Default user password, created on startup                               | 12345678                       |
| MF_USERS_PASS_REGEX             | Password regex                                                          | `^.{8,}$`                      |
| MF_USERS_ACCESS_TOKEN_DURATION  | Duration for an access token to be valid                                | 15m                            |
| MF_USERS_REFRESH_TOKEN_DURATION | Duration for a refresh token to be valid                                | 24h                            |
| MF_TOKEN_RESET_ENDPOINT         | Password request reset endpoint, for constructing link                  | /reset-request                 |
| MF_USERS_HTTP_HOST              | Users service HTTP host                                                 | localhost                      |
| MF_USERS_HTTP_PORT              | Users service HTTP port                                                 | 9002                           |
| MF_USERS_HTTP_SERVER_CERT       | Path to server certificate in pem format                                | ""                             |
| MF_USERS_HTTP_SERVER_KEY        | Path to server key in pem format                                        | ""                             |
| MF_USERS_GRPC_HOST              | Users service GRPC host                                                 | localhost                      |
| MF_USERS_GRPC_PORT              | Users service GRPC port                                                 | 7001                           |
| MF_USERS_GRPC_SERVER_CERT       | Path to server certificate in pem format                                | ""                             |
| MF_USERS_GRPC_SERVER_KEY        | Path to server key in pem format                                        | ""                             |
| MF_USERS_DB_HOST                | Database host address                                                   | localhost                      |
| MF_USERS_DB_PORT                | Database host port                                                      | 5432                           |
| MF_USERS_DB_USER                | Database user                                                           | mainflux                       |
| MF_USERS_DB_PASS                | Database password                                                       | mainflux                       |
| MF_USERS_DB_NAME                | Name of the database used by the service                                | users                          |
| MF_USERS_DB_SSL_MODE            | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                        |
| MF_USERS_DB_SSL_CERT            | Path to the PEM encoded certificate file                                | ""                             |
| MF_USERS_DB_SSL_KEY             | Path to the PEM encoded key file                                        | ""                             |
| MF_USERS_DB_SSL_ROOT_CERT       | Path to the PEM encoded root certificate file                           | ""                             |
| MF_EMAIL_HOST                   | Mail server host                                                        | localhost                      |
| MF_EMAIL_PORT                   | Mail server port                                                        | 25                             |
| MF_EMAIL_USERNAME               | Mail server username                                                    |                                |
| MF_EMAIL_PASSWORD               | Mail server password                                                    |                                |
| MF_EMAIL_FROM_ADDRESS           | Email "from" address                                                    |                                |
| MF_EMAIL_FROM_NAME              | Email "from" name                                                       |                                |
| MF_EMAIL_TEMPLATE               | Email template for sending emails with password reset link              | email.tmpl                     |
| MF_JAEGER_URL                   | Jaeger server URL                                                       | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY               | Send telemetry to mainflux call home server.                            | true                           |
| MF_INSTANCE_ID                  | Mainflux instance ID                                                    | ""                             |

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml#L109-L143) service section in docker-compose to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the service
make users

# copy binary to bin
make install

# set the environment variables and run the service
MF_USERS_LOG_LEVEL=[Users log level] \
MF_USERS_SECRET_KEY=[Secret key used to generate tokens] \
MF_USERS_ADMIN_EMAIL=[Default user, created on startup] \
MF_USERS_ADMIN_PASSWORD=[Default user password, created on startup] \
MF_USERS_PASS_REGEX=[Password regex] \
MF_USERS_ACCESS_TOKEN_DURATION=[Duration for an access token to be valid] \
MF_USERS_REFRESH_TOKEN_DURATION=[Duration for a refresh token to be valid] \
MF_TOKEN_RESET_ENDPOINT=[Password reset token endpoint] \
MF_USERS_HTTP_HOST=[Service HTTP host] \
MF_USERS_HTTP_PORT=[Service HTTP port] \
MF_USERS_HTTP_SERVER_CERT=[Path to server certificate] \
MF_USERS_HTTP_SERVER_KEY=[Path to server key] \
MF_USERS_GRPC_HOST=[Service GRPC host] \
MF_USERS_GRPC_PORT=[Service GRPC port] \
MF_USERS_GRPC_SERVER_CERT=[Path to server certificate] \
MF_USERS_GRPC_SERVER_KEY=[Path to server key] \
MF_USERS_DB_HOST=[Database host address] \
MF_USERS_DB_PORT=[Database host port] \
MF_USERS_DB_USER=[Database user] \
MF_USERS_DB_PASS=[Database password] \
MF_USERS_DB_NAME=[Name of the database used by the service] \
MF_USERS_DB_SSL_MODE=[SSL mode to connect to the database with] \
MF_USERS_DB_SSL_CERT=[Path to the PEM encoded certificate file] \
MF_USERS_DB_SSL_KEY=[Path to the PEM encoded key file] \
MF_USERS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] \
MF_EMAIL_HOST=[Mail server host] \
MF_EMAIL_PORT=[Mail server port] \
MF_EMAIL_USERNAME=[Mail server username] \
MF_EMAIL_PASSWORD=[Mail server password] \
MF_EMAIL_FROM_ADDRESS=[Email from address] \
MF_EMAIL_FROM_NAME=[Email from name] \
MF_EMAIL_TEMPLATE=[Email template file] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to Jaeger (true/false)] \
MF_USERS_INSTANCE_ID=[Instance ID] \
$GOBIN/mainflux-users
```

If `MF_EMAIL_TEMPLATE` doesn't point to any file service will function but password reset functionality will not work.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](https://api.mainflux.io/?urls.primaryName=users-openapi.yml).

[doc]: https://docs.mainflux.io
