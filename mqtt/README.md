# MQTT adapter

MQTT adapter provides an MQTT API for sending messages through the platform. MQTT adapter uses [mProxy](https://github.com/absmach/mproxy) for proxying traffic between client and MQTT broker.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                                   | Description                                                                         | Default                             |
| ------------------------------------------ | ----------------------------------------------------------------------------------- | ----------------------------------- |
| MG_MQTT_ADAPTER_LOG_LEVEL                 | Log level for the MQTT Adapter (debug, info, warn, error)                           | info                                |
| MG_MQTT_ADAPTER_MQTT_PORT                 | mProxy port                                                                         | 1883                                |
| MG_MQTT_ADAPTER_MQTT_TARGET_HOST          | MQTT broker host                                                                    | localhost                           |
| MG_MQTT_ADAPTER_MQTT_TARGET_PORT          | MQTT broker port                                                                    | 1883                                |
| MG_MQTT_ADAPTER_MQTT_QOS                  | MQTT broker QoS                                                                     | 1                                   |
| MG_MQTT_ADAPTER_FORWARDER_TIMEOUT         | MQTT forwarder for multiprotocol communication timeout                              | 30s                                 |
| MG_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK  | URL of broker health check                                                          | ""                                  |
| MG_MQTT_ADAPTER_WS_PORT                   | mProxy MQTT over WS port                                                            | 8080                                |
| MG_MQTT_ADAPTER_WS_TARGET_HOST            | MQTT broker host for MQTT over WS                                                   | localhost                           |
| MG_MQTT_ADAPTER_WS_TARGET_PORT            | MQTT broker port for MQTT over WS                                                   | 8080                                |
| MG_MQTT_ADAPTER_WS_TARGET_PATH            | MQTT broker MQTT over WS path                                                       | /mqtt                               |
| MG_MQTT_ADAPTER_CACHE_NUM_COUNTERS        | Number of cache counters to keep that hold access frequency information             | 200000                              |
| MG_MQTT_ADAPTER_CACHE_MAX_COST            | Maximum size of the cache(in bytes)                                                 | 1048576                             |
| MG_MQTT_ADAPTER_CACHE_BUFFER_ITEMS        | Number of cache `Get` buffers                                                       | 64                                  |
| MG_MQTT_ADAPTER_INSTANCE                  | Instance name for MQTT adapter                                                      | ""                                  |
| MG_CLIENTS_GRPC_URL                       | Clients service Auth gRPC URL                                                       | <localhost:7000>                    |
| MG_CLIENTS_GRPC_TIMEOUT                   | Clients service Auth gRPC request timeout in seconds                                | 1s                                  |
| MG_CLIENTS_GRPC_CLIENT_CERT               | Path to the PEM encoded clients service Auth gRPC client certificate file           | ""                                  |
| MG_CLIENTS_GRPC_CLIENT_KEY                | Path to the PEM encoded clients service Auth gRPC client key file                   | ""                                  |
| MG_CLIENTS_GRPC_SERVER_CERTS              | Path to the PEM encoded clients server Auth gRPC server trusted CA certificate file | ""                                  |
| MG_ES_URL                                 | Event sourcing URL                                                                  | <amqp://guest:guest@rabbitmq:5672/> |
| MG_MESSAGE_BROKER_URL                     | Message broker instance URL                                                         | <amqp://guest:guest@rabbitmq:5672/> |
| MG_JAEGER_URL                             | Jaeger server URL                                                                   | <http://localhost:4318/v1/traces>   |
| MG_JAEGER_TRACE_RATIO                     | Jaeger sampling ratio                                                               | 1.0                                 |
| MG_SEND_TELEMETRY                         | Send telemetry to supermq call home server                                          | true                                |
| MG_MQTT_ADAPTER_INSTANCE_ID               | Service instance ID                                                                 | ""                                  |
| MG_MQTT_ADAPTER_CERT_FILE                 | Path to the PEM encoded TLS certificate file for MQTT adapter                       | ""                                  |
| MG_MQTT_ADAPTER_KEY_FILE                  | Path to the PEM encoded TLS key file for MQTT adapter                               | ""                                  |
| MG_MQTT_ADAPTER_SERVER_CA_FILE            | Path to the PEM encoded server CA certificate file for MQTT adapter                 | ""                                  |
| MG_MQTT_ADAPTER_CLIENT_CA_FILE            | Path to the PEM encoded client CA certificate file for MQTT adapter                 | ""                                  |
| MG_MQTT_ADAPTER_OCSP_RESPONDER_URL        | URL of the OCSP responder for MQTT adapter                                          | ""                                  |
| MG_MQTT_ADAPTER_CERT_VERIFICATION_METHODS | Methods for certificate verification (e.g., ocsp)                                   | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`mqtt-adapter`](https://github.com/absmach/supermq/blob/main/docker/docker-compose.yaml) service section in docker-compose file to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, clients service and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/supermq

cd supermq

# compile the mqtt
make mqtt

# copy binary to bin
make install

# set the environment variables and run the service
MG_MQTT_ADAPTER_LOG_LEVEL=info \
MG_MQTT_ADAPTER_MQTT_PORT=1883 \
MG_MQTT_ADAPTER_MQTT_TARGET_HOST=localhost \
MG_MQTT_ADAPTER_MQTT_TARGET_PORT=1883 \
MG_MQTT_ADAPTER_MQTT_QOS=1 \
MG_MQTT_ADAPTER_FORWARDER_TIMEOUT=30s \
MG_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK="" \
MG_MQTT_ADAPTER_WS_PORT=8080 \
MG_MQTT_ADAPTER_WS_TARGET_HOST=localhost \
MG_MQTT_ADAPTER_WS_TARGET_PORT=8080 \
MG_MQTT_ADAPTER_WS_TARGET_PATH=/mqtt \
MG_MQTT_ADAPTER_CACHE_NUM_COUNTERS=200000 \
MG_MQTT_ADAPTER_CACHE_MAX_COST=1048576 \
MG_MQTT_ADAPTER_CACHE_BUFFER_ITEMS=64 \
MG_MQTT_ADAPTER_INSTANCE="" \
MG_CLIENTS_GRPC_URL=localhost:7000 \
MG_CLIENTS_GRPC_TIMEOUT=1s \
MG_CLIENTS_GRPC_CLIENT_CERT="" \
MG_CLIENTS_GRPC_CLIENT_KEY="" \
MG_CLIENTS_GRPC_SERVER_CERTS="" \
MG_ES_URL=amqp://guest:guest@rabbitmq:5672/ \
MG_MESSAGE_BROKER_URL=amqp://guest:guest@rabbitmq:5672/ \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_MQTT_ADAPTER_INSTANCE_ID="" \
$GOBIN/supermq-mqtt
```

Setting `MG_CLIENTS_GRPC_CLIENT_CERT` and `MG_CLIENTS_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `MG_CLIENTS_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

Setting `MG_MQTT_ADAPTER_CERT_FILE`, `MG_MQTT_ADAPTER_KEY_FILE`, and `MG_MQTT_ADAPTER_SERVER_CA_FILE` will enable TLS for incoming MQTT connections. The service expects a file in PEM format for both the certificate and the key. The service expects a file in PEM format of trusted CAs. Setting `MG_MQTT_ADAPTER_CLIENT_CA_FILE` will enable client certificate verification for incoming MQTT connections trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `MG_MQTT_ADAPTER_CERT_VERIFICATION_METHODS` to "ocsp" will enable OCSP verification for incoming MQTT connections. Setting `MG_MQTT_ADAPTER_OCSP_RESPONDER_URL` will set the OCSP responder URL for OCSP verification.

For more information about service capabilities and its usage, please check out the API documentation [API](https://github.com/absmach/supermq/blob/main/api/asyncapi/mqtt.yaml).
