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

> N.B. The `things-db` and `users-db` containers are built from a vanilla PostgreSQL docker image downloaded from docker hub which does not persist the data when these containers are rebuilt. Thus, __rebuilding of all docker containers with `make dockers` or rebuilding the `things-db` and `users-db` containers separately with `make docker_things-db` and `make docker_users-db` respectively, will cause data loss. All your users, things, channels and connections between them will be lost!__ As we use this setup only for development, we don't guarantee any permanent data persistence. If you need to retain the data between the container rebuilds you can attach volume to the `things-db` and `users-db` containers. Check the official docs on how to use volumes [here](https://docs.docker.com/storage/volumes/) and [here](https://docs.docker.com/compose/compose-file/#volumes).

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
