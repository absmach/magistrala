# Provision service

Provision service provides an HTTP API to interact with [Mainflux][mainflux]. 
Provision service is used to setup initial applications configuration i.e. things, channels, connections and certificates that will be required for the specific use case especially useful for gateway provision.  

For gateways to communicate with [Mainflux][mainflux] configuration is required (mqtt host, thing, channels, certificates...). To get the configuration gateway will send a request to [Bootstrap][bootstrap] service providing `<external_id>` and `<external_key>` in request. To make a request to [Bootstrap][bootstrap] service you can use [Agent][agent] service on a gateway.  

To create bootstrap configuration you can use [Bootstrap][bootstrap] or `Provision` service. [Mainflux UI][mfxui] uses [Bootstrap][bootstrap] service for creating gateway configurations.  `Provision` service should provide an easy way of provisioning your gateways i.e creating bootstrap configuration and as many things and channels that your setup requires.  

Also you may use provision service to create certificates for each thing. Each service running on gateway may require more than one thing and channel for communication. Let's say that you are using services [Agent][agent] and [Export](https://github.com/mainflux/export) on a gateway you will need two channels for `Agent` (`data` and `control`) and one for `Export` and one thing. Additionally if you enabled mtls each service will need its own thing and certificate for access to [Mainflux][mainflux]. Your setup could require any number of things and channels this kind of setup we can call `provision layout`.

Provision service provides a way of specifying this `provision layout` and creating a setup according to that layout by serving requests on `/mapping` endpoint. Provision layout is configured in [config.toml](configs/config.toml).

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                       | Default                          |
| ----------------------------------- | ------------------------------------------------- | -------------------------------- |
| MF_PROVISION_USER                   | User (email) for accessing Mainflux               | user@example.com                 |
| MF_PROVISION_PASS                   | Mainflux password                                 | user123                          |
| MF_PROVISION_API_KEY                | Mainflux authentication token                     |                                  |
| MF_PROVISION_CONFIG_FILE            | Provision config file                             | config.toml                    |
| MF_PROVISION_HTTP_PORT              | Provision service listening port                  | 8091                             |
| MF_PROVISION_ENV_CLIENTS_TLS        | Mainflux SDK TLS verification                     | false                            |
| MF_PROVISION_CA_CERTS               | Mainflux gRPC secure certs                        |                                  |
| MF_PROVISION_SERVER_CERT            | Mainflux gRPC secure server cert                  |                                  |
| MF_PROVISION_SERVER_KEY             | Mainflux gRPC secure server key                   |                                  |
| MF_PROVISION_SERVER_KEY             | Mainflux gRPC secure server key                   |                                  |
| MF_PROVISION_MQTT_URL               | Mainflux MQTT adapter URL                         | http://localhost:1883          |
| MF_PROVISION_USERS_LOCATION         | Users service URL                                 | http://locahost                |
| MF_PROVISION_THINGS_LOCATION        | Things service URL                                | http://localhost               |
| MF_PROVISION_LOG_LEVEL              | Service log level                                 | http://localhost               |
| MF_PROVISION_HTTP_PORT              | Service listening port                            | 8091                           |
| MF_PROVISION_USER                   | Mainflux user username                            | test@example.com               |
| MF_PROVISION_PASS                   | Mainflux user password                            | password                       |
| MF_PROVISION_BS_SVC_URL             | Mainflux Bootstrap service URL                    | http://localhost/things/configs |
| MF_PROVISION_BS_SVC_WHITELIST_URL   | Mainflux Bootstrap service whitelist URL          | http://localhost/things/state  |
| MF_PROVISION_CERTS_SVC_URL          | Certificats service URL                           | http://localhost/certs         |
| MF_PROVISION_X509_PROVISIONING      | Should X509 client cert be provisioned            | false                          |
| MF_PROVISION_BS_CONFIG_PROVISIONING | Should thing config be saved in Bootstrap service | true                           |
| MF_PROVISION_BS_AUTO_WHITEIST       | Should thing be auto whitelisted                  | true                           |
| MF_PROVISION_BS_CONTENT             | Bootstrap service content                         | {}                             |

By default, call to `/mapping` endpoint will create one thing and two channels (`control` and `data`) and connect it. If there is a requirement for different provision layout we can use [config](docker/configs/config.toml) file in addition to environment variables. 

For the purposes of running provision as an add-on in docker composition environment variables seems more suitable. Environment variables are set in [.env](.env).  

