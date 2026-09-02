# Docker Composition

Configure environment variables and run Magistrala Docker Composition.

> \*Note\*\*: `docker-compose` uses `.env` file to set all environment variables. Ensure that you run the command from the same location as .env file.

## Installation

Follow the [official Docker Compose installation guide](https://docs.docker.com/compose/install/) to install Docker Compose.

## Usage

Run the following commands from the project root directory.

```bash
make run_latest
```

`run_latest` depends on `docker/.env.tokens`, so on first run it invokes
`scripts/generate-atom-secrets.sh` to mint one unscoped Atom access token per
Magistrala service. The script writes two files from the same random material:

- `docker/atom-bootstrap.yaml` — mounted into the Atom container. Atom hashes
  each token secret at first boot and stamps the credential row
  `managed_by='config'`, so the API can never leak or revoke them.
- `docker/.env.tokens` — loaded by docker compose so downstream services see
  their `ATOM_SERVICE_TOKEN` values.

Both files are gitignored. Rotate all service tokens with:

```bash
make atom-secrets-rotate
```

which regenerates the pair, restarts the Atom container, and prints a
reminder to restart downstream services so they pick up the new tokens.

The Atom runtime image is selected with `ATOM_IMAGE` in `docker/.env`. To test Magistrala against a local Atom checkout, build that checkout with a local tag and set `ATOM_IMAGE` to that tag before running Compose.

If you use `docker compose` directly instead of the Makefile, pass both env files:

```bash
scripts/generate-atom-secrets.sh   # first time only
docker compose -f docker/docker-compose.yaml \
  --env-file docker/.env --env-file docker/.env.tokens up
```

To start additional addon services:

```bash
docker compose -f docker/addons/<path>/docker-compose.yaml --env-file docker/.env up
```

To pull images from a specific release in `ghcr.io/absmach/magistrala`, change `MG_RELEASE_TAG` in `.env` before running these commands.

## Broker Configuration

Magistrala uses a single broker for both message exchange and the events store,
and it is FluxMQ by default. The broker is selected at build time through
`MG_MESSAGE_BROKER_TYPE`, the events store through `MG_ES_TYPE`, and both are
compiled in as Go build tags by the Makefile.

| Component      | Variable                 | Values                              | Default      |
| :------------- | :----------------------- | :---------------------------------- | :----------- |
| Message broker | `MG_MESSAGE_BROKER_TYPE` | `msg_fluxmq`, `msg_nats`            | `msg_fluxmq` |
| Events store   | `MG_ES_TYPE`             | `es_fluxmq`, `es_nats`, `es_redis`  | `es_fluxmq`  |

`es_redis` selects the Redis events store, which needs a message broker deployed
alongside it for message exchange.

Changing either one requires rebuilding the images, and the matching URL in
`docker/.env`:

```bash
MG_MESSAGE_BROKER_TYPE=msg_nats MG_ES_TYPE=es_nats make dockers
```

```env
MG_MESSAGE_BROKER_TYPE=msg_nats
MG_MESSAGE_BROKER_URL=nats://nats:4222
MG_ES_TYPE=es_nats
MG_ES_URL=nats://nats:4222
```

For Redis as the events store, point `MG_ES_URL` at Redis and keep a message
broker for message exchange:

```bash
MG_ES_TYPE=es_redis make dockers
```

```env
MG_ES_TYPE=es_redis
MG_ES_URL=${MG_REDIS_URL}
```

## Nginx Configuration

Nginx is the entry point for all traffic to Magistrala.
By using environment variables file at `docker/.env` you can modify the below given Nginx directive.

| Environment Variable           | Description                                                                                                                                                                                                                                                                      |
| ------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MG_PUBLIC_HOST`               | Public DNS name for the Docker host. This value is used by UI URLs and Let's Encrypt certificate requests.                                                                                                                                                                       |
| `MG_UI_HOST`                   | Internal Compose hostname for the UI service. Defaults to `ui`.                                                                                                                                                                                                                  |
| `MG_LETSENCRYPT_ENABLED`       | Set to `true` to request and use a Let's Encrypt certificate. Set to `false` to comment out the Let's Encrypt cert/key paths and use the fallback Nginx certificate.                                                                                                             |
| `MG_LETSENCRYPT_EMAIL`         | Email address used by Let's Encrypt for expiry and account notifications. Required when running the `letsencrypt` profile.                                                                                                                                                       |
| `MG_LETSENCRYPT_STAGING`       | Set to `true` to request staging certificates while testing. Set to `false` for trusted production certificates.                                                                                                                                                                 |
| `MG_LETSENCRYPT_FORCE_RENEWAL` | Set to `true` for one certbot run when replacing a staging certificate with a production certificate. Set it back to `false` after the production certificate is issued.                                                                                                         |
| `MG_NGINX_SERVER_NAME`         | `MG_NGINX_SERVER_NAME` environmental variable is used to configure nginx directive `server_name`. If environmental variable `MG_NGINX_SERVER_NAME` is empty then default value `localhost` will set to `server_name`.                                                            |
| `MG_NGINX_SERVER_CERT`         | `MG_NGINX_SERVER_CERT` environmental variable is used to configure nginx directive `ssl_certificate`. If environmental variable `MG_NGINX_SERVER_CERT` is empty then by default server certificate in the path `docker/ssl/certs/magistrala-server.crt`  will be assigned.       |
| `MG_NGINX_SERVER_KEY`          | `MG_NGINX_SERVER_KEY` environmental variable is used to configure nginx directive `ssl_certificate_key`. If environmental variable `MG_NGINX_SERVER_KEY` is empty then by default server certificate key in the path `docker/ssl/certs/magistrala-server.key`  will be assigned. |
| `MG_NGINX_SERVER_CLIENT_CA`    | `MG_NGINX_SERVER_CLIENT_CA` environmental variable is used to configure nginx directive `ssl_client_certificate`. If environmental variable `MG_NGINX_SERVER_CLIENT_CA` is empty then by default certificate in the path `docker/ssl/certs/ca.crt` will be assigned.             |
| `MG_NGINX_SERVER_DHPARAM`      | `MG_NGINX_SERVER_DHPARAM` environmental variable is used to configure nginx directive `ssl_dhparam`. If environmental variable `MG_NGINX_SERVER_DHPARAM` is empty then by default file in the path `docker/ssl/dhparam.pem` will be assigned.                                    |

Adjust these values in `.env` to configure TLS / SSL behavior for your deployment.

### HTTPS UI with Let's Encrypt

The Compose stack can request and renew a Let's Encrypt certificate with the optional `letsencrypt` profile. This secures the public Nginx entrypoint and serves the UI through `https://${MG_PUBLIC_HOST}/`. Plain UI requests to `/` are redirected to HTTPS, while API and messaging routes keep their existing protocol behavior. Certbot stores challenge files and issued certificates under ignored local paths in `docker/ssl/`.

Prerequisites:

- `MG_PUBLIC_HOST` must resolve to the Docker host.
- Ports `80` and `443` must be reachable from the public internet.
- Set `MG_LETSENCRYPT_EMAIL` before requesting a certificate.

For a staging certificate, run one command from the project root:

```bash
make run_tls host=example.com email=admin@example.com
```

For a trusted production certificate, set `staging=false`:

```bash
make run_tls host=example.com email=admin@example.com staging=false
```

The target updates `docker/.env`, starts the Compose stack with the fallback certificate, runs certbot, switches Nginx to the issued certificate, and recreates Nginx. It also configures public UI URLs to `https://${MG_PUBLIC_HOST}`.

To configure the same instance without Let's Encrypt, use:

```bash
make run_tls host=example.com letsencrypt=false
```

That command updates `docker/.env`, comments out `MG_NGINX_SERVER_CERT` and `MG_NGINX_SERVER_KEY`, stops certbot if it exists, and runs the stack with the fallback Nginx certificate.

If you are replacing an existing valid certificate and want certbot to request a new one immediately, pass `force=true`:

```bash
make run_tls host=example.com email=admin@example.com staging=false force=true
```

The generated certificate paths in `docker/.env` are:

```env
MG_NGINX_SERVER_CERT=./ssl/letsencrypt/live/<host>/fullchain.pem
MG_NGINX_SERVER_KEY=./ssl/letsencrypt/live/<host>/privkey.pem
```

The setup script comments or uncomments those values automatically. Operators should not need to edit them by hand.

The certbot service keeps running and checks renewal twice a day. When a certificate is renewed, it sends a `HUP` signal to the Nginx process so new TLS handshakes use the renewed certificate.

## Makefile Integration

The included `Makefile` defines build and Docker‑build targets for all Magistrala services. Key points:

- `SERVICES`: list of services (auth, clients, channels, http, coap, mqtt, ws, etc.)

- `DOCKERS`, `DOCKERS_DEV`: build targets for production and development Docker images
- `make dockers`, `make dockers_dev`: always tag images as `ghcr.io/absmach/magistrala/<service>`

- Build arguments embed version, commit hash, and build timestamp into the binary

Build all services:

```bash
make all        # builds all services
make dockers    # builds all Docker images
```

Start services with Docker compose:

```bash
make run_latest
```

To clean up:

```bash
make cleandocker
```

To run tests(unit tests + API tests)

```bash
make test
```
