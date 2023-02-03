# SMTP Notifier

SMTP Notifier implements notifier for send SMTP notifications.

## Configuration

The Subscription service using SMTP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                             | Default               |
| --------------------------------- | ----------------------------------------------------------------------- | --------------------- |
| MF_SMTP_NOTIFIER_LOG_LEVEL        | Log level for SMT Notifier (debug, info, warn, error)                   | info                  |
| MF_SMTP_NOTIFIER_DB_HOST          | Database host address                                                   | localhost             |
| MF_SMTP_NOTIFIER_DB_PORT          | Database host port                                                      | 5432                  |
| MF_SMTP_NOTIFIER_DB_USER          | Database user                                                           | mainflux              |
| MF_SMTP_NOTIFIER_DB_PASS          | Database password                                                       | mainflux              |
| MF_SMTP_NOTIFIER_DB               | Name of the database used by the service                                | subscriptions         |
| MF_SMTP_NOTIFIER_CONFIG_PATH      | Path to the config file with message broker subjects configuration      | disable               |
| MF_SMTP_NOTIFIER_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) |                       |
| MF_SMTP_NOTIFIER_DB_SSL_CERT      | Path to the PEM encoded cert file                                       |                       |
| MF_SMTP_NOTIFIER_DB_SSL_KEY       | Path to the PEM encoded certificate key                                 |                       |
| MF_SMTP_NOTIFIER_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           |                       |
| MF_SMTP_NOTIFIER_PORT             | HTTP server port                                                        | 8180                  |
| MF_SMTP_NOTIFIER_SERVER_CERT      | Path to server cert in pem format                                       |                       |
| MF_SMTP_NOTIFIER_SERVER_KEY       | Path to server key in pem format                                        |                       |
| MF_JAEGER_URL                     | Jaeger server URL                                                       | localhost:6831        |
| MF_BROKER_URL                     | Message broker URL                                                      | nats://127.0.0.1:4222 |
| MF_EMAIL_HOST                     | Mail server host                                                        | localhost             |
| MF_EMAIL_PORT                     | Mail server port                                                        | 25                    |
| MF_EMAIL_USERNAME                 | Mail server username                                                    |                       |
| MF_EMAIL_PASSWORD                 | Mail server password                                                    |                       |
| MF_EMAIL_FROM_ADDRESS             | Email "from" address                                                    |                       |
| MF_EMAIL_FROM_NAME                | Email "from" name                                                       |                       |
| MF_EMAIL_TEMPLATE                 | Email template for sending notification emails                          | email.tmpl            |
| MF_AUTH_GRPC_URL                  | Auth service gRPC URL                                                   | localhost:8181        |
| MF_AUTH_GRPC_TIMEOUT              | Auth service gRPC request timeout in seconds                            | 1s                    |
| MF_AUTH_CLIENT_TLS                | Auth client TLS flag                                                    | false                 |
| MF_AUTH_CA_CERTS                  | Path to Auth client CA certs in pem format                              |                       |

## Usage

Starting service will start consuming messages and sending emails when a message is received.

[doc]: https://docs.mainflux.io
