# OPC-UA Adapter

Adapter between Magistrala IoT system and an OPC-UA Server.

This adapter sits between Magistrala and an OPC-UA server and just forwards the messages from one system to another.

OPC-UA Server is used for connectivity layer and the data is pushed via this adapter service to Magistrala, where it is persisted and routed to other protocols via Magistrala multi-protocol message broker. Magistrala adds user accounts, application management and security in order to obtain the overall end-to-end OPC-UA solution.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                          | Description                                             | Default                             |
| --------------------------------- | ------------------------------------------------------- | ----------------------------------- |
| MG_OPCUA_ADAPTER_LOG_LEVEL        | Log level for the WS Adapter (debug, info, warn, error) | info                                |
| MG_OPCUA_ADAPTER_HTTP_HOST        | Service OPC-UA host                                     | ""                                  |
| MG_OPCUA_ADAPTER_HTTP_PORT        | Service WOPC-UAS port                                   | 8180                                |
| MG_OPCUA_ADAPTER_HTTP_SERVER_CERT | Path to the PEM encoded server certificate file         | ""                                  |
| MG_OPCUA_ADAPTER_HTTP_SERVER_KEY  | Path to the PEM encoded server key file                 | ""                                  |
| MG_OPCUA_ADAPTER_ROUTE_MAP_URL    | Route-map database URL                                  | <redis://localhost:6379/0>          |
| MG_ES_URL                         | Event source URL                                        | <nats://localhost:4222>             |
| MG_OPCUA_ADAPTER_EVENT_CONSUMER   | Service event consumer name                             | opcua-adapter                       |
| MG_MESSAGE_BROKER_URL             | Message broker instance URL                             | <nats://localhost:4222>             |
| MG_JAEGER_URL                     | Jaeger server URL                                       | <http://localhost:14268/api/traces> |
| MG_JAEGER_TRACE_RATIO             | Jaeger sampling ratio                                   | 1.0                                 |
| MG_SEND_TELEMETRY                 | Send telemetry to magistrala call home server           | true                                |
| MG_OPCUA_ADAPTER_INSTANCE_ID      | Service instance ID                                     | ""                                  |

## Deployment

The service itself is distributed as Docker container. Check the [`opcua-adapter`](https://github.com/absmach/magistrala/blob/master/docker/addons/opcua-adapter/docker-compose.yml) service section in docker-compose to see how service is deployed.

Running this service outside of container requires working instance of the message broker service, redis routemap server and Jaeger server.
To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the opcua-adapter
make opcua

# copy binary to bin
make install

# set the environment variables and run the service
MG_OPCUA_ADAPTER_LOG_LEVEL=info \
MG_OPCUA_ADAPTER_HTTP_HOST=localhost \
MG_OPCUA_ADAPTER_HTTP_PORT=8180 \
MG_OPCUA_ADAPTER_HTTP_SERVER_CERT="" \
MG_OPCUA_ADAPTER_HTTP_SERVER_KEY="" \
MG_OPCUA_ADAPTER_ROUTE_MAP_URL=redis://localhost:6379/0 \
MG_ES_URL=nats://localhost:4222 \
MG_OPCUA_ADAPTER_EVENT_CONSUMER=opcua-adapter \
MG_MESSAGE_BROKER_URL=nats://localhost:4222 \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_OPCUA_ADAPTER_INSTANCE_ID="" \
$GOBIN/magistrala-opcua
```

Setting `MG_LORA_ADAPTER_HTTP_SERVER_CERT` and `MG_LORA_ADAPTER_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is available in `<project_root>/docker/addons/opcua-adapter/docker-compose.yml`. In order to run Magistrala opcua-adapter, execute the following command:

```bash
docker-compose -f docker/addons/opcua-adapter/docker-compose.yml up -d
```

## Usage

For more information about service capabilities and its usage, please check out the [Magistrala documentation](https://docs.mainflux.io/opcua).
