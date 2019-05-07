# Postgres writer

Postgres writer provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                        | Default               |
|-------------------------------------|------------------------------------|-----------------------|
| MF_NATS_URL                         | NATS instance URL                  | nats://localhost:4222 |
| MF_POSTGRES_WRITER_LOG_LEVEL        | Service log level                  | error                 |
| MF_POSTGRES_WRITER_PORT             | Service HTTP port                  | 9104                  |
| MF_POSTGRES_WRITER_DB_HOST          | Postgres DB host                   | postgres              |
| MF_POSTGRES_WRITER_DB_PORT          | Postgres DB port                   | 5432                  |
| MF_POSTGRES_WRITER_DB_USER          | Postgres user                      | mainflux              |
| MF_POSTGRES_WRITER_DB_PASS          | Postgres password                  | mainflux              |
| MF_POSTGRES_WRITER_DB_NAME          | Postgres database name             | messages              |
| MF_POSTGRES_WRITER_DB_SSL_MODE      | Postgres SSL mode                  | disabled              |
| MF_POSTGRES_WRITER_DB_SSL_CERT      | Postgres SSL certificate path      | ""                    |
| MF_POSTGRES_WRITER_DB_SSL_KEY       | Postgres SSL key                   | ""                    |
| MF_POSTGRES_WRITER_DB_SSL_ROOT_CERT | Postgres SSL root certificate path | ""                    |

## Deployment

```yaml
  postgres-writer:
    image: mainflux/postgres-writer:[version]
    container_name: [instance name]
    depends_on:
      - postgres
      - nats
    restart: on-failure
    environment:
      MF_NATS_URL: [NATS instance URL]
      MF_POSTGRES_WRITER_LOG_LEVEL: [Service log level]
      MF_POSTGRES_WRITER_PORT: [Service HTTP port]
      MF_POSTGRES_WRITER_DB_HOST: [Postgres host]
      MF_POSTGRES_WRITER_DB_PORT: [Postgres port]
      MF_POSTGRES_WRITER_DB_USER: [Postgres user]
      MF_POSTGRES_WRITER_DB_PASS: [Postgres password]
      MF_POSTGRES_WRITER_DB_NAME: [Postgres database name]
      MF_POSTGRES_WRITER_DB_SSL_MODE: [Postgres SSL mode]
      MF_POSTGRES_WRITER_DB_SSL_CERT: [Postgres SSL cert]
      MF_POSTGRES_WRITER_DB_SSL_KEY: [Postgres SSL key]
      MF_POSTGRES_WRITER_DB_SSL_ROOT_CERT: [Postgres SSL Root cert]
    ports:
      - 9104:9104
    networks:
      - docker_mainflux-base-net
```

To start the service, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux


cd $GOPATH/src/github.com/mainflux/mainflux

# compile the postgres writer
make postgres-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_NATS_URL=[NATS instance URL] MF_POSTGRES_WRITER_LOG_LEVEL=[Service log level] MF_POSTGRES_WRITER_PORT=[Service HTTP port] MF_POSTGRES_WRITER_DB_HOST=[Postgres host] MF_POSTGRES_WRITER_DB_PORT=[Postgres port] MF_POSTGRES_WRITER_DB_USER=[Postgres user] MF_POSTGRES_WRITER_DB_PASS=[Postgres password] MF_POSTGRES_WRITER_DB_NAME=[Postgres database name] MF_POSTGRES_WRITER_DB_SSL_MODE=[Postgres SSL mode] MF_POSTGRES_WRITER_DB_SSL_CERT=[Postgres SSL cert] MF_POSTGRES_WRITER_DB_SSL_KEY=[Postgres SSL key] MF_POSTGRES_WRITER_DB_SSL_ROOT_CERT=[Postgres SSL Root cert] $GOBIN/mainflux-postgres-writer
```

## Usage

Starting service will start consuming normalized messages in SenML format.
