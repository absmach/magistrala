# SMTP Notifier

SMTP Notifier implements notifier for send SMTP notifications.

## Configuration

The Subscription service using SMTP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                             | Default                        |
| --------------------------------- | ----------------------------------------------------------------------- | ------------------------------ |
| MG_SMTP_NOTIFIER_LOG_LEVEL        | Log level for SMT Notifier (debug, info, warn, error)                   | info                           |
| MG_SMTP_NOTIFIER_FROM_ADDRESS     | From address for SMTP notifications                                     |                                |
| MG_SMTP_NOTIFIER_CONFIG_PATH      | Path to the config file with message broker subjects configuration      | disable                        |
| MG_SMTP_NOTIFIER_HTTP_HOST        | SMTP Notifier service HTTP host                                         | localhost                      |
| MG_SMTP_NOTIFIER_HTTP_PORT        | SMTP Notifier service HTTP port                                         | 9015                           |
| MG_SMTP_NOTIFIER_HTTP_SERVER_CERT | SMTP Notifier service HTTP server certificate path                      | ""                             |
| MG_SMTP_NOTIFIER_HTTP_SERVER_KEY  | SMTP Notifier service HTTP server key                                   | ""                             |
| MG_SMTP_NOTIFIER_DB_HOST          | Database host address                                                   | localhost                      |
| MG_SMTP_NOTIFIER_DB_PORT          | Database host port                                                      | 5432                           |
| MG_SMTP_NOTIFIER_DB_USER          | Database user                                                           | magistrala                     |
| MG_SMTP_NOTIFIER_DB_PASS          | Database password                                                       | magistrala                     |
| MG_SMTP_NOTIFIER_DB_NAME          | Name of the database used by the service                                | subscriptions                  |
| MG_SMTP_NOTIFIER_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                        |
| MG_SMTP_NOTIFIER_DB_SSL_CERT      | Path to the PEM encoded cert file                                       | ""                             |
| MG_SMTP_NOTIFIER_DB_SSL_KEY       | Path to the PEM encoded certificate key                                 | ""                             |
| MG_SMTP_NOTIFIER_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           | ""                             |
| MG_JAEGER_URL                     | Jaeger server URL                                                       | http://jaeger:14268/api/traces |
| MG_MESSAGE_BROKER_URL             | Message broker URL                                                      | nats://127.0.0.1:4222          |
| MG_EMAIL_HOST                     | Mail server host                                                        | localhost                      |
| MG_EMAIL_PORT                     | Mail server port                                                        | 25                             |
| MG_EMAIL_USERNAME                 | Mail server username                                                    |                                |
| MG_EMAIL_PASSWORD                 | Mail server password                                                    |                                |
| MG_EMAIL_FROM_ADDRESS             | Email "from" address                                                    |                                |
| MG_EMAIL_FROM_NAME                | Email "from" name                                                       |                                |
| MG_EMAIL_TEMPLATE                 | Email template for sending notification emails                          | email.tmpl                     |
| MG_AUTH_GRPC_URL                  | Auth service gRPC URL                                                   | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT              | Auth service gRPC request timeout in seconds                            | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS           | Auth service gRPC TLS flag                                              | false                          |
| MG_AUTH_GRPC_CA_CERT              | Path to Auth service CA cert in pem format                              | ""                             |
| MG_AUTH_CLIENT_TLS                | Auth client TLS flag                                                    | false                          |
| MG_AUTH_CA_CERTS                  | Path to Auth client CA certs in pem format                              | ""                             |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server                           | true                           |
| MG_SMTP_NOTIFIER_INSTANCE_ID      | SMTP Notifier instance ID                                               | ""                             |

## Usage

Starting service will start consuming messages and sending emails when a message is received.

[doc]: https://docs.mainflux.io
