## Getting Mainflux

Mainflux can be fetched from the official [Mainflux GitHub repository](https://github.com/Mainflux/mainflux):

```
go get github.com/mainflux/mainflux
cd $GOPATH/src/github.com/mainflux/mainflux
```

## Building

### Prerequisites
Make sure that you have [Protocol Buffers](https://developers.google.com/protocol-buffers/) compiler (`protoc`) installed.

[Go Protobuf](https://github.com/golang/protobuf) installation instructions are [here](https://github.com/golang/protobuf#installation).
Go Protobuf uses C bindings, so you will need to install [C++ protobuf](https://github.com/google/protobuf) as a prerequisite.
Mainflux uses `Protocol Buffers for Go with Gadgets` to generate faster marshaling and unmarshaling Go code. Protocol Buffers for Go with Gadgets instalation instructions can be found (here)(https://github.com/gogo/protobuf).

### Build All Services

Use `GNU Make` tool to build all Mainflux services:

```
make
```

Build artefacts will be put in the `build` directory.

> N.B. All Mainflux services are built as a statically linked binaries. This way they can be portable (transferred to any platform just by placing them there and running them) as they contain all needed libraries and do not relay on shared system libraries. This helps creating [FROM scratch](https://hub.docker.com/_/scratch/) dockers.

### Build Individual Microservice
Individual microservices can be built with:

```
make <microservice_name>
```

For example:

```
make http
```

will build the HTTP Adapter microservice.

### Building Dockers

Dockers can be built with:

```
make dockers
```

or individually with:

```
make docker_<microservice_name>
```

For example:

```
make docker_http
```

> N.B. Mainflux creates `FROM scratch` docker containers which are compact and small in size.

> N.B. The `things-db` and `users-db` containers are built from a vanilla PostgreSQL docker image downloaded from docker hub which does not persist the data when these containers are rebuilt. Thus, __rebuilding of all docker containers with `make dockers` or rebuilding the `things-db` and `users-db` containers separately with `make docker_things-db` and `make docker_users-db` respectively, will cause data loss. All your users, things, channels and connections between them will be lost!__ As we use this setup only for development, we don't guarantee any permanent data persistence. If you need to retain the data between the container rebuilds you can attach volume to the `things-db` and `users-db` containers. Check the official docs on how to use volumes [here](https://docs.docker.com/storage/volumes/) and [here](https://docs.docker.com/compose/compose-file/#volumes). For examples on how to add persistent volumes check the [Overriding the default docker-compose configuration](#overriding-the-default-docker-compose-configuration) section.

#### Building Docker images for development

In order to speed up build process, you can use commands such as:

```bash
make dockers_dev
```

or individually with

```bash
make docker_dev_<microservice_name>
```

Commands `make dockers` and `make dockers_dev` are similar. The main difference is that building images in the development mode is done on the local machine, rather than an intermediate image, which makes building images much faster. Before running this command, corresponding binary needs to be built in order to make changes visible. This can be done using `make` or `make <service_name>` command. Commands `make dockers_dev` and `make docker_dev_<service_name>` should be used only for development to speed up the process of image building. **For deployment images, commands from section above should be used.**

### Overriding the default docker-compose configuration
Sometimes, depending on the use case and the user's needs it might be useful to override or add some extra parameters to the docker-compose configuration. These configuration changes can be done by specifying multiple compose files with the [docker-compose command line option -f](https://docs.docker.com/compose/reference/overview/) as described [here](https://docs.docker.com/compose/extends/).
The following format of the `docker-compose` can be used to extend or override the configuration:
```
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.custom1.yml -f docker/docker-compose.custom2.yml up [-d]
```
In the command above each successive file overrides the previous parameters.

A practical example in our case would be to add persistent volumes to the users-db, things-db, influxdb and grafana containers so that we don't loose data between updates of Mainflux. The volumes are mapped to the default location on the host machine, something like `/var/lib/docker/volumes/project-name_volume-name`:

```yaml
# docker/docker-compose.persistence.yml
version: "3"

volumes:
  mainflux-users-db-volume:
  mainflux-things-db-volume:

services:
  users-db:
    volumes:
      - mainflux-users-db-volume:/var/lib/postgresql/data

  things-db:
    volumes:
      - mainflux-things-db-volume:/var/lib/postgresql/data
```

```yaml
# docker/addons/influxdb-writer/docker-compose.persistence.yml
version: "3"

volumes:
  influxdb-volume:
  grafana-volume:

services:
  influxdb:
    volumes:
      - influxdb-volume:/var/lib/influxdb

  grafana:
    volumes:
      - grafana-volume:/var/lib/grafana
```
When we have the override files in place, to compose the whole infrastructure including the persistent volumes we can execute:
```
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.persistence.yml up -d
docker-compose -f docker/addons/influxdb-writer/docker-compose.yml -f docker/addons/influxdb-writer/docker-compose.persistence.yml up -d
```

__Note:__ Please store your customizations to some folder outside the Mainflux's source folder and maybe add them to some other git repository. You can always apply your customizations by pointing to the right file using `docker-compose -f ...`. Also be sure not to use the `make cleandocker` and `make cleanghost` as they might delete something which you don't want to delete (i.e. the newly created persistent volumes).

### MQTT Microservice
The MQTT Microservice in Mainflux is special, as it is currently the only microservice written in NodeJS. It is not compiled, but node modules need to be downloaded in order to start the service:

```
cd mqtt
npm install
```

Note that there is a shorthand for doing these commands with `make` tool:

```
make mqtt
```

After that, the MQTT Adapter can be started from top directory (as it needs to find `*.proto` files) with:
```
node mqtt/mqtt.js
```

### Protobuf
If you've made any changes to `.proto` files, you should call `protoc` command prior to compiling individual microservices.

To do this by hand, execute:

```
protoc --gofast_out=plugins=grpc:. *.proto
```

A shorthand to do this via `make` tool is:

```
make proto
```

> N.B. This must be done once at the beginning in order to generate protobuf Go structures needed for the build. However, if you don't change any of `.proto` files, this step is not mandatory, since all generated files are included in the repo (those are files with `.pb.go` extension).

### Cross-compiling for ARM
Mainflux can be compiled for ARM platform and run on Raspberry Pi or other similar IoT gateways, by following the instructions [here](https://dave.cheney.net/2015/08/22/cross-compilation-with-go-1-5) or [here](https://www.alexruf.net/golang/arm/raspberrypi/2016/01/16/cross-compile-with-go-1-5-for-raspberry-pi.html) as well as information
found [here](https://github.com/golang/go/wiki/GoArm). The environment variables `GOARCH=arm` and `GOARM=7` must be set for the compilation.

Cross-compilation for ARM with Mainflux make:

```
GOOS=linux GOARCH=arm GOARM=7 make
```

## Running tests
To run all of the tests you can execute:
```
make test
```
Dockertest is used for the tests, so to run them, you will need the Docker daemon/service running.

## Installing
Installing Go binaries is simple: just move them from `build` to `$GOBIN` (do not fortget to add `$GOBIN` to your `$PATH`).

You can execute:

```
make install
```

which will do this copying of the binaries.

> N.B. Only Go binaries will be installed this way. The MQTT adapter is a NodeJS script and will stay in the `mqtt` dir.

## Deployment

### Prerequisites
Mainflux depends on several infrastructural services, notably [NATS](https://www.nats.io/) broker and [PostgreSQL](https://www.postgresql.org/) database.

#### NATS
Mainflux uses NATS as it's central message bus. For development purposes (when not run via Docker), it expects that NATS is installed on the local system.

To do this execute:

```
go get github.com/nats-io/gnatsd
```

This will install `gnatsd` binary that can be simply run by executing:

```
gnatsd
```

#### PostgreSQL
Mainflux uses PostgreSQL to store metadata (`users`, `things` and `channels` entities alongside with authorization tokens).
It expects that PostgreSQL DB is installed, set up and running on the local system.

Information how to set-up (prepare) PostgreSQL database can be found [here](https://support.rackspace.com/how-to/postgresql-creating-and-dropping-roles/),
and it is done by executing following commands:

```
# Create `users` and `things` databases
sudo -u postgres createdb users
sudo -u postgres createdb things

# Set-up Postgres roles
sudo su - postgres
psql -U postgres
postgres=# CREATE ROLE mainflux WITH LOGIN ENCRYPTED PASSWORD 'mainflux';
postgres=# ALTER USER mainflux WITH LOGIN ENCRYPTED PASSWORD 'mainflux';
```

### Mainflux Services
Running of the Mainflux microservices can be tricky, as there is a lot of them and each demand configuration in the form of environment variables.

The whole system (set of microservices) can be run with one command:

```
make rundev
```

which will properly configure and run all microservices.

Please assure that MQTT microservice has `node_modules` installed, as explained in _MQTT Microservice_ chapter.

> N.B. `make rundev` actually calls helper script `scripts/run.sh`, so you can inspect this script for the details.

## Events
In order to be easily integratable system, Mainflux is using [Redis Streams](https://redis.io/topics/streams-intro) 
as an event log for event sourcing. Services that are publishing events to Redis Streams 
are `things` service, `bootstrap` service, and `mqtt` adapter.

### Things Service
For every operation that has side effects (that is changing service state) `things` 
service will generate new event and publish it to Redis Stream called `mainflux.things`. 
Every event has its own event ID that is automatically generated and `operation` 
field that can have one of the following values:
- `thing.create` for thing creation,
- `thing.update` for thing update,
- `thing.remove` for thing removal,
- `thing.connect` for connecting a thing to a channel,
- `thing.disconnect` for disconnecting thing from a channel,
- `channel.create` for channel creation,
- `channel.update` for channel update,
- `channel.remove` for channel removal.

By fetching and processing these events you can reconstruct `things` service state. 
If you store some of your custom data in `metadata` field, this is the perfect 
way to fetch it and process it. If you want to integrate through 
[docker-compose.yml](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml)
you can use `mainflux-things-redis` service. Just connect to it and consume events 
from Redis Stream named `mainflux.things`.

#### Thing create event

Whenever thing is created, `things` service will generate new `create` event. This 
event will have the following format:
```
1) "1555334740911-0"
2)  1) "type"
    2) "device"
    3) "operation"
    4) "thing.create"
    5) "name"
    6) "d0"
    7) "id"
    8) "3c36273a-94ea-4802-84d6-a51de140112e"
    9) "owner"
   10) "john.doe@email.com"
   11) "metadata"
   12) "{}"
```

As you can see from this example, every odd field represents field name while every 
even field represents field value. This is standard event format for Redis Streams.
If you want to extract `metadata` field from this event, you'll have to read it as
string first, and then you can deserialize it to some structured format.

#### Thing update event
Whenever thing instance is updated, `things` service will generate new `update` event.
This event will have the following format:
```
1) "1555336161544-0"
2) 1) "operation"
   2) "thing.update"
   3) "name"
   4) "weio"
   5) "id"
   6) "3c36273a-94ea-4802-84d6-a51de140112e"
   7) "type"
   8) "device"
```
Note that thing update event will contain only those fields that were updated using
update endpoint.

#### Thing remove event
Whenever thing instance is removed from the system, `things` service will generate and
publish new `remove` event. This event will have the following format:
```
1) 1) "1555339313003-0"
2) 1) "id"
   2) "3c36273a-94ea-4802-84d6-a51de140112e"
   3) "operation"
   4) "thing.remove"
```

#### Channel create event
Whenever channel instance is created, `things` service will generate and publish new
`create` event. This event will have the following format:
```
1) "1555334740918-0"
2) 1) "id"
   2) "16fb2748-8d3b-4783-b272-bb5f4ad4d661"
   3) "owner"
   4) "john.doe@email.com"
   5) "operation"
   6) "channel.create"
   7) "name"
   8) "c1"
```

#### Channel update event
Whenever channel instance is updated, `things` service will generate and publish new
`update` event. This event will have the following format:
```
1) "1555338870341-0"
2) 1) "name"
   2) "chan"
   3) "id"
   4) "d9d8f31b-f8d4-49c5-b943-6db10d8e2949"
   5) "operation"
   6) "channel.update"
```
Note that update channel event will contain only those fields that were updated using 
update channel endpoint.

#### Channel remove event
Whenever channel instance is removed from the system, `things` service will generate and
publish new `remove` event. This event will have the following format:
```
1) 1) "1555339429661-0"
2) 1) "id"
   2) "d9d8f31b-f8d4-49c5-b943-6db10d8e2949"
   3) "operation"
   4) "channel.remove"
```

#### Connect thing to a channel event
Whenever thing is connected to a channel on `things` service, `things` service will
generate and publish new `connect` event. This event will have the following format:
```
1) "1555334740920-0"
2) 1) "chan_id"
   2) "d9d8f31b-f8d4-49c5-b943-6db10d8e2949"
   3) "thing_id"
   4) "3c36273a-94ea-4802-84d6-a51de140112e"
   5) "operation"
   6) "thing.connect"
```

#### Disconnect thing from a channel event
Whenever thing is disconnected from a channel on `things` service, `things` service
will generate and publish new `disconnect` event. This event will have the following
format:
```
1) "1555334740920-0"
2) 1) "chan_id"
   2) "d9d8f31b-f8d4-49c5-b943-6db10d8e2949"
   3) "thing_id"
   4) "3c36273a-94ea-4802-84d6-a51de140112e"
   5) "operation"
   6) "thing.disconnect"
```

> **Note:** Every one of these events will omit fields that were not used or are not 
relevant for specific operation. Also, field ordering is not guaranteed, so DO NOT 
rely on it.

### Bootstrap Service
Bootstrap service publishes events to Redis Stream called `mainflux.bootstrap`.
Every event from this service contains `operation` field which indicates one of
the following event types:
- `config.create` for configuration creation,
- `config.update` for configuration update,
- `config.remove` for configuration removal,
- `thing.bootstrap` for device bootstrap,
- `thing.state_change` for device state change,
- `thing.update_connections` for device connection update.

If you want to integrate through 
[docker-compose.yml](https://github.com/mainflux/mainflux/blob/master/docker/addons/bootstrap/docker-compose.yml)
you can use `mainflux-bootstrap-redis` service. Just connect to it and consume events 
from Redis Stream named `mainflux.bootstrap`.

#### Configuration create event
Whenever configuration is created, `bootstrap` service will generate and publish
new `create` event. This event will have the following format:
```
1) "1555404899581-0"
2)  1) "owner"
    2) "john.doe@email.com"
    3) "name"
    4) "some"
    5) "channels"
    6) "ff13ca9c-7322-4c28-a25c-4fe5c7b753fc, c3642289-501d-4974-82f2-ecccc71b2d82, c3642289-501d-4974-82f2-ecccc71b2d83, cd4ce940-9173-43e3-86f7-f788e055eb14"
    7) "externalID"
    8) "9c:b6:d:eb:9f:fd"
    9) "content"
   10) "{}"
   11) "timestamp"
   12) "1555404899"
   13) "operation"
   14) "config.create"
   15) "thing_id"
   16) "63a110d4-2b77-48d2-aa46-2582681eeb82"
```

#### Configuration update event
Whenever configuration is updated, `bootstrap` service will generate and publish
new `update` event. This event will have the following format:
```
1) "1555405104368-0"
2)  1) "content"
    2) "NOV_MGT_HOST: http://127.0.0.1:7000\nDOCKER_MGT_HOST: http://127.0.0.1:2375\nAGENT_MGT_HOST: https://127.0.0.1:7003\nMF_MQTT_HOST: tcp://104.248.142.133:8443"              
    3) "timestamp"
    4) "1555405104"
    5) "operation"
    6) "config.update"
    7) "thing_id"
    8) "63a110d4-2b77-48d2-aa46-2582681eeb82"
    9) "name"
   10) "weio"
```

#### Configuration remove event
Whenever configuration is removed, `bootstrap` service will generate and publish
new `remove` event. This event will have the following format:
```
1) "1555405464328-0"
2) 1) "thing_id"
   2) "63a110d4-2b77-48d2-aa46-2582681eeb82"
   3) "timestamp"
   4) "1555405464"
   5) "operation"
   6) "config.remove"
```

#### Thing bootstrap event
Whenever thing is bootstrapped, `bootstrap` service will generate and publish
new `bootstrap` event. This event will have the following format:
```
1) "1555405173785-0"
2) 1) "externalID"
   2) "9c:b6:d:eb:9f:fd"
   3) "success"
   4) "1"
   5) "timestamp"
   6) "1555405173"
   7) "operation"
   8) "thing.bootstrap"
```

#### Thing change state event
Whenever thing's state changes, `bootstrap` service will generate and publish
new `change state` event. This event will have the following format:
```
1) "1555405294806-0"
2) 1) "thing_id"
   2) "63a110d4-2b77-48d2-aa46-2582681eeb82"
   3) "state"
   4) "0"
   5) "timestamp"
   6) "1555405294"
   7) "operation"
   8) "thing.state_change"
```

#### Thing update connections event
Whenever thing's list of connections is updated, `bootstrap` service will generate
and publish new `update connections` event. This event will have the following format:
```
1) "1555405373360-0"
2) 1) "operation"
   2) "thing.update_connections"
   3) "thing_id"
   4) "63a110d4-2b77-48d2-aa46-2582681eeb82"
   5) "channels"
   6) "ff13ca9c-7322-4c28-a25c-4fe5c7b753fc, 925461e6-edfb-4755-9242-8a57199b90a5, c3642289-501d-4974-82f2-ecccc71b2d82"
   7) "timestamp"
   8) "1555405373"
```

### MQTT Adapter
Instead of using heartbeat to know when client is connected through MQTT adapter one
can fetch events from Redis Streams that MQTT adapter publishes. MQTT adapter
publishes events every time client connects and disconnects to stream named `mainflux.mqtt`.

Events that are coming from MQTT adapter have following fields:
- `thing_id` ID of a thing that has connected to MQTT adapter,
- `timestamp` is in Epoch UNIX Time Stamp format,
- `event_type` can have two possible values, connect and disconnect,
- `instance` represents MQTT adapter instance.

If you want to integrate through 
[docker-compose.yml](https://github.com/mainflux/mainflux/blob/master/docker/docker-compose.yml)
you can use `mainflux-mqtt-redis` service. Just connect to it and consume events 
from Redis Stream named `mainflux.mqtt`.

Example of connect event:
```
1) 1) "1555351214144-0"
2) 1) "thing_id"
   2) "1c597a85-b68e-42ff-8ed8-a3a761884bc4"
   3) "timestamp"
   4) "1555351214"
   5) "event_type"
   6) "connect"
   7) "instance"
   8) "mqtt-adapter-1"
      
```

Example of disconnect event:
```
1) 1) "1555351214188-0"
2) 1) "thing_id"
   2) "1c597a85-b68e-42ff-8ed8-a3a761884bc4"
   3) "timestamp"
   4) "1555351214"
   5) "event_type"
   6) "disconnect"
   7) "instance"
   8) "mqtt-adapter-1"
```
