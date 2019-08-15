# BOOTSTRAP SERVICE

New devices need to be configured properly and connected to the Mainflux. Bootstrap service is used in order to accomplish that. This service provides the following features:

  1) Creating new Mainflux Things
  2) Providing basic configuration for the newly created Things
  3) Enabling/disabling Things

Pre-provisioning a new Thing is as simple as sending Configuration data to the Bootstrap service. Once the Thing is online, it sends a request for initial config to Bootstrap service. Bootstrap service provides an API for enabling and disabling Things. Only enabled Things can exchange messages over Mainflux. Bootstrapping does not implicitly enable Things, it has to be done manually.

In order to bootstrap successfully, the Thing needs to send bootstrapping request to the specific URL, as well as a secret key. This key and URL are pre-provisioned during the manufacturing process. If the Thing is provisioned on the Bootstrap service side, the corresponding configuration will be sent as a response. Otherwise, the Thing will be saved so that it can be provisioned later.

## Thing Configuration Entity

Thing Configuration consists of two logical parts: the custom configuration that can be interpreted by the Thing itself and Mainflux-related configuration. Mainflux config contains:

  1) corresponding Mainflux Thing ID
  2) corresponding Mainflux Thing key
  3) list of the Mainflux channels the Thing is connected to

>Note: list of channels contains IDs of the Mainflux channels. These channels are _pre-provisioned_ on the Mainflux side and, unlike corresponding Mainflux Thing, Bootstrap service is not able to create Mainflux Channels.

Enabling and disabling Thing (adding Thing to/from whitelist) is as simple as connecting corresponding Mainflux Thing to the given list of Channels. Configuration keeps _state_ of the Thing:

| State    | What it means                                          |
|----------|--------------------------------------------------------|
| Inactive | Thing is created, but isn't enabled                    |
| Active   | Thing is able to communicate using Mainflux            |

Switching between states `Active` and `Inactive` enables and disables Thing, respectively.

Thing configuration also contains the so-called `external ID` and `external key`. An external ID is a unique identifier of corresponding Thing. For example, a device MAC address is a good choice for external ID. External key is a secret key that is used for authentication during the bootstrapping procedure.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

| Variable                      | Description                                                             | Default                          |
|-------------------------------|-------------------------------------------------------------------------|-----------------------           |
| MF_BOOTSTRAP_LOG_LEVEL        | Log level for Bootstrap (debug, info, warn, error)                      | error                            |
| MF_BOOTSTRAP_DB_HOST          | Database host address                                                   | localhost                        |
| MF_BOOTSTRAP_DB_PORT          | Database host port                                                      | 5432                             |
| MF_BOOTSTRAP_DB_USER          | Database user                                                           | mainflux                         |
| MF_BOOTSTRAP_DB_PASS          | Database password                                                       | mainflux                         |
| MF_BOOTSTRAP_DB               | Name of the database used by the service                                | bootstrap                        |
| MF_BOOTSTRAP_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full) | disable                          |
| MF_BOOTSTRAP_DB_SSL_CERT      | Path to the PEM encoded certificate file                                |                                  |
| MF_BOOTSTRAP_DB_SSL_KEY       | Path to the PEM encoded key file                                        |                                  |
| MF_BOOTSTRAP_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                           |                                  |
| MF_BOOTSTRAP_ENCRYPT_KEY      | Secret key for secure bootstrapping encryption                          | 12345678910111213141516171819202 |
| MF_BOOTSTRAP_CLIENT_TLS       | Flag that indicates if TLS should be turned on                          | false                            |
| MF_BOOTSTRAP_CA_CERTS         | Path to trusted CAs in PEM format                                       |                                  |
| MF_BOOTSTRAP_PORT             | Bootstrap service HTTP port                                             | 8180                             |
| MF_BOOTSTRAP_SERVER_CERT      | Path to server certificate in pem format                                |                                  |
| MF_BOOTSTRAP_SERVER_KEY       | Path to server key in pem format                                        |                                  |
| MF_SDK_BASE_URL               | Base url for Mainflux SDK                                               | http://localhost                 |
| MF_SDK_THINGS_PREFIX          | SDK prefix for Things service                                           |                                  |
| MF_USERS_URL                  | Users service URL                                                       | localhost:8181                   |
| MF_THINGS_ES_URL              | Things service event source URL                                         | localhost:6379                   |
| MF_THINGS_ES_PASS             | Things service event source password                                    |                                  |
| MF_THINGS_ES_DB               | Things service event source database                                    | 0                                |
| MF_BOOTSTRAP_ES_URL           | Bootstrap service event source URL                                      | localhost:6379                   |
| MF_BOOTSTRAP_ES_PASS          | Bootstrap service event source password                                 |                                  |
| MF_BOOTSTRAP_ES_DB            | Bootstrap service event source database                                 | 0                                |
| MF_BOOTSTRAP_INSTANCE_NAME    | Bootstrap service instance name                                         | bootstrap                        |
| MF_JAEGER_URL                 | Jaeger server URL                                                       | localhost:6831                   |
| MF_BOOTSTRAP_THINGS_TIMEOUT   | Things gRPC request timeout in seconds                                  | 1                                |

