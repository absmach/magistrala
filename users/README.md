# Users service

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

| Variable                  | Description                                                             | Default        |
|---------------------------|-------------------------------------------------------------------------|----------------|
| MF_USERS_LOG_LEVEL        | Log level for Users (debug, info, warn, error)                          | error          |
| MF_USERS_DB_HOST          | Database host address                                                   | localhost      |
| MF_USERS_DB_PORT          | Database host port                                                      | 5432           |
| MF_USERS_DB_USER          | Database user                                                           | mainflux       |
| MF_USERS_DB_PASSWORD      | Database password                                                       | mainflux       |
| MF_USERS_DB               | Name of the database used by the service                                | users          |
| MF_USERS_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable        |
| MF_USERS_DB_SSL_CERT      | Path to the PEM encoded certificate file                                |                |
| MF_USERS_DB_SSL_KEY       | Path to the PEM encoded key file                                        |                |
| MF_USERS_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           |                |
| MF_USERS_HTTP_PORT        | Users service HTTP port                                                 | 8180           |
| MF_USERS_GRPC_PORT        | Users service gRPC port                                                 | 8181           |
| MF_USERS_SERVER_CERT      | Path to server certificate in pem format                                |                |
| MF_USERS_SERVER_KEY       | Path to server key in pem format                                        |                |
| MF_USERS_SECRET           | String used for signing tokens                                          | users          |
| MF_JAEGER_URL             | Jaeger server URL                                                       | localhost:6831 |
| MF_EMAIL_DRIVER           | Mail server driver, mail server for sending reset password token        | smtp           |
| MF_EMAIL_HOST             | Mail server host                                                        | localhost      |
| MF_EMAIL_PORT             | Mail server port                                                        | 25             |
| MF_EMAIL_USERNAME         | Mail server username                                                    |                |
| MF_EMAIL_PASSWORD         | Mail server password                                                    |                |
| MF_EMAIL_FROM_ADDRESS     | Email "from" address                                                    |                |
| MF_EMAIL_FROM_NAME        | Email "from" name                                                       |                |
| MF_TOKEN_SECRET           | Token secret for signing token used for password reset                  |                |
| MF_TOKEN_DURATION         | Token duration in minutes                                               | 5              |
| MF_TOKEN_DEBUG_LEVEL      | Token util debug level                                                  | info           |
| MF_TOKEN_RESET_ENDPOINT   | Password request reset endpoint, for constructing link                  | /reset-request |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
services:
  users:
    image: mainflux/users:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:[configured HTTP port]
    environment:
      MF_USERS_LOG_LEVEL: [Users log level]
      MF_USERS_DB_HOST: [Database host address]
      MF_USERS_DB_PORT: [Database host port]
      MF_USERS_DB_USER: [Database user]
      MF_USERS_DB_PASS: [Database password]
      MF_USERS_DB: [Name of the database used by the service]
      MF_USERS_DB_SSL_MODE: [SSL mode to connect to the database with]
      MF_USERS_DB_SSL_CERT: [Path to the PEM encoded certificate file]
      MF_USERS_DB_SSL_KEY: [Path to the PEM encoded key file]
      MF_USERS_DB_SSL_ROOT_CERT: [Path to the PEM encoded root certificate file]
      MF_USERS_HTTP_PORT: [Service HTTP port]
      MF_USERS_GRPC_PORT: [Service gRPC port]
      MF_USERS_SECRET: [String used for signing tokens]
      MF_USERS_SERVER_CERT: [String path to server certificate in pem format]
      MF_USERS_SERVER_KEY: [String path to server key in pem format]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_EMAIL_DRIVER: [Mail server driver smtp]
      MF_EMAIL_HOST: [MF_EMAIL_HOST]
      MF_EMAIL_PORT: [MF_EMAIL_PORT]
      MF_EMAIL_USERNAME: [MF_EMAIL_USERNAME]
      MF_EMAIL_PASSWORD: [MF_EMAIL_PASSWORD]
      MF_EMAIL_FROM_ADDRESS: [MF_EMAIL_FROM_ADDRESS]
      MF_EMAIL_FROM_NAME: [MF_EMAIL_FROM_NAME]
      MF_TOKEN_SECRET: [MF_TOKEN_SECRET]
      MF_TOKEN_DURATION: [MF_TOKEN_DURATION]
      MF_TOKEN_DEBUG_LEVEL: [MF_TOKEN_DEBUG_LEVEL]
      MF_TOKEN_RESET_ENDPOINT: [MF_TOKEN_RESET_ENDPOINT]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the service
make users

# copy binary to bin
make install

# set the environment variables and run the service
MF_USERS_LOG_LEVEL=[Users log level] MF_USERS_DB_HOST=[Database host address] MF_USERS_DB_PORT=[Database host port] MF_USERS_DB_USER=[Database user] MF_USERS_DB_PASS=[Database password] MF_USERS_DB=[Name of the database used by the service] MF_USERS_DB_SSL_MODE=[SSL mode to connect to the database with] MF_USERS_DB_SSL_CERT=[Path to the PEM encoded certificate file] MF_USERS_DB_SSL_KEY=[Path to the PEM encoded key file] MF_USERS_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] MF_USERS_HTTP_PORT=[Service HTTP port] MF_USERS_GRPC_PORT=[Service gRPC port] MF_USERS_SECRET=[String used for signing tokens] MF_USERS_SERVER_CERT=[Path to server certificate] MF_USERS_SERVER_KEY=[Path to server key] MF_JAEGER_URL=[Jaeger server URL] $GOBIN/mainflux-users
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).

[doc]: http://mainflux.readthedocs.io