Configuration can be specified in [config.toml](configs/config.toml). Config file can specify all the settings that environment variables can configure and in addition
`/mapping` endpoint provision layout can be configured.

In `config.toml` we can enlist array of things and channels that we want to create and make connections between them which we call provision layout.

Metadata can be whatever suits your needs except that at least one thing needs to have `external_id` (which is populated with value from [request](#example)). Thing that has `external_id` will be used for creating bootstrap configuration which can be fetched with [Agent][agent].
For channels metadata `type` is reserved for `control` and `data` which we use with [Agent][agent].

Example of provision layout below
```toml
[[things]]
  name = "thing"

  [things.metadata]
    external_id = "xxxxxx"


[[channels]]
  name = "control-channel"

  [channels.metadata]
    type = "control"

[[channels]]
  name = "data-channel"

  [channels.metadata]
    type = "data"

[[channels]]
  name = "export-channel"

  [channels.metadata]
    type = "data"
```

## Authentication
In order to create necessary entities provision service needs to authenticate against Mainflux. To provide authentication credentials to the provision service you can pass it in an environment variable or in a config file as Mainflux user and password or as API token (that can be issued on `/users` or `/keys` endpoint of [authn](../authn/README.md)). 

Additionally users or API token can be passed in Authorization header, this authentication takes precedence over others.

* `username`, `password` - (`MF_PROVISION_USER`, `MF_PROVISION_PASSWORD` in [.env](../.env), `mf_user`, `mf_pass` in [config.toml](../docker/addons/provision/configs/config.toml))
* API Key - (`MF_PROVISION_API_KEY` in [.env](../.env) or [config.toml](../docker/addons/provision/configs/config.toml))
* `Authorization: Token|ApiKey` - request authorization header containing either users token or API key. Check [authn](../authn/README.md).

## Running
Provision service can be run as a standalone or in docker composition as addon to the core docker composition.

Standalone:
```bash
MF_PROVISION_BS_SVC_URL=http://localhost:8202/things \
MF_PROVISION_THINGS_LOCATION=http://localhost:8182 \
MF_PROVISION_USERS_LOCATION=http://localhost:8180 \
MF_PROVISION_CONFIG_FILE=docker/addons/provision/configs/config.toml \
build/mainflux-provision
```

Docker composition:
```bash
docker-compose -f docker/addons/provision/docker-compose.yml up
```

For the case that credentials or API token is passed in configuration file or environment variables, call to `/mapping` endpoint doesn't require `Authentication` header:
```bash
curl -s -S  -X POST  http://localhost:8888/mapping  -H 'Content-Type: application/json' -d '{"external_id": "33:52:77:99:43", "external_key": "223334fw2"}'
```

In the case that provision service is not deployed with credentials or API key or you want to use user other than one being set in environment (or config file):
```bash
curl -s -S  -X POST  http://localhost:8091/mapping -H "Authorization: <token|api_key>" -H 'Content-Type: application/json' -d '{"external_id": "<external_id>", "external_key": "<external_key>"}'
```

Or if you want to specify a name for thing different than in `config.toml` you can specify post data as:

```json
{"name": "<name>", "external_id": "<external_id>", "external_key": "<external_key>"}
```

Response contains created things, channels and certificates if any:
```json
{
  "things": [
    {
      "id": "c22b0c0f-8c03-40da-a06b-37ed3a72c8d1",
      "name": "thing",
      "key": "007cce56-e0eb-40d6-b2b9-ed348a97d1eb",
      "metadata": {
        "external_id": "33:52:79:C3:43"
      }
    }
  ],
  "channels": [
    {
      "id": "064c680e-181b-4b58-975e-6983313a5170",
      "name": "control-channel",
      "metadata": {
        "type": "control"
      }
    },
    {
      "id": "579da92d-6078-4801-a18a-dd1cfa2aa44f",
      "name": "data-channel",
      "metadata": {
        "type": "data"
      }
    }
  ],
  "whitelisted": {
    "c22b0c0f-8c03-40da-a06b-37ed3a72c8d1": true
  }
}
```

[mainflux]: https://github.com/mainflux/mainflux
[bootstrap]: https://github.com/mainflux/mainflux/tree/master/bootstrap
[export]: https://github.com/mainflux/export
[agent]: https://github.com/mainflux/agent
[mfxui]: https://github.com/mainflux/mainflux/ui