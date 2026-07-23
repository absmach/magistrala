# BOOTSTRAP SERVICE

New devices need to be configured properly and connected to the Magistrala. Bootstrap service is used in order to accomplish that. This service provides the following features:

1. Creating new Magistrala Clients
2. Providing basic configuration for the newly created Clients
3. Enabling/disabling bootstrap enrollments

Pre-provisioning a new Client is as simple as sending Configuration data to the Bootstrap service. Once the Client is online, it sends a request for initial config to Bootstrap service. Bootstrap service provides an API for enabling and disabling bootstrap enrollments. Bootstrapping does not implicitly enable an enrollment; it has to be done manually.

In order to bootstrap successfully, the Client needs to send bootstrapping request to the specific URL, as well as a secret key. This key and URL are pre-provisioned during the manufacturing process. If the Client is provisioned on the Bootstrap service side, the corresponding configuration will be sent as a response. Otherwise, the Client will be saved so that it can be provisioned later.

## Client Configuration Entity

Client Configuration consists of two logical parts: the custom configuration that can be interpreted by the Client itself and Magistrala-related configuration. Magistrala config contains:

1. corresponding Magistrala Client ID
2. corresponding Magistrala Client key
3. list of the Magistrala channels the Client is connected to

> Note: list of channels contains IDs of the Magistrala channels. These channels are _pre-provisioned_ on the Magistrala side and, unlike corresponding Magistrala Client, Bootstrap service is not able to create Magistrala Channels.

Enabling and disabling a bootstrap enrollment is an enrollment toggle. Configuration keeps a _status_:

| Status   | What it means                                               |
| -------- | ----------------------------------------------------------- |
| disabled | Enrollment exists, but bootstrap is not allowed             |
| enabled  | Enrollment can be used to fetch bootstrap configuration     |

Switching between statuses `enabled` and `disabled` enables and disables the enrollment, respectively.

Client configuration also contains the so-called `external ID` and `external key`. An external ID is a unique identifier of corresponding Client. For example, a device MAC address is a good choice for external ID. External key is a secret key that is used for authentication during the bootstrapping procedure.

## Configuration

The service is configured using the environment variables presented in the following table. Note that any unset variables will be replaced with their default values.

Bootstrap uses two independent encryption layers:

- `MG_BOOTSTRAP_DB_ENCRYPTION_KEY` is a service-owned 32-byte master key. HKDF-derived AES-256-GCM keys encrypt enrollment external keys, domain transport keys and secret binding snapshots before PostgreSQL storage.
- Each domain has a random 256-bit transport key managed through `/{domainID}/clients/bootstrap/transport-keys`. That key encrypts only secure device requests and responses and is itself encrypted by the database master key at rest. Creating or rotating a key returns its base64url secret; authorized domain managers can reveal it later.
- An authorized manager can generate a five-minute, single-use device credential with `POST /{domainID}/clients/configs/{configID}/secure-credential`. The returned `Authorization: Client v2...` value is generated on demand and is never stored.

Changing a domain transport key does not change the database master key. Rotation keeps the old domain key valid for 24 hours while devices move to the new key.

