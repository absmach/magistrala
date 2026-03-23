# Users

Users service provides an HTTP API for managing users. Through this API clients are able to do the following actions:

- register new accounts
- login
- manage account(s) (list, update, delete)

For in-depth explanation of the aforementioned scenarios, as well as thorough understanding of SuperMQ, please check out the [official documentation][doc].

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                                             | Default                           |
| --------------------------------- | ----------------------------------------------------------------------- | --------------------------------- |
| `MG_USERS_LOG_LEVEL`               | Log level for users service (debug, info, warn, error)                  | info                              |
| `MG_USERS_ADMIN_EMAIL`             | Default user, created on startup                                        | <admin@example.com>               |
| `MG_USERS_ADMIN_PASSWORD`          | Default user password, created on startup                               | 12345678                          |
| `MG_USERS_PASS_REGEX`              | Password regex                                                          | ^.{8,}$                           |
| `MG_USERS_HTTP_HOST`               | Users service HTTP host                                                 | localhost                         |
| `MG_USERS_HTTP_PORT`               | Users service HTTP port                                                 | 9002                              |
| `MG_USERS_HTTP_SERVER_CERT`        | Path to the PEM encoded server certificate file                         | ""                                |
| `MG_USERS_HTTP_SERVER_KEY`         | Path to the PEM encoded server key file                                 | ""                                |
| `MG_USERS_HTTP_SERVER_CA_CERTS`    | Path to the PEM encoded server CA certificate file                      | ""                                |
| `MG_USERS_HTTP_CLIENT_CA_CERTS`    | Path to the PEM encoded client CA certificate file                      | ""                                |
| `MG_AUTH_GRPC_URL`                 | Auth service GRPC URL                                                   | localhost:8181                    |
| `MG_AUTH_GRPC_TIMEOUT`             | Auth service GRPC timeout                                               | 1s                                |
| `MG_AUTH_GRPC_CLIENT_CERT`         | Path to the PEM encoded client certificate file                         | ""                                |
| `MG_AUTH_GRPC_CLIENT_KEY`          | Path to the PEM encoded client key file                                 | ""                                |
| `MG_AUTH_GRPC_SERVER_CA_CERTS`     | Path to the PEM encoded server CA certificate file                      | ""                                |
| `MG_USERS_DB_HOST`                 | Database host address                                                   | localhost                         |
| `MG_USERS_DB_PORT`                 | Database host port                                                      | 5432                              |
| `MG_USERS_DB_USER`                 | Database user                                                           | supermq                           |
| `MG_USERS_DB_PASS`                 | Database password                                                       | supermq                           |
| `MG_USERS_DB_NAME`                 | Name of the database used by the service                                | users                             |
| `MG_USERS_DB_SSL_MODE`             | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                           |
| `MG_USERS_DB_SSL_CERT`             | Path to the PEM encoded certificate file                                | ""                                |
| `MG_USERS_DB_SSL_KEY`              | Path to the PEM encoded key file                                        | ""                                |
| `MG_USERS_DB_SSL_ROOT_CERT`        | Path to the PEM encoded root certificate file                           | ""                                |
| `MG_EMAIL_HOST`                    | Mail server host                                                        | localhost                         |
| `MG_EMAIL_PORT`                    | Mail server port                                                        | 25                                |
| `MG_EMAIL_USERNAME`                | Mail server username                                                    | ""                                |
| `MG_EMAIL_PASSWORD`                | Mail server password                                                    | ""                                |
| `MG_EMAIL_FROM_ADDRESS`            | Email "from" address                                                    | ""                                |
| `MG_EMAIL_FROM_NAME`               | Email "from" name                                                       | ""                                |
| `MG_PASSWORD_RESET_URL_PREFIX`     | Password reset URL prefix                                               | <http://localhost/password/reset> |
| `MG_PASSWORD_RESET_EMAIL_TEMPLATE` | Password reset email template                                           | reset-password-email.tmpl         |
| `MG_VERIFICATION_URL_PREFIX`       | Verification URL prefix                                                 | <http://localhost/verify-email>   |
| `MG_VERIFICATION_EMAIL_TEMPLATE`   | Verification email template                                             | verification-email.tmpl           |
| `MG_USERS_ES_URL`                  | Event store URL                                                         | <nats://localhost:4222>           |
| `MG_JAEGER_URL`                    | Jaeger server URL                                                       | <http://localhost:4318/v1/traces> |
| `MG_OAUTH_UI_REDIRECT_URL`         | OAuth UI redirect URL                                                   | <http://localhost:9095/domains>   |
| `MG_OAUTH_UI_ERROR_URL`            | OAuth UI error URL                                                      | <http://localhost:9095/error>     |
| `MG_USERS_DELETE_INTERVAL`         | Interval for deleting users                                             | 24h                               |
| `MG_USERS_DELETE_AFTER`            | Time after which users are deleted                                      | 720h                              |
| `MG_JAEGER_TRACE_RATIO`            | Jaeger sampling ratio                                                   | 1.0                               |
| `MG_SEND_TELEMETRY`                | Send telemetry to supermq call home server.                             | true                              |
| `MG_USERS_INSTANCE_ID`             | SuperMQ instance ID                                                     | ""                                |

