# SMPP Notifier

SMPP Notifier implements notifier for send SMS notifications.

## Configuration

The Subscription service using SMPP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                                           | Default               |
| ------------------------------------| -------------------------------------------------------------------- | --------------------- |
| MF_SMPP_NOTIFIER_LOG_LEVEL          | Log level for SMPP Notifier (debug, info, warn, error)                | info                  |
| MF_SMPP_NOTIFIER_DB_HOST            | Database host address                                                 | localhost             |
| MF_SMPP_NOTIFIER_DB_PORT            | Database host port                                                    | 5432                  |
| MF_SMPP_NOTIFIER_DB_USER            | Database user                                                         | mainflux              |
| MF_SMPP_NOTIFIER_DB_PASS            | Database password                                                     | mainflux              |
| MF_SMPP_NOTIFIER_DB                 | Name of the database used by the service                              | subscriptions         |
| MF_SMPP_NOTIFIER_WRITER_CONFIG_PATH | DB connection SSL mode (disable, require, verify-ca, verify-full)     | disable               |
| MF_SMPP_NOTIFIER_DB_SSL_MODE        | Path to the PEM encoded certificate file                              |                       |
| MF_SMPP_NOTIFIER_DB_SSL_CERT        | Path to the PEM encoded key file                                      |                       |
| MF_SMPP_NOTIFIER_DB_SSL_KEY         | Path to the PEM encoded root certificate file                         |                       |
| MF_SMPP_NOTIFIER_DB_SSL_ROOT_CERT   | Users service HTTP port                                               | 9002                  |
| MF_SMPP_NOTIFIER_HTTP_PORT          | Path to server certificate in pem format                              |                       |
| MF_SMPP_NOTIFIER_SERVER_CERT        | Path to server cert in pem format                                     |                       |
| MF_SMPP_NOTIFIER_SERVER_KEY         | Path to server key in pem format                                      |                       |
| MF_JAEGER_URL                       | Jaeger server URL                                                     | localhost:6831        |
| MF_BROKER_URL                       | Message broker URL                                                    | nats://127.0.0.1:4222 |
| MF_SMPP_ADDRESS                     | SMPP address [host:port]                                              |                       |
| MF_SMPP_USERNAME                    | SMPP Username                                                         |                       |
| MF_SMPP_PASSWORD                    | SMPP Password                                                         |                       |
| MF_SMPP_SYSTEM_TYPE                 | SMPP System Type                                                      |                       |
| MF_SMPP_SRC_ADDR_TON                | SMPP source address TON                                               |                       |
| MF_SMPP_DST_ADDR_TON                | SMPP destination address TON                                          |                       |
| MF_SMPP_SRC_ADDR_NPI                | SMPP source address NPI                                               |                       |
| MF_SMPP_DST_ADDR_NPI                | SMPP destination address NPI                                          |                       |
| MF_AUTH_GRPC_TIMEOUT                | Users service gRPC request timeout in seconds                         | 1s                    |
| MF_AUTH_CLIENT_TLS                  | Users client TLS flag                                                 | false                 |
| MF_AUTH_CA_CERTS                    | Path to Auth client CA certs in pem format                            |                       |
| MF_SEND_TELEMETRY                   | Send telemetry to mainflux call home server                           | true                  |

## Usage

Starting service will start consuming messages and sending SMS when a message is received.

[doc]: http://mainflux.readthedocs.io
