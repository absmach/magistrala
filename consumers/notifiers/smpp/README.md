# SMPP Notifier

SMPP Notifier implements notifier for send SMS notifications.

## Configuration

The Subscription service using SMPP Notifier is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                          | Description                                                                       | Default                        |
| --------------------------------- | --------------------------------------------------------------------------------- | ------------------------------ |
| MG_SMPP_NOTIFIER_LOG_LEVEL        | Log level for SMPP Notifier (debug, info, warn, error)                            | info                           |
| MG_SMPP_NOTIFIER_FROM_ADDRESS     | From address for SMS notifications                                                |                                |
| MG_SMPP_NOTIFIER_CONFIG_PATH      | Config file path with Message broker subjects list, payload type and content-type | /config.toml                   |
| MG_SMPP_NOTIFIER_HTTP_HOST        | Service HTTP host                                                                 | localhost                      |
| MG_SMPP_NOTIFIER_HTTP_PORT        | Service HTTP port                                                                 | 9014                           |
| MG_SMPP_NOTIFIER_HTTP_SERVER_CERT | Service HTTP server certificate path                                              | ""                             |
| MG_SMPP_NOTIFIER_HTTP_SERVER_KEY  | Service HTTP server key                                                           | ""                             |
| MG_SMPP_NOTIFIER_DB_HOST          | Database host address                                                             | localhost                      |
| MG_SMPP_NOTIFIER_DB_PORT          | Database host port                                                                | 5432                           |
| MG_SMPP_NOTIFIER_DB_USER          | Database user                                                                     | magistrala                     |
| MG_SMPP_NOTIFIER_DB_PASS          | Database password                                                                 | magistrala                     |
| MG_SMPP_NOTIFIER_DB_NAME          | Name of the database used by the service                                          | subscriptions                  |
| MG_SMPP_NOTIFIER_DB_SSL_MODE      | DB connection SSL mode (disable, require, verify-ca, verify-full)                 | disable                        |
| MG_SMPP_NOTIFIER_DB_SSL_CERT      | Path to the PEM encoded certificate file                                          | ""                             |
| MG_SMPP_NOTIFIER_DB_SSL_KEY       | Path to the PEM encoded key file                                                  | ""                             |
| MG_SMPP_NOTIFIER_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                                     | ""                             |
| MG_SMPP_ADDRESS                   | SMPP address [host:port]                                                          |                                |
| MG_SMPP_USERNAME                  | SMPP Username                                                                     |                                |
| MG_SMPP_PASSWORD                  | SMPP Password                                                                     |                                |
| MG_SMPP_SYSTEM_TYPE               | SMPP System Type                                                                  |                                |
| MG_SMPP_SRC_ADDR_TON              | SMPP source address TON                                                           |                                |
| MG_SMPP_DST_ADDR_TON              | SMPP destination address TON                                                      |                                |
| MG_SMPP_SRC_ADDR_NPI              | SMPP source address NPI                                                           |                                |
| MG_SMPP_DST_ADDR_NPI              | SMPP destination address NPI                                                      |                                |
| MG_AUTH_GRPC_URL                  | Auth service gRPC URL                                                             | localhost:7001                 |
| MG_AUTH_GRPC_TIMEOUT              | Auth service gRPC request timeout in seconds                                      | 1s                             |
| MG_AUTH_GRPC_CLIENT_TLS           | Auth client TLS flag                                                              | false                          |
| MG_AUTH_GRPC_CA_CERT              | Path to Auth client CA certs in pem format                                        | ""                             |
| MG_MESSAGE_BROKER_URL             | Message broker URL                                                                | nats://127.0.0.1:4222          |
| MG_JAEGER_URL                     | Jaeger server URL                                                                 | http://jaeger:14268/api/traces |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server                                     | true                           |
| MG_SMPP_NOTIFIER_INSTANCE_ID      | SMPP Notifier instance ID                                                         | ""                             |

## Usage

Starting service will start consuming messages and sending SMS when a message is received.

[doc]: http://mainflux.readthedocs.io