## Deployment

The service itself is distributed as Docker container. Check the [`users`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml) service section in docker-compose file to see how service is deployed.

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
MG_USERS_LOG_LEVEL=info \
MG_USERS_ADMIN_EMAIL=admin@example.com \
MG_USERS_ADMIN_PASSWORD=12345678 \
MG_USERS_PASS_REGEX="^.{8,}$" \
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
MG_USERS_DB_USER=supermq \
MG_USERS_DB_PASS=supermq \
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
MG_PASSWORD_RESET_URL_PREFIX=http://localhost:9002/password/reset \
MG_PASSWORD_RESET_EMAIL_TEMPLATE=docker/templates/reset-password-email.tmpl \
MG_VERIFICATION_URL_PREFIX=http://localhost:9002/users/verify-email \
MG_VERIFICATION_EMAIL_TEMPLATE=docker/templates/verification-email.tmpl \
MG_USERS_ES_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_OAUTH_UI_REDIRECT_URL=http://localhost:9095/domains \
MG_OAUTH_UI_ERROR_URL=http://localhost:9095/error \
MG_USERS_DELETE_INTERVAL=24h \
MG_USERS_DELETE_AFTER=720h \
MG_USERS_INSTANCE_ID="" \
$GOBIN/supermq-users
```

If `MG_EMAIL_TEMPLATE` doesn't point to any file service will function but password reset functionality will not work. The email environment variables are used to send emails with password reset link. The service expects a file in Go template format. The template should be something like [this](https://github.com/absmach/supermq/blob/main/docker/templates/users.tmpl).

Setting `MG_USERS_HTTP_SERVER_CERT` and `MG_USERS_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_USERS_HTTP_SERVER_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `MG_USERS_HTTP_CLIENT_CA_CERTS` will enable TLS against the service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

Setting `MG_AUTH_GRPC_CLIENT_CERT` and `MG_AUTH_GRPC_CLIENT_KEY` will enable TLS against the auth service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_AUTH_GRPC_SERVER_CA_CERTS` will enable TLS against the auth service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

## HTTP API

Base URL defaults to `http://localhost:9002`. Unless otherwise noted, endpoints require `Authorization: Bearer <access_token>`.

### Usage

| Operation | Description |
| --- | --- |
| Register | Create a user; optionally protected if self-registration is disabled. |
| Issue token | Exchange identity (email/username) and secret for access/refresh tokens. |
| Refresh token | Exchange a refresh token for a new access token. |
| Profile | Fetch the authenticated user profile. |
| List/search users | Page and filter users. |
| View user | Retrieve a user by ID . |
| Update user | Patch names/metadata/tags/profile picture; update email/username/role/tags/password via dedicated endpoints. |
| Status | Enable/disable a user  or delete a user. |
| Verification | Send verification email; verify via emailed link. |
| Password reset | Request a reset link and set a new password. |

### Best practices

