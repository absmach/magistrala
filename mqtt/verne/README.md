# MQTT adapter

MQTT adapter provides an MQTT API for sending and receiving messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                | Description                      | Default               |
|-------------------------|----------------------------------|-----------------------|
| MF_NATS_URL             | NATS instance URL                | nats://localhost:4222 |
| MF_THINGS_AUTH_HTTP_URL | Things service HTTP URL for Auth | http://localhost:8989 |
| MF_MQTT_ADAPTER_ES_URL  | Redis ES URL                     | http://localhost:6379 |

Apart from this, VerneMQ configuration found
[here](https://github.com/ThingMesh/docker-vernemq/blob/master/vernemq.conf.default) can be customized.

When run in the Docker, 

```yaml
DOCKER_VERNEMQ_PLUGINS__VMQ_PASSWD: "off"
DOCKER_VERNEMQ_PLUGINS__VMQ_ACL: "off"
DOCKER_VERNEMQ_PLUGINS__MFX_AUTH: "on"
DOCKER_VERNEMQ_PLUGINS__MFX_AUTH__PATH: /mainflux/_build/default
DOCKER_VERNEMQ_LISTENER__WS__DEFAULT: "127.0.0.1:8880"
```

> N.B. in this Docker env var setup, `__` replaces `.` in the config file,
> so `plugins.mfx_auth.path` becomes `DOCKER_VERNEMQ_PLUGINS__MFX_AUTH__PATH`

## Deployment

### Docker
The service is distributed as Docker container. The following snippet provides
a compose file template that can be used to deploy the service container locally:

```yaml
version: "2"
services:
  mqtt-adapter:
      image: mainflux/mqtt:latest
      container_name: mainflux-mqtt
      depends_on:
        - things
        - nats
        - mqtt-redis
      restart: on-failure
      environment:
        MF_MQTT_ADAPTER_LOG_LEVEL: ${MF_MQTT_ADAPTER_LOG_LEVEL}
        MF_MQTT_INSTANCE_ID: mqtt-adapter
        MF_MQTT_ADAPTER_PORT: ${MF_MQTT_ADAPTER_PORT}
        MF_MQTT_ADAPTER_WS_PORT: ${MF_MQTT_ADAPTER_WS_PORT}
        MF_MQTT_ADAPTER_REDIS_URL: tcp://mqtt-redis:${MF_REDIS_TCP_PORT}
        MF_MQTT_ADAPTER_ES_URL: tcp://es-redis:${MF_REDIS_TCP_PORT}
        MF_NATS_URL: ${MF_NATS_URL}
        MF_THINGS_AUTH_HTTP_URL: http://things:${MF_THINGS_AUTH_HTTP_PORT}
        DOCKER_VERNEMQ_PLUGINS__VMQ_PASSWD: "off"
        DOCKER_VERNEMQ_PLUGINS__VMQ_ACL: "off"
        DOCKER_VERNEMQ_PLUGINS__MFX_AUTH: "on"
        DOCKER_VERNEMQ_PLUGINS__MFX_AUTH__PATH: /mainflux/_build/default
        DOCKER_VERNEMQ_LISTENER__WS__DEFAULT: "127.0.0.1:8880"
      ports:
        - ${MF_MQTT_ADAPTER_PORT}:${MF_MQTT_ADAPTER_PORT}
        - ${MF_MQTT_ADAPTER_WS_PORT}:${MF_MQTT_ADAPTER_WS_PORT}
      networks:
        - mainflux-base-net
```

### Native
#### Prepare
Install [gpb](https://github.com/tomas-abrahamsson/gpb)
```
git clone https://github.com/tomas-abrahamsson/gpb.git
cd gpb
git checkout 4.9.0
make -j 16
```

Then generate Erlang proto files:
```
mkdir -p ./src/proto
./gpb/bin/protoc-erl -I ./gpb/ ../*.proto -o ./src/proto
cp ./gpb/include/gpb.hrl ./src/proto/
```

If gRPC us used for auth (not enabled yet, to be enabled in the future):
```
git clone https://github.com/Bluehouse-Technology/grpc_client.git
cd grpc_client && make -j 16
make shell
```
Then in Erlang shell:
```
1> grpc_client:compile("../../internal.proto").
```

Outside of shell:
```
mv ./internal_client.erl ../src/proto
```

#### Compile
```
./rebar3 compile
```

#### Load Plugin

First start VerneMQ broker:
```
cd $VERNEMQ_BROKER_PATH
./_build/default/rel/vernemq/bin/vernemq start
```

Remove other plugins:
```
cd $VERNEMQ_BROKER_PATH
./_build/default/rel/vernemq/bin/vmq-admin plugin disable -n vmq_passwd
./_build/default/rel/vernemq/bin/vmq-admin plugin disable -n vmq_acl
```

Enable Mainflux `mfx_auth` plugin:
```
cd $VERNEMQ_BROKER_PATH
./_build/default/rel/vernemq/bin/vmq-admin plugin enable -n mfx_auth -p <path_to_mfx_auth_plugin>/_build/default
```

## Debugging
Inspect logs:
```
cd $VERNEMQ_BROKER_PATH
cat _build/default/rel/vernemq/log/console.log
cat _build/default/rel/vernemq/log/error.log
```


