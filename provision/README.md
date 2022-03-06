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

| Variable                            | Description                                       | Default                               |
| ----------------------------------- | ------------------------------------------------- | --------------------------------------|
| MF_PROVISION_LOG_LEVEL              | Service log level                                 | debug                                 |
| MF_PROVISION_USER                   | User (email) for accessing Mainflux               | user@example.com                      |
| MF_PROVISION_PASS                   | Mainflux password                                 | user123                               |
| MF_PROVISION_API_KEY                | Mainflux authentication token                     |                                       |
| MF_PROVISION_CONFIG_FILE            | Provision config file                             | config.toml                           |
| MF_PROVISION_HTTP_PORT              | Provision service listening port                  | 8190                                  |
| MF_PROVISION_ENV_CLIENTS_TLS        | Mainflux SDK TLS verification                     | false                                 |
| MF_PROVISION_SERVER_CERT            | Mainflux gRPC secure server cert                  |                                       |
| MF_PROVISION_SERVER_KEY             | Mainflux gRPC secure server key                   |                                       |
| MF_PROVISION_USERS_LOCATION         | Users service URL                                 | http://users:8180                     |
| MF_PROVISION_THINGS_LOCATION        | Things service URL                                | http://things:8182                    |
| MF_PROVISION_BS_SVC_URL             | Mainflux Bootstrap service URL                    | http://bootstrap:8202/things          |
| MF_PROVISION_BS_SVC_WHITELIST_URL   | Mainflux Bootstrap service whitelist URL          | http://bootstrap:8202/things/state    |
| MF_PROVISION_CERTS_SVC_URL          | Certificates service URL                          | http://certs:8204/certs               |
| MF_PROVISION_X509_PROVISIONING      | Should X509 client cert be provisioned            | false                                 |
| MF_PROVISION_BS_CONFIG_PROVISIONING | Should thing config be saved in Bootstrap service | true                                  |
| MF_PROVISION_BS_AUTO_WHITELIST      | Should thing be auto whitelisted                  | true                                  |
| MF_PROVISION_BS_CONTENT             | Bootstrap service configs content, JSON format    | {}                                    |
| MF_PROVISION_CERTS_RSA_BITS         | Certificate RSA bits parameter                    | 4096                                  |
| MF_PROVISION_CERTS_HOURS_VALID      | Number of hours that certificate is valid         | "2400h"                               |

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
In order to create necessary entities provision service needs to authenticate against Mainflux. To provide authentication credentials to the provision service you can pass it in an environment variable or in a config file as Mainflux user and password or as API token (that can be issued on `/users` or `/keys` endpoint of [auth](../auth/README.md)).

Additionally users or API token can be passed in Authorization header, this authentication takes precedence over others.

* `username`, `password` - (`MF_PROVISION_USER`, `MF_PROVISION_PASSWORD` in [.env](../.env), `mf_user`, `mf_pass` in [config.toml](../docker/addons/provision/configs/config.toml))
* API Key - (`MF_PROVISION_API_KEY` in [.env](../.env) or [config.toml](../docker/addons/provision/configs/config.toml))
* `Authorization: Bearer Token|ApiKey` - request authorization header containing either users token or API key. Check [auth](../auth/README.md).

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
curl -s -S  -X POST  http://localhost:<MF_PROVISION_HTTP_PORT>/mapping  -H 'Content-Type: application/json' -d '{"external_id": "33:52:77:99:43", "external_key": "223334fw2"}'
```

In the case that provision service is not deployed with credentials or API key or you want to use user other than one being set in environment (or config file):
```bash
curl -s -S  -X POST  http://localhost:<MF_PROVISION_HTTP_PORT>/mapping -H "Authorization: Bearer <token|api_key>" -H 'Content-Type: application/json' -d '{"external_id": "<external_id>", "external_key": "<external_key>"}'
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