| Variable                       | Description                                                                      | Default                           |
| ------------------------------ | -------------------------------------------------------------------------------- | --------------------------------- |
| MG_BOOTSTRAP_LOG_LEVEL        | Log level for Bootstrap (debug, info, warn, error)                               | info                              |
| MG_BOOTSTRAP_DB_HOST          | Database host address                                                            | localhost                         |
| MG_BOOTSTRAP_DB_PORT          | Database host port                                                               | 5432                              |
| MG_BOOTSTRAP_DB_USER          | Database user                                                                    | magistrala                        |
| MG_BOOTSTRAP_DB_PASS          | Database password                                                                | magistrala                        |
| MG_BOOTSTRAP_DB_NAME          | Name of the database used by the service                                         | bootstrap                         |
| MG_BOOTSTRAP_DB_SSL_MODE      | Database connection SSL mode (disable, require, verify-ca, verify-full)          | disable                           |
| MG_BOOTSTRAP_DB_SSL_CERT      | Path to the PEM encoded certificate file                                         | ""                                |
| MG_BOOTSTRAP_DB_SSL_KEY       | Path to the PEM encoded key file                                                 | ""                                |
| MG_BOOTSTRAP_DB_SSL_ROOT_CERT | Path to the PEM encoded root certificate file                                    | ""                                |
| MG_BOOTSTRAP_DB_ENCRYPTION_KEY | 32-byte master key used only to encrypt Bootstrap secrets stored in PostgreSQL  | 12345678910111213141516171819202  |
| MG_BOOTSTRAP_DB_ENCRYPTION_KEY_ID | Identifier written into database encryption envelopes                       | primary                           |
| MG_BOOTSTRAP_HTTP_HOST        | Bootstrap service HTTP host                                                      | ""                                |
| MG_BOOTSTRAP_HTTP_PORT        | Bootstrap service HTTP port                                                      | 9013                              |
| MG_BOOTSTRAP_HTTP_SERVER_CERT | Path to server certificate in pem format                                         | ""                                |
| MG_BOOTSTRAP_HTTP_SERVER_KEY  | Path to server key in pem format                                                 | ""                                |
| MG_BOOTSTRAP_EVENT_CONSUMER   | Bootstrap service event source consumer name                                     | bootstrap                         |
| MG_ES_URL                     | Event store URL                                                                  | <nats://localhost:4222>           |
| ATOM_URL                      | ATOM HTTP endpoint used for identity, authorization and resource lookup          | required                          |
| ATOM_SERVICE_TOKEN            | Service token used for ATOM resource lookup and projection                       | required                          |
| ATOM_JWKS_URL                 | ATOM JWKS endpoint used to verify bearer tokens                                  | `${ATOM_URL}/.well-known/jwks.json` |
| ATOM_JWT_ISSUER               | Expected ATOM access-token issuer                                                | `ATOM_URL`                        |
| ATOM_JWT_AUDIENCE             | Expected ATOM access-token audience                                              | magistrala                        |
| ATOM_TIMEOUT                  | Timeout for ATOM requests                                                        | 5s                                |
| MG_JAEGER_URL                 | Jaeger server URL                                                                | <http://localhost:4318/v1/traces> |
| MG_JAEGER_TRACE_RATIO         | Jaeger sampling ratio                                                            | 1.0                               |
| MG_SEND_TELEMETRY             | Send telemetry to magistrala call home server                                    | true                              |
| MG_BOOTSTRAP_INSTANCE_ID      | Bootstrap service instance ID                                                    | ""                                |

## Deployment

The service itself is distributed as Docker container. Check the [`bootstrap`](https://github.com/absmach/magistrala/blob/main/docker/addons/bootstrap/docker-compose.yaml) service section in docker-compose file to see how service is deployed.

To start the service outside of the container, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/absmach/magistrala

cd magistrala

# compile the servic e
make bootstrap

# copy binary to bin
make install

# set the environment variables and run the service
MG_BOOTSTRAP_LOG_LEVEL=info \
MG_BOOTSTRAP_DB_HOST=localhost \
MG_BOOTSTRAP_DB_PORT=5432 \
MG_BOOTSTRAP_DB_USER=magistrala \
MG_BOOTSTRAP_DB_PASS=magistrala \
MG_BOOTSTRAP_DB_NAME=bootstrap \
MG_BOOTSTRAP_DB_SSL_MODE=disable \
MG_BOOTSTRAP_DB_SSL_CERT="" \
MG_BOOTSTRAP_DB_SSL_KEY="" \
MG_BOOTSTRAP_DB_SSL_ROOT_CERT="" \
MG_BOOTSTRAP_HTTP_HOST=localhost \
MG_BOOTSTRAP_HTTP_PORT=9010 \
MG_BOOTSTRAP_HTTP_SERVER_CERT="" \
MG_BOOTSTRAP_HTTP_SERVER_KEY="" \
MG_BOOTSTRAP_EVENT_CONSUMER=bootstrap \
MG_ES_URL=nats://localhost:4222 \
ATOM_URL=http://localhost:8080 \
ATOM_SERVICE_TOKEN=<bootstrap-service-token> \
ATOM_JWKS_URL=http://localhost:8080/.well-known/jwks.json \
ATOM_JWT_ISSUER=http://localhost:8080 \
ATOM_JWT_AUDIENCE=magistrala \
ATOM_TIMEOUT=5s \
MG_JAEGER_URL=http://localhost:14268/api/traces \
MG_JAEGER_TRACE_RATIO=1.0 \
MG_SEND_TELEMETRY=true \
MG_BOOTSTRAP_INSTANCE_ID="" \
$GOBIN/magistrala-bootstrap
```

Setting `MG_BOOTSTRAP_HTTP_SERVER_CERT` and `MG_BOOTSTRAP_HTTP_SERVER_KEY` will enable TLS against the service. The service expects a file in PEM format for both the certificate and the key.

Bootstrap PostgreSQL is authoritative for enrollments, profiles, templates, certificates, encrypted external keys and encrypted binding snapshots. ATOM is authoritative for identity and authorization and receives only non-secret bootstrap resource projections.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.absmach.eu/?urls.primaryName=bootstrap.yaml).
