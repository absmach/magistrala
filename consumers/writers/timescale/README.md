# Timescale writer

Timescale writer provides message repository implementation for Timescale.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                             | Description                                               | Default                |
| -----------------------------------  | --------------------------------------------------------- | ---------------------- |
| MF_BROKER_URL                        | Message broker instance URL                               | nats://localhost:4222  |
| MF_TIMESCALE_WRITER_LOG_LEVEL        | Service log level                                         | info                   |
| MF_TIMESCALE_WRITER_PORT             | Service HTTP port                                         | 8180                   |
| MF_TIMESCALE_WRITER_DB_HOST          | Timescale DB host                                         | timescale              |
| MF_TIMESCALE_WRITER_DB_PORT          | Timescale DB port                                         | 5432                   |
| MF_TIMESCALE_WRITER_DB_USER          | Timescale user                                            | mainflux               |
| MF_TIMESCALE_WRITER_DB_PASS          | Timescale password                                        | mainflux               |
| MF_TIMESCALE_WRITER_DB               | Timescale database name                                   | messages               |
| MF_TIMESCALE_WRITER_DB_SSL_MODE      | Timescale SSL mode                                        | disabled               |
| MF_TIMESCALE_WRITER_DB_SSL_CERT      | Timescale SSL certificate path                            | ""                     |
| MF_TIMESCALE_WRITER_DB_SSL_KEY       | Timescale SSL key                                         | ""                     |
| MF_TIMESCALE_WRITER_DB_SSL_ROOT_CERT | Timescale SSL root certificate path                       | ""                     |
| MF_TIMESCALE_WRITER_CONFIG_PATH      | Configuration file path with Message broker subjects list | /config.toml           |

## Deployment

The service itself is distributed as Docker container. Check the [`timescale-writer`](https://github.com/mainflux/mainflux/blob/master/docker/addons/timescale-writer/docker-compose.yml#L34-L59) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the timescale writer
make timescale-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_BROKER_URL=[Message broker instance URL] \
MF_TIMESCALE_WRITER_LOG_LEVEL=[Service log level] \
MF_TIMESCALE_WRITER_PORT=[Service HTTP port] \
MF_TIMESCALE_WRITER_DB_HOST=[Timescale host] \
MF_TIMESCALE_WRITER_DB_PORT=[Timescale port] \
MF_TIMESCALE_WRITER_DB_USER=[Timescale user] \
MF_TIMESCALE_WRITER_DB_PASS=[Timescale password] \
MF_TIMESCALE_WRITER_DB=[Timescale database name] \
MF_TIMESCALE_WRITER_DB_SSL_MODE=[Timescale SSL mode] \
MF_TIMESCALE_WRITER_DB_SSL_CERT=[Timescale SSL cert] \
MF_TIMESCALE_WRITER_DB_SSL_KEY=[Timescale SSL key] \
MF_TIMESCALE_WRITER_DB_SSL_ROOT_CERT=[Timescale SSL Root cert] \
MF_TIMESCALE_WRITER_CONFIG_PATH=[Configuration file path with Message broker subjects list] \
MF_TIMESCALE_WRITER_TRANSFORMER=[Message transformer type] \
$GOBIN/mainflux-timescale-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
