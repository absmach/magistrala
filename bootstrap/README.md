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

- `MG_BOOTSTRAP_DB_ENCRYPTION_KEY` is a service-owned 32-byte master key. HKDF-derived AES-256-GCM keys encrypt enrollment external keys and secret binding snapshots before PostgreSQL storage.
- Every enrollment has a Bootstrap root key containing at least 10 characters. Its exact UTF-8 bytes are used as HKDF input. The service generates a strong random base64url string when `external_key` is omitted during enrollment creation. The device stores its Bootstrap URL, external ID, root key, and key version separately from the configuration it downloads.
- The device never sends that root key. It requests a one-minute server challenge, returns an HMAC-SHA256 proof made with an HKDF-derived authentication key, and receives an AES-256-GCM encrypted configuration made with a different HKDF-derived response key. Each challenge is single-use, and the protocol does not require a device clock.

Replacing an enrollment's external key increments its key version and invalidates outstanding challenges. It does not change the database master key or any other device's Bootstrap key.

## Device Bootstrap flow

1. Request a challenge:
   `POST /devices/bootstrap/challenges/{externalID}`.
2. Generate a random 32-byte device nonce.
3. Encode the external key as UTF-8 and use those exact bytes as the root key.
   Derive a 32-byte authentication key with HKDF-SHA256 using the external ID
   as salt and `magistrala-bootstrap-auth-v1` as info.
4. HMAC the newline-separated values `v1`, external ID, challenge ID, server
   nonce, device nonce, and decimal key version.
5. Send the unpadded-base64url device nonce and proof to
   `POST /devices/bootstrap/configurations/{externalID}`.
6. Derive the response key with the same root key and salt but
   `magistrala-bootstrap-response-v1` as info, then AES-GCM decrypt the returned
   JSON envelope as described in the OpenAPI document.
7. Read `content_type` from the decrypted response and parse the `content`
   string accordingly (`application/json`, `application/yaml`,
   `application/toml`, or `text/plain`).

This application-layer encryption protects device authentication and Bootstrap
content even when a constrained deployment uses plain HTTP. Plain HTTP still
reveals network metadata and permits traffic blocking, replay attempts, and
denial of service, so HTTPS should be used whenever the device supports it.

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

## Legacy bcrypt enrollment migration

This release does not retain the pre-HKDF Bootstrap request. A bcrypt hash
cannot provide the recoverable root key needed for the challenge/proof protocol,
so every bcrypt-backed enrollment must be re-enrolled before the release.

Re-enroll each legacy device as follows:

1. Inventory bcrypt-backed Bootstrap rows and identify their device owners.
2. Generate and deliver a new Bootstrap root key through the device's approved
   out-of-band provisioning channel; never recover or log the old key.
3. Create a replacement enrollment with a unique replacement external ID and
   that new key; this stores an encrypted key envelope with
   `bootstrap_key_version` set to one.
4. Update the device to use its replacement external ID with
   `POST /devices/bootstrap/challenges/{externalID}`
   followed by `POST /devices/bootstrap/configurations/{externalID}`, then
   verify one successful challenge/proof bootstrap.
5. Disable and remove the bcrypt-backed enrollment only after the replacement
   has successfully bootstrapped. Keep a rollback record of the original
   enrollment metadata, never its key, until the migration window closes.

## Usage

For more information about service capabilities and its usage, please check out the [API documentation](https://docs.api.magistrala.absmach.eu/?urls.primaryName=bootstrap.yaml).
