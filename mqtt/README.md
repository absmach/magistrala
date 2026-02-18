# MQTT adapter

MQTT adapter provides an MQTT API for sending messages through the platform. MQTT adapter uses [mProxy](https://github.com/absmach/mproxy) for proxying traffic between client and MQTT broker.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                                   | Description                                                                         | Default                             |
| ------------------------------------------ | ----------------------------------------------------------------------------------- | ----------------------------------- |
| SMQ_MQTT_ADAPTER_LOG_LEVEL                 | Log level for the MQTT Adapter (debug, info, warn, error)                           | info                                |
| SMQ_MQTT_ADAPTER_MQTT_PORT                 | mProxy port                                                                         | 1883                                |
| SMQ_MQTT_ADAPTER_MQTT_TARGET_HOST          | MQTT broker host                                                                    | localhost                           |
| SMQ_MQTT_ADAPTER_MQTT_TARGET_PORT          | MQTT broker port                                                                    | 1883                                |
| SMQ_MQTT_ADAPTER_MQTT_QOS                  | MQTT broker QoS                                                                     | 1                                   |
| SMQ_MQTT_ADAPTER_FORWARDER_TIMEOUT         | MQTT forwarder for multiprotocol communication timeout                              | 30s                                 |
| SMQ_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK  | URL of broker health check                                                          | ""                                  |
| SMQ_MQTT_ADAPTER_WS_PORT                   | mProxy MQTT over WS port                                                            | 8080                                |
| SMQ_MQTT_ADAPTER_WS_TARGET_HOST            | MQTT broker host for MQTT over WS                                                   | localhost                           |
| SMQ_MQTT_ADAPTER_WS_TARGET_PORT            | MQTT broker port for MQTT over WS                                                   | 8080                                |
| SMQ_MQTT_ADAPTER_WS_TARGET_PATH            | MQTT broker MQTT over WS path                                                       | /mqtt                               |
| SMQ_MQTT_ADAPTER_CACHE_NUM_COUNTERS        | Number of cache counters to keep that hold access frequency information             | 200000                              |
| SMQ_MQTT_ADAPTER_CACHE_MAX_COST            | Maximum size of the cache(in bytes)                                                 | 1048576                             |
| SMQ_MQTT_ADAPTER_CACHE_BUFFER_ITEMS        | Number of cache `Get` buffers                                                       | 64                                  |
| SMQ_MQTT_ADAPTER_INSTANCE                  | Instance name for MQTT adapter                                                      | ""                                  |
| SMQ_CLIENTS_GRPC_URL                       | Clients service Auth gRPC URL                                                       | <localhost:7000>                    |
| SMQ_CLIENTS_GRPC_TIMEOUT                   | Clients service Auth gRPC request timeout in seconds                                | 1s                                  |
| SMQ_CLIENTS_GRPC_CLIENT_CERT               | Path to the PEM encoded clients service Auth gRPC client certificate file           | ""                                  |
| SMQ_CLIENTS_GRPC_CLIENT_KEY                | Path to the PEM encoded clients service Auth gRPC client key file                   | ""                                  |
| SMQ_CLIENTS_GRPC_SERVER_CERTS              | Path to the PEM encoded clients server Auth gRPC server trusted CA certificate file | ""                                  |
| SMQ_ES_URL                                 | Event sourcing URL                                                                  | <amqp://guest:guest@rabbitmq:5672/> |
| SMQ_MESSAGE_BROKER_URL                     | Message broker instance URL                                                         | <amqp://guest:guest@rabbitmq:5672/> |
| SMQ_JAEGER_URL                             | Jaeger server URL                                                                   | <http://localhost:4318/v1/traces>   |
| SMQ_JAEGER_TRACE_RATIO                     | Jaeger sampling ratio                                                               | 1.0                                 |
| SMQ_SEND_TELEMETRY                         | Send telemetry to supermq call home server                                          | true                                |
| SMQ_MQTT_ADAPTER_INSTANCE_ID               | Service instance ID                                                                 | ""                                  |
| SMQ_MQTT_ADAPTER_CERT_FILE                 | Path to the PEM encoded TLS certificate file for MQTT adapter                       | ""                                  |
| SMQ_MQTT_ADAPTER_KEY_FILE                  | Path to the PEM encoded TLS key file for MQTT adapter                               | ""                                  |
| SMQ_MQTT_ADAPTER_SERVER_CA_FILE            | Path to the PEM encoded server CA certificate file for MQTT adapter                 | ""                                  |
| SMQ_MQTT_ADAPTER_CLIENT_CA_FILE            | Path to the PEM encoded client CA certificate file for MQTT adapter                 | ""                                  |
| SMQ_MQTT_ADAPTER_OCSP_RESPONDER_URL        | URL of the OCSP responder for MQTT adapter                                          | ""                                  |
| SMQ_MQTT_ADAPTER_CERT_VERIFICATION_METHODS | Methods for certificate verification (e.g., ocsp)                                   | ""                                  |

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
SMQ_MQTT_ADAPTER_LOG_LEVEL=info \
SMQ_MQTT_ADAPTER_MQTT_PORT=1883 \
SMQ_MQTT_ADAPTER_MQTT_TARGET_HOST=localhost \
SMQ_MQTT_ADAPTER_MQTT_TARGET_PORT=1883 \
SMQ_MQTT_ADAPTER_MQTT_QOS=1 \
SMQ_MQTT_ADAPTER_FORWARDER_TIMEOUT=30s \
SMQ_MQTT_ADAPTER_MQTT_TARGET_HEALTH_CHECK="" \
SMQ_MQTT_ADAPTER_WS_PORT=8080 \
SMQ_MQTT_ADAPTER_WS_TARGET_HOST=localhost \
SMQ_MQTT_ADAPTER_WS_TARGET_PORT=8080 \
SMQ_MQTT_ADAPTER_WS_TARGET_PATH=/mqtt \
SMQ_MQTT_ADAPTER_CACHE_NUM_COUNTERS=200000 \
SMQ_MQTT_ADAPTER_CACHE_MAX_COST=1048576 \
SMQ_MQTT_ADAPTER_CACHE_BUFFER_ITEMS=64 \
SMQ_MQTT_ADAPTER_INSTANCE="" \
SMQ_CLIENTS_GRPC_URL=localhost:7000 \
SMQ_CLIENTS_GRPC_TIMEOUT=1s \
SMQ_CLIENTS_GRPC_CLIENT_CERT="" \
SMQ_CLIENTS_GRPC_CLIENT_KEY="" \
SMQ_CLIENTS_GRPC_SERVER_CERTS="" \
SMQ_ES_URL=amqp://guest:guest@rabbitmq:5672/ \
SMQ_MESSAGE_BROKER_URL=amqp://guest:guest@rabbitmq:5672/ \
SMQ_JAEGER_URL=http://localhost:14268/api/traces \
SMQ_JAEGER_TRACE_RATIO=1.0 \
SMQ_SEND_TELEMETRY=true \
SMQ_MQTT_ADAPTER_INSTANCE_ID="" \
$GOBIN/supermq-mqtt
```

Setting `SMQ_CLIENTS_GRPC_CLIENT_CERT` and `SMQ_CLIENTS_GRPC_CLIENT_KEY` will enable TLS against the clients service. The service expects a file in PEM format for both the certificate and the key. Setting `SMQ_CLIENTS_GRPC_SERVER_CERTS` will enable TLS against the clients service trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs.

Setting `SMQ_MQTT_ADAPTER_CERT_FILE`, `SMQ_MQTT_ADAPTER_KEY_FILE`, and `SMQ_MQTT_ADAPTER_SERVER_CA_FILE` will enable TLS for incoming MQTT connections. The service expects a file in PEM format for both the certificate and the key. The service expects a file in PEM format of trusted CAs. Setting `SMQ_MQTT_ADAPTER_CLIENT_CA_FILE` will enable client certificate verification for incoming MQTT connections trusting only those CAs that are provided. The service expects a file in PEM format of trusted CAs. Setting `SMQ_MQTT_ADAPTER_CERT_VERIFICATION_METHODS` to "ocsp" will enable OCSP verification for incoming MQTT connections. Setting `SMQ_MQTT_ADAPTER_OCSP_RESPONDER_URL` will set the OCSP responder URL for OCSP verification.

For more information about service capabilities and its usage, please check out the API documentation [API](https://github.com/absmach/supermq/blob/main/api/asyncapi/mqtt.yaml).
