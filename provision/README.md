# PROVISION service

PROVISION service provides an HTTP API to interact with Mainflux.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                  | Description                                       | Default                                  |
|---------------------------|---------------------------------------------------|------------------------------------------|
| MF_USER                   | User (email) for accessing Mainflux               |  user@example.com                        |
| MF_PASS                   | Mainflux password                                 |  user123                                 |
| MF_PROVISION_HTTP_PORT    | Provision service listening port                  |  8091                                    |
| MF_ENV_CLIENTS_TLS        | Mainflux SDK TLS verification                     |  false                                   |
| MF_PROVISION_CA_CERTS     | Mainflux gRPC secure certs                        |  ""                                      |
| MF_PROVISION_SERVER_CERT  | Mainflux gRPC secure server cert                  | ""                                       |
| MF_PROVISION_SERVER_KEY   | Mainflux gRPC secure server key                   | ""                                       |
| MF_PROVISION_SERVER_KEY   | Mainflux gRPC secure server key                   | ""                                       |
| MF_MQTT_URL               | Mainflux MQTT adapter URL                         | "http://localhost:1883"                  |
| MF_USERS_LOCATION         | Users service URL                                 | "http://locahost"                        |
| MF_THINGS_LOCATION        | Things service URL                                | "http://localhost"                       |
| MF_PROVISION_LOG_LEVEL    | Service log level                                 | "http://localhost"                       |
| MF_PROVISION_HTTP_PORT    | Service listening port                            | "8091"                                   |
| MF_USER                   | Mainflux user username                            | "test@example.com"                       |
| MF_PASS                   | Mainflux user password                            | "password"                               |
| MF_BS_SVC_URL             | Mainflux Bootstrap service URL                    | http://localhost/things/configs"         |
| MF_BS_SVC_WHITELISTE_URL  | Mainflux Bootstrap service whitelist URL          | "http://localhost/things/state"          |
| MF_CERTS_SVC_URL          | Certificats service URL                           | "http://localhost/certs"                 |
| MF_X509_PROVISIONING      | Should X509 client cert be provisioned            | "false"                                  |
| MF_BS_CONFIG_PROVISIONING | Should thing config be saved in Bootstrap service | "true"                                   |
| MF_BS_AUTO_WHITEIST       | Should thing be auto whitelisted                  | "true"                                   |
| MF_BS_CONTENT             | Bootstrap service content                         | "{}"


## Example 
```
curl -X POST \
  http://localhost:8091/mapping\
  -H 'Content-Type: application/json' \
  -d '{ "externalid" : "02:42:fE:65:CB:3d", "externalkey: "key12345678" }'
  ```