## Certificates
Provision service has `/certs` endpoint that can be used to generate certificates for things when mTLS is required:
- `users_token` - users authentication token or API token
- `thing_id` - id of the thing for which certificate is going to be generated
```bash
curl -s  -X POST  http://localhost:8190/certs -H "Authorization: Bearer <users_token>" -H 'Content-Type: application/json'   -d '{"thing_id": "<thing_id>", "key_bits":4096, "ttl":"2400h" }'
```
```json                                                                                                                            
{
  "thing_cert": "-----BEGIN CERTIFICATE-----\nMIIEmDCCA4CgAwIBAgIQCZ0NOq2oKLo+XftbAu0TfzANBgkqhkiG9w0BAQsFADBX\nMRIwEAYDVQQDDAlsb2NhbGhvc3QxETAPBgNVBAoMCE1haW5mbHV4MQwwCgYDVQQL\nDANJb1QxIDAeBgkqhkiG9w0BCQEWEWluZm9AbWFpbmZsdXguY29tMB4XDTIwMDYw\nNTEyMzc1M1oXDTIwMDkxMzEyMzc1M1owVTERMA8GA1UEChMITWFpbmZsdXgxETAP\nBgNVBAsTCG1haW5mbHV4MS0wKwYDVQQDEyQyYmZlYmZmMC05ODZhLTQ3ZTAtOGQ3\nYS00YTRiN2UyYjU3OGUwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCn\nWvTuOIdhqOLEREcEJqfQAtDoYu3rUDijOffXuWFZgNqfZTGmoD5ZqJXxwbZ4tCST\npdSteHtyr7JXnPJQN1dsslU+q3haKjFoZRc39/7u4/8XCTwlqbMl9YVcwqS+FLkM\niLSyyqzryP7Y8H8cidTKg56p5JALaEKfzZS6Km3G+CCinR6hNNW9ckWsy29a0/9E\nMAUtM+Lsk5OjsHzOnWruuqHsCx4ODI5aJQaMC1qntkbXkht0WDiwAt9SDQ3uLWru\nAoSJDK9a6EgR3a0Jf7ZiVPiwlZNjrB/I5OQyFDGqcmSAl2rdJqPkmaDXKKFyL1cG\nMIyHv62QzJoMdRoXu20lxyGxAvEjQNVHux4LA3dbf/85nEVTI2uP8crMf2Jnzbg5\n9zF+iTMJGpUlatCyK2RJS/mvHbbUIf5Ro3VbcPHbgFroJ7qMFz0Fc5kYY8IdwXjG\nlyG9MobKEO2CfBGRjPmCuTQq2HcuOy7F6KfQf3HToI8MmC5hBtCmTNbV8I3GIjWA\n/xJQLm2pVZ41QhrnNGtuqAYoe3Zt6OldxGRcoAj7KlIpYcPZ55PJ6mWcV6dB9Fnl\n5mYOwQL8jtfybbGWvqJldhTxUqm7/EbAaF0Qjmh4oOHMl2xADrmYzJHvf0llwr6g\noRQuzqxPi0aW3tkFNsm63NX1Ab5BXFQhMSj5+82blwIDAQABo2IwYDAOBgNVHQ8B\nAf8EBAMCB4AwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMA4GA1UdDgQH\nBAUBAgMEBjAfBgNVHSMEGDAWgBRs4xR91qEjNRGmw391xS7x6Tc+8jANBgkqhkiG\n9w0BAQsFAAOCAQEAphLT8PjawRRWswU1B5oWnnqeTllnvGB88sjDPLAG0UiBlDLX\nwoPiBVPWuYV+MMJuaREgheYF1Ahx4Jrfy9stFDU7B99ON1T58oM1aKEq4rKc+/Ke\nyxrAFTonclC0LNaaOvpZZjsPFWr2muTQO8XHiS8icw3BLxEzoF+5aJ8ihtxRtfKL\nUvtHDqC6IPAbSUcvqyjrFh3RrTUAyGOzW12IEWSXP9DLwoiLPwJ6kCVoXdG/asjz\nUpk/jj7AUn9oJNF8nUbyhdOnmeJ2z0x1ylgYrIAxvGzm8zs+NEVN67CrBYKwstlN\nvw7DRQsCvGJjZzWj28VV3FGLtXFgu52bFZNBww==\n-----END CERTIFICATE-----\n",                            
  "thing_cert_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIJJwIBAAKCAgEAp1r07jiHYajixERHBCan0ALQ6GLt61A4ozn317lhWYDan2Ux\npqA+WaiV8cG2eLQkk6XUrXh7cq+yV5zyUDdXbLJVPqt4WioxaGUXN/f+7uP/Fwk8\nJamzJfWFXMKkvhS5DIi0ssqs68j+2PB/HInUyoOeqeSQC2hCn82Uuiptxvggop0e\noTTVvXJFrMtvWtP/RDAFLTPi7JOTo7B8zp1q7rqh7AseDgyOWiUGjAtap7ZG15Ib\ndFg4sALfUg0N7i1q7gKEiQyvWuhIEd2tCX+2YlT4sJWTY6wfyOTkMhQxqnJkgJdq\n3Saj5Jmg1yihci9XBjCMh7+tkMyaDHUaF7ttJcchsQLxI0DVR7seCwN3W3//OZxF\nUyNrj/HKzH9iZ824OfcxfokzCRqVJWrQsitkSUv5rx221CH+UaN1W3Dx24Ba6Ce6\njBc9BXOZGGPCHcF4xpchvTKGyhDtgnwRkYz5grk0Kth3Ljsuxein0H9x06CPDJgu\nYQbQpkzW1fCNxiI1gP8SUC5tqVWeNUIa5zRrbqgGKHt2bejpXcRkXKAI+ypSKWHD\n2eeTyeplnFenQfRZ5eZmDsEC/I7X8m2xlr6iZXYU8VKpu/xGwGhdEI5oeKDhzJds\nQA65mMyR739JZcK+oKEULs6sT4tGlt7ZBTbJutzV9QG+QVxUITEo+fvNm5cCAwEA\nAQKCAgAmCIfNc89gpG8Ux6eUC+zrWxh7F7CWX97fSZdH0XuMSbplqyvDgHtrCOM6\n1BlSCS6e13skCVOU1tUjECoJjOoza7vvyCxL4XblEMRcFeI8DFi2tYST0qNCJzAt\nypaCFFeRv6fBUkpGM6GnT9Czfad8drkiRy1tSj6J7sC0JlxYcZ+JFUgWvtksesHW\n6UzfSXqj1n32reoOdeOBueRDWIcqxgNyj3w/GR9o4S1BunrZzpT+/Nd8c2g+qAh0\nrz7ROEUq3iucseNQN6XZWZWvqPScGE+EYhni9wUqNMqfjvNSlzi7+K1yoQtyMm/Z\nNgSq3JNcdsAZQbiCRd1ko2BQsGm3ZBnbsAJ1Dxcn+i9nF5DT/ddWjUWin6LYWuUM\n/0Bqfv3etlrFuP6yxc8bPEMX0ucJg4yVxdkDrm1tYlJ+ANEQoOlZqhngvjz0f8uO\nOtEcDLmiG5VG6Yl72UtWIw+ALnKc5U7ib43Qve0bDAKR5zlHODcRetN9BCMvpekY\nOA4hohkllTP25xmMzLokBqY9n38zEt74kJOp67VKMvhoF7QkrLOfKWCRJjFL7/9I\nHDa6jb31INA9Wu+p/2LIa6I1SUYnMvCUqISgF2hBG9Q9S9TZvKnYUvfurhFS9jZv\n18sxW7IFYWmQyioo+gsAmfKLolJtLl9hCmTfYi7oqCh/EtZdIQKCAQEA0Umkp0Uu\nimVilLjgYGTWLcg8T3NWaELQzb2HYRXSzEq/M8GOtEr7TR7noJBm8fcgl55HEnPl\ni4cEJrr+VprzGbdMtXjHbCD+I945GA6vv3khg7mbqS9a1Uw6gjrQEZgZQU+/IVCu\n9Pbvx8Af32xaBWuN2cFzC7Z6iB815LPc2O5qyZ3+3nEUPah+Z+a9WEeTR6M0hy5c\nkkaRqhehugHDgqMRWGt8GfsFOmaR13kvfFfKadPRPkaGkftCSKBMWjrU4uX7aulm\nD7k4VDbnXIBMhI039+0znSkhZdcV1zk6qwBYn9TtZ11PTlspFPjtPxqS5M6IGflw\nsXkZGv4rZ5CkiQKCAQEAzLVdw2qw/8rWGsCV39EKp7hXLvp7+FuodPvX1L55lWB0\nvmSOldGcNvb2ZsK3RNvgteb8VfKRgaY6waeN5Qm1UXazsOX4F+GThPGHstdNuzkt\nJofRQQHQVR3npZbCngSkSZdahQ9SjiLIDKn8baPN8I8HfpJ4oHLUvkayavbch1kJ\nYWUfGtVKxHGX5m/nnxLdgbJEx9Q+3Qa7DDHuxTqsEqhkk0R0Ganred34HjpDNMs6\nV95HFNolW3yKfuHETKA1bLhej+XdMa11Ts5hBVGCMnnT07WcGhxtyK2dSa656SyT\ngT9+Hd1VWZ/KPpAkQmH9boOr2ihE+oAXiZ4D1t53HwKCAQAD0cA7fTu4Mtl1tVoC\n6FQwSbMwD/7HsFB3MLpDv041hDexDhs4lxW29pVrjLcUO1pQ6gaKA6twvGoK+uah\nVfqRwZKYzTd2dbOtm+SW183FRMSjzsNUdxTFR7rZnZEmgQwU8Quf5AUNW2RM1Oi/\n/w41gxz3mFwtHotl6IvnPJEPNGqme0enb5Da/zQvWTqjXcsGR6gxv1rZIIiP/hZp\nepbCz48FehCtuLMDudN3hzKipkd/Xuo2pLrX9ynigWpjSyePbHsGHHRMXSj2AHqA\naab71EftMlr6x0FgxmgToWu8qyjy4cPjWwSTfX5mb5SEzktX+ZzqPG8eDgOzRmgs\nX6thAoIBADL3kQG/hZQaL1Z3zpjsFggOKH7E1KrQP0/pCCKqzeC4JDjnFm0MxCUX\nNd/96N1XFUqU2QyZGUs7VPO0QOrekOtYb4LCrxNbEXyPGicX3f2YTbqDJEFYL0OR\n74PV1ly7cR/1dA8e8oH6/O3SQMwXdYXIRqhn1Wq1TGyXc4KYNe3o6CH8qFLo+fWR\nBq3T/MopS0coWGGcYY5sR5PQts8aPY9jp67W40UkfkFYV5dHEEaLttn7uJzjd1ug\n1Waj1VjypnqMKNcQ9xKQSl21mohVc+IXXPsgA16o51iIiVm4DAeXFp6ebUsIOWDY\nHOWYw75XYV7rn5TwY8Qusi2MTw5nUycCggEAB/45U0LW7ZGpks/aF/BeGaSWiLIG\nodBWUjRQ4w+Le/pTC8Ci9fiidxuCDH6TQbsUTGKOk7GsfncWHTQJogaMyO26IJ1N\nmYGgK2JJvs7PKyIkocPDVD/Yh0gIzQIE92ZdyXUT21pIYKDUB9e3p0fy/+E0pyeI\nsmsV8oaLr4tZRY1cMogI+pvtUUferbLQmZHhFd9X3m3RslR43Dl1qpYQyzE3x/a3\nWA2NJZbJhh+LiAKzqk7swXOqrTrmXuzLcjMG+T/3lizrbLLuKjQrf+eehlpw0db0\nHVVvkMLOP5ZH/ImkmvOZJY7xxup89VV7LD7TfMKwXafOrjMDdvTAYPtgxw==\n-----END RSA PRIVATE KEY-----\n"
}
```

[mainflux]: https://github.com/mainflux/mainflux
[bootstrap]: https://github.com/mainflux/mainflux/tree/master/bootstrap
[export]: https://github.com/mainflux/export
[agent]: https://github.com/mainflux/agent
[mfxui]: https://github.com/mainflux/mainflux/ui