- Disable self-registration in production; onboard users via admin tokens or your IdP.
- Keep `allow_unverified_user` false and require email verification before granting domain roles.
- Enforce TLS for HTTP and mTLS for gRPC by setting server/client cert env vars.
- Harden passwords with `MG_USERS_PASS_REGEX` and rotate credentials; purge stale accounts via `MG_USERS_DELETE_AFTER`.
- Rate-limit token issuance and password reset endpoints at your API gateway; export Prometheus metrics to watch for abuse.
- Store SMTP credentials and certificates in a secrets manager; avoid embedding secrets in images or repos.

### API examples

#### Register a user

```bash
curl -X POST "http://localhost:9002/users" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Ada",
    "last_name": "Lovelace",
    "credentials": { "username": "ada", "secret": "changeMe123" },
    "email": "ada@example.com",
    "role": 0,
    "status": 0,
    "tags": ["iot", "beta"],
    "metadata": { "team": "core" }
  }'
```

Expected response (201 Created):

```json
{
  "id": "c0b0c68c-5b93-4a93-8f1a-5d63a3f5c3c7",
  "first_name": "Ada",
  "last_name": "Lovelace",
  "email": "ada@example.com",
  "role": 0,
  "status": 0,
  "tags": ["iot", "beta"],
  "metadata": { "team": "core" },
  "created_at": "2024-10-24T13:31:52Z"
}
```

#### Issue access/refresh tokens (login)

```bash
curl -X POST "http://localhost:9002/users/tokens/issue" \
  -H "Content-Type: application/json" \
  -d '{ "identity": "ada@example.com", "secret": "changeMe123" }'
```

Expected response (201 Created):

```json
{
  "access_token": "eyJhbGciOi...",
  "refresh_token": "eyJhbGciOi...",
  "access_type": "Bearer"
}
```

#### View authenticated profile

```bash
curl -X GET "http://localhost:9002/users/profile" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Expected response:

```json
{
  "id": "c0b0c68c-5b93-4a93-8f1a-5d63a3f5c3c7",
  "first_name": "Ada",
  "last_name": "Lovelace",
  "email": "ada@example.com",
  "role": 0,
  "status": 0,
  "tags": ["iot", "beta"],
  "metadata": { "team": "core" },
  "verified_at": "2024-10-24T14:02:00Z",
  "created_at": "2024-10-24T13:31:52Z",
  "updated_at": "2024-10-24T14:02:00Z"
}
```

#### List users

```bash
curl -X GET "http://localhost:9002/users?limit=5&status=enabled&dir=desc" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Expected response:

```json
{
  "total": 2,
  "offset": 0,
  "limit": 5,
  "users": [
    {
      "id": "c0b0c68c-5b93-4a93-8f1a-5d63a3f5c3c7",
      "first_name": "Ada",
      "last_name": "Lovelace",
      "email": "ada@example.com",
      "role": 0,
      "status": 0,
      "tags": ["iot", "beta"],
      "created_at": "2024-10-24T13:31:52Z"
    }
  ]
}
```

#### Update user metadata and name

```bash
curl -X PATCH "http://localhost:9002/users/${USER_ID}" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Ada",
    "last_name": "Byron",
    "metadata": { "team": "edge" },
    "tags": ["edge", "beta"]
  }'
```

Expected response:

```json
{
  "id": "c0b0c68c-5b93-4a93-8f1a-5d63a3f5c3c7",
  "first_name": "Ada",
  "last_name": "Byron",
  "email": "ada@example.com",
  "tags": ["edge", "beta"],
  "metadata": { "team": "edge" },
  "status": 0,
  "role": 0,
  "updated_at": "2024-10-24T14:45:10Z",
  "updated_by": "a5b6c7d8-e901-4fab-9bcd-123456789abc"
}
```

#### Request password reset

```bash
curl -X POST "http://localhost:9002/password/reset-request" \
  -H "Content-Type: application/json" \
  -d '{ "email": "ada@example.com" }'
```

Expected response (201 Created):

```json
{ "msg": "Email with reset link is sent" }
```

#### Health check

```bash
curl -X GET "http://localhost:9002/health"
```

Expected response:

```json
{
  "status": "pass",
  "version": "0.18.0",
  "commit": "ffffffff",
  "description": "users service",
  "build_time": "1970-01-01_00:00:00",
  "instance_id": "b4f1d5d2-4f24-4c2a-9a40-123456789abc"
}
```

[doc]: https://docs.supermq.absmach.eu/
