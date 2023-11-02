# OPC-UA Adapter

Adapter between Magistrala IoT system and an OPC-UA Server.

This adapter sits between Magistrala and an OPC-UA server and just forwards the messages from one system to another.

OPC-UA Server is used for connectivity layer and the data is pushed via this adapter service to Magistrala, where it is persisted and routed to other protocols via Magistrala multi-protocol message broker. Magistrala adds user accounts, application management and security in order to obtain the overall end-to-end OPC-UA solution.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                        | Description                                   | Default                          |
| ------------------------------- | --------------------------------------------- | -------------------------------- |
| MG_OPCUA_ADAPTER_HTTP_PORT      | Service HTTP port                             | 8180                             |
| MG_OPCUA_ADAPTER_LOG_LEVEL      | Service Log level                             | info                             |
| MG_MESSAGE_BROKER_URL           | Message broker instance URL                   | <nats://localhost:4222>          |
| MG_OPCUA_ADAPTER_INTERVAL_MS    | OPC-UA Server Interval in milliseconds        | 1000                             |
| MG_OPCUA_ADAPTER_POLICY         | OPC-UA Server Policy                          |                                  |
| MG_OPCUA_ADAPTER_MODE           | OPC-UA Server Mode                            |                                  |
| MG_OPCUA_ADAPTER_CERT_FILE      | OPC-UA Server Certificate file                |                                  |
| MG_OPCUA_ADAPTER_KEY_FILE       | OPC-UA Server Key file                        |                                  |
| MG_OPCUA_ADAPTER_ROUTE_MAP_URL  | Route-map database URL                        | <redis://localhost:6379/0>       |
| MG_THINGS_ES_URL                | Things service event source URL               | <localhost:6379>                 |
| MG_THINGS_ES_PASS               | Things service event source password          |                                  |
| MG_THINGS_ES_DB                 | Things service event source DB                | 0                                |
| MG_OPCUA_ADAPTER_EVENT_CONSUMER | Service event consumer name                   | opcua                            |
| MG_JAEGER_URL                   | Jaeger server URL                             | <http://jaeger:14268/api/traces> |
| MG_SEND_TELEMETRY               | Send telemetry to magistrala call home server | true                             |

## Deployment

The service itself is distributed as Docker container. Check the [`opcua-adapter`](https://github.com/absmach/magistrala/blob/master/docker/addons/opcua-adapter/docker-compose.yml#L29-L53) service section in
docker-compose to see how service is deployed.

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
MG_OPCUA_ADAPTER_HTTP_PORT=[Service HTTP port] \
MG_OPCUA_ADAPTER_LOG_LEVEL=[OPC-UA Adapter Log Level] \
MG_MESSAGE_BROKER_URL=[Message broker instance URL] \
MG_OPCUA_ADAPTER_INTERVAL_MS: [OPC-UA Server Interval (milliseconds)] \
MG_OPCUA_ADAPTER_POLICY=[OPC-UA Server Policy] \
MG_OPCUA_ADAPTER_MODE=[OPC-UA Server Mode] \
MG_OPCUA_ADAPTER_CERT_FILE=[OPC-UA Server Certificate file] \
MG_OPCUA_ADAPTER_KEY_FILE=[OPC-UA Server Key file] \
MG_OPCUA_ADAPTER_ROUTE_MAP_URL=[Route-map database URL] \
MG_THINGS_ES_URL=[Things service event source URL] \
MG_THINGS_ES_PASS=[Things service event source password] \
MG_THINGS_ES_DB=[Things service event source password] \
MG_OPCUA_ADAPTER_EVENT_CONSUMER=[OPC-UA adapter instance name] \
$GOBIN/magistrala-opcua
```

### Using docker-compose

This service can be deployed using docker containers.
Docker compose file is available in `<project_root>/docker/addons/opcua-adapter/docker-compose.yml`. In order to run Magistrala opcua-adapter, execute the following command:

```bash
docker-compose -f docker/addons/opcua-adapter/docker-compose.yml up -d
```

## Usage

For more information about service capabilities and its usage, please check out
the [Magistrala documentation](https://docs.mainflux.io/opcua).
