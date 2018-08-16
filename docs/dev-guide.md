## Getting Mainflux

Mainflux can be fetched from official [Mainflux GitHub repository](https://github.com/Mainflux/mainflux):

```
mkdir -p $GOPATH/src/github.com/mainflux
cd $GOPATH/src/github.com/mainflux
git clone https://github.com/mainflux/mainflux.git
cd mainflux
```

## Building

### Prerequisites
Make sure that you have [Protocol Buffers](https://developers.google.com/protocol-buffers/) compiler (`protoc`) installed.

[Go Protobuf](https://github.com/golang/protobuf) installation instructions are [here](https://github.com/golang/protobuf#installation).
Go Protobuf uses C bindings, so you will need to install [C++ protobuf](https://github.com/google/protobuf) as a prerequisite.

### Build All Services

Use `GNU Make` tool to build all mainflux services:

```
make
```

Build artefacts will be put in the `build` directory.

> N.B. All Mainflux services are built as a statically linked binaries. This way they can be portable (transfered to any platform just by placing them there and running them) as they contain all needed libraries and do not relay on system shared libs. This helps creating [FROM scratch](https://hub.docker.com/_/scratch/) dockers.

### Build Individual Microservice
Individual microservices can be built with command:

```
make <microservice_name>
```

For example:

```
make http
```

will build HTTP Adapter microservice.

### Building Dockers

Dockers can be built with:

```
make dockers
```

or individually with

```
make docker_<microservice_name>
```

For example:

```
make docker_http
```

> N.B. Mainflux creates `FROM scratch` docker containers which as compact and small in size.

### MQTT Microservice
MQTT Microservice in Mainflux is special, as it is currently the only microservice written in NodeJS. It is not compiled,
but node modules need to be downloaded in order to start the service:

```
cd mqtt
npm install
```

After that MQTT Adapter can be started from top directory (as it needs to find `*.proto` files) with:
```
node mqtt/mqtt.js
```

### Protobuf
Aforementioned  `make` (which is an alias for `make all` target) is calling `protoc` command prior to compiling individual microservices.

To do this by hand, execute:

```
protoc --go_out=plugins=grpc:. *.proto
```

A shorthand to do this via `make` tool is:

```
make proto
```

> N.B. This must be done one time in the beginning in order to generate protobuf Go structures needed for the build.

### Cross-compiling for ARM
Mainflux can be compiled for ARM platform and run on Raspberry Pi or other similar IoT gateways.

Following the instructions [here](https://dave.cheney.net/2015/08/22/cross-compilation-with-go-1-5) or [here](https://www.alexruf.net/golang/arm/raspberrypi/2016/01/16/cross-compile-with-go-1-5-for-raspberry-pi.html) as well as information
found [here](https://github.com/golang/go/wiki/GoArm), environment variables `GOARCH=arm` and `GOARM=7` must be set for the compilation.

Cross-compilation for ARM with Mainflux make:

```
GOOS=linux GOARCH=arm GOARM=7 make
```

## Installing
Installing Go binaries is simple: just move them to `$GOBIN` (do not fortget to add `$GOBIN` to your `$PATH`).

## Deployment

### Prerequisites
Mainflux depends on several infrastructureal services, notably [NATS](https://www.nats.io/) broker and [PostgreSQL](https://www.postgresql.org/) database.

#### NATS
Mainflux uses NATS as it's central message bus. For development purposes (when not run via Docker), it expects that NATS is installed on the local system.

To do this execute:

```
go get github.com/nats-io/go-nats
```

This will install `gnatsd` binary that can be simply run by invoking

```
gnatsd
```

#### PostgreSQL
Mainflux uses PostgreSQL to store metadata (`users`, `things` and `channels` entities alongside with authorization tokens).
It expects that PostgreSQL DB is installed, setu-up and running on the local system.

Inflormation how to set-up (prepare) PostgreSQL database can be found [here](https://support.rackspace.com/how-to/postgresql-creating-and-dropping-roles/),
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
Running of the Mainflux microservices can be tricky, as there is a lot of them and each demand config in the form of environment variables.

Whole system (set of microservices) can be run with one command:

```
make run
```

which will properly configure and run all microservices.

Please assure that MQTT microservice has `node_modules` installed, as explained in _MQTT Microservice_ chapter.

> N.B. `make run` actually calls helper script `scripts/run.sh`, so you can inspect this script for the details.