## Deployment

The service itself is distributed as Docker container. The following snippet
provides a compose file template that can be used to deploy the service container
locally:

```yaml
version: "2"
  bootstrap:
    image: mainflux/bootstrap:latest
    container_name: mainflux-bootstrap
    depends_on:
      - bootstrap-db
    restart: on-failure
    ports:
      - 8200:8200
    environment:
      MF_BOOTSTRAP_LOG_LEVEL: [Bootstrap log level]
      MF_BOOTSTRAP_DB_HOST: [Database host address]
      MF_BOOTSTRAP_DB_PORT: [Database host port]
      MF_BOOTSTRAP_DB_USER: [Database user]
      MF_BOOTSTRAP_DB_PASS: [Database password]
      MF_BOOTSTRAP_DB: [Name of the database used by the service]
      MF_BOOTSTRAP_DB_SSL_MODE: [SSL mode to connect to the database with]
      MF_BOOTSTRAP_DB_SSL_CERT: [Path to the PEM encoded certificate file]
      MF_BOOTSTRAP_DB_SSL_KEY: [Path to the PEM encoded key file]
      MF_BOOTSTRAP_DB_SSL_ROOT_CERT: [Path to the PEM encoded root certificate file]
      MF_BOOTSTRAP_ENCRYPT_KEY: [Hex-encoded encryption key used for secure bootstrap]
      MF_BOOTSTRAP_CLIENT_TLS: [Boolean value to enable/disable client TLS]
      MF_BOOTSTRAP_CA_CERTS: [Path to trusted CAs in PEM format]
      MF_BOOTSTRAP_PORT: 8200
      MF_BOOTSTRAP_SERVER_CERT: [String path to server cert in pem format]
      MF_BOOTSTRAP_SERVER_KEY: [String path to server key in pem format]
      MF_SDK_BASE_URL: [Base SDK URL for the Mainflux services]
      MF_SDK_THINGS_PREFIX: [SDK prefix for Things service]
      MF_USERS_URL: [Users service URL]
      MF_THINGS_ES_URL: [Things service event source URL]
      MF_THINGS_ES_PASS: [Things service event source password]
      MF_THINGS_ES_DB: [Things service event source database]
      MF_BOOTSTRAP_ES_URL: [Bootstrap service event source URL]
      MF_BOOTSTRAP_ES_PASS: [Bootstrap service event source password]
      MF_BOOTSTRAP_ES_DB: [Bootstrap service event source database]
      MF_BOOTSTRAP_INSTANCE_NAME: [Bootstrap service instance name]
      MF_JAEGER_URL: [Jaeger server URL]
      MF_BOOTSTRAP_THINGS_TIMEOUT: [Things gRPC request timeout in seconds]
```

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
go get github.com/mainflux/mainflux

cd $GOPATH/src/github.com/mainflux/mainflux

# compile the service
make bootstrap

# copy binary to bin
make install

# set the environment variables and run the service
MF_BOOTSTRAP_LOG_LEVEL=[Bootstrap log level] MF_BOOTSTRAP_DB_HOST=[Database host address] MF_BOOTSTRAP_DB_PORT=[Database host port] MF_BOOTSTRAP_DB_USER=[Database user] MF_BOOTSTRAP_DB_PASS=[Database password] MF_BOOTSTRAP_DB=[Name of the database used by the service] MF_BOOTSTRAP_DB_SSL_MODE=[SSL mode to connect to the database with] MF_BOOTSTRAP_DB_SSL_CERT=[Path to the PEM encoded certificate file] MF_BOOTSTRAP_DB_SSL_KEY=[Path to the PEM encoded key file] MF_BOOTSTRAP_DB_SSL_ROOT_CERT=[Path to the PEM encoded root certificate file] MF_BOOTSTRAP_ENCRYPT_KEY=[Hex-encoded encryption key used for secure bootstrap] MF_BOOTSTRAP_CLIENT_TLS=[Boolean value to enable/disable client TLS] MF_BOOTSTRAP_CA_CERTS=[Path to trusted CAs in PEM format] MF_BOOTSTRAP_PORT=[Service HTTP port] MF_BOOTSTRAP_SERVER_CERT=[Path to server certificate] MF_BOOTSTRAP_SERVER_KEY=[Path to server key] MF_SDK_BASE_URL=[Base SDK URL for the Mainflux services] MF_SDK_THINGS_PREFIX=[SDK prefix for Things service] MF_USERS_URL=[Users service URL] MF_JAEGER_URL=[Jaeger server URL] MF_BOOTSTRAP_THINGS_TIMEOUT=[Things gRPC request timeout in seconds] $GOBIN/mainflux-bootstrap
```

Setting `MF_BOOTSTRAP_CA_CERTS` expects a file in PEM format of trusted CAs. This will enable TLS against the Users gRPC endpoint trusting only those CAs that are provided.

## Usage

For more information about service capabilities and its usage, please check out
the [API documentation](swagger.yml).

[doc]: http://mainflux.readthedocs.io
