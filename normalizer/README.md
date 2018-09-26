# Message normalizer

Normalizer service consumes events published by adapters, normalizes SenML-formatted
ones, and publishes them to the post-processing stream.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                  | Description                  | Default               |
|---------------------------|------------------------------|-----------------------|
| MF_NATS_URL               | NATS instance URL            | nats://localhost:4222 |
| MF_NORMALIZER_LOG_LEVEL   | Log level for the Normalizer | error                 |
| MF_NORMALIZER_PORT        | Normalizer service HTTP port | 8180                  |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
services:
  normalizer:
    image: mainflux/normalizer:[version]
    container_name: [instance name]
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_NORMALIZER_LOG_LEVEL: [Normalizer log level]
      MF_NORMALIZER_PORT: [Service HTTP port]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the app; make sure to set the proper GOOS value
make normalizer

# copy binary to bin
make install

# set the environment variables and run the service
MF_NATS_URL=[NATS instance URL] MF_NORMALIZER_LOG_LEVEL=[Normalizer log level] MF_NORMALIZER_PORT=[Service HTTP port] $GOBIN/mainflux-normalizer
```
