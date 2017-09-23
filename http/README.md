# Mainflux HTTP adapter

Mainflux HTTP adapter provides an HTTP API for sending messages through the
platform.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable         | Description       | Default               |
|------------------|-------------------|-----------------------|
| ADAPTER_NATS_URL | NATS instance URL | nats://localhost:4222 |

## Deployment

The service is distributed as Docker container. The following snippet provides
a compose file template that can be used to deploy the service container locally:

```yaml
version: "2"
services:
  adapter:
    image: mainflux/http-adapter:[version]
    container_name: [instance name]
    ports:
      - [host machine port]:8180
    environment:
      ADAPTER_NATS_URL: [NATS instance URL]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/http-adapter

cd $GOPATH/github.com/mainflux/http-adapter/cmd

# compile the app; make sure to set the proper GOOS value
CGO_ENABLED=0 GOOS=[platform identifier] go build -ldflags "-s" -a -installsuffix cgo -o app

# set the environment variables and run the service
ADAPTER_NATS_URL=[NATS instance URL] app
```

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yaml).
