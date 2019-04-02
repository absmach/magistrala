# **SECURING COMMUNICATION**

By default gRPC communication is not secure as Mainflux system is most often run in a private network behind the reverse proxy.

However, TLS can be activated and configured.

# Server configuration

## Securing PostgreSQL connections

By default, Mainflux will connect to Postgres using insecure transport.
If a secured connection is required, you can select the SSL mode and set paths to any extra certificates and keys needed. 

`MF_USERS_DB_SSL_MODE` the SSL connection mode for Users.
`MF_USERS_DB_SSL_CERT` the path to the certificate file for Users.
`MF_USERS_DB_SSL_KEY` the path to the key file for Users.
`MF_USERS_DB_SSL_ROOT_CERT` the path to the root certificate file for Users.

`MF_THINGS_DB_SSL_MODE` the SSL connection mode for Things.
`MF_THINGS_DB_SSL_CERT` the path to the certificate file for Things.
`MF_THINGS_DB_SSL_KEY` the path to the key file for Things.
`MF_THINGS_DB_SSL_ROOT_CERT` the path to the root certificate file for Things.

Supported database connection modes are: `disabled` (default), `required`, `verify-ca` and `verify-full`.

## Securing gRPC

### Users

If either the cert or key is not set, the server will use insecure transport.

`MF_USERS_SERVER_CERT` the path to server certificate in pem format.

`MF_USERS_SERVER_KEY` the path to the server key in pem format.

### Things

If either the cert or key is not set, the server will use insecure transport.

`MF_THINGS_SERVER_CERT` the path to server certificate in pem format.

`MF_THINGS_SERVER_KEY` the path to the server key in pem format.

## Client configuration

If you wish to secure the gRPC connection to `things` and `users` services you must define the CAs that you trust.  This does not support mutual certificate authentication.

### HTTP Adapter

`MF_HTTP_ADAPTER_CA_CERTS` - the path to a file that contains the CAs in PEM format. If not set, the default connection will be insecure. If it fails to read the file, the adapter will fail to start up.

### Things

`MF_THINGS_CA_CERTS` - the path to a file that contains the CAs in PEM format. If not set, the default connection will be insecure. If it fails to read the file, the service will fail to start up.

# Mutual authentication

In the most of the cases, HTTPS, WSS, MQTTS or secure CoAP are secure enough. However, sometimes you might need even more secure connection. Mainflux supports mutual TLS authentication (_mTLS_) based on (X.509 certificates)[https://tools.ietf.org/html/rfc5280]. By default the TLS protocol only proves the identity of the server to the client using X.509 certificate and the authentication of the client to the server is left to the application layer. TLS also offers client-to-server authentication using client-side X.509 authentication. This is called two-way or mutual authentication. Mainflux currently supports mTLS over HTTP, WS, and MQTT protocols. In order to run Docker composition with mTLS turned on, you can execute following command from the project root:

```bash
AUTH=x509 docker-compose -f docker/docker-compose.yml up -d
```

Mutual authentication includes client side certificates. Certificates can be generated using simple script provided (here)[http://www.github.com/mainflux/mainflux/tree/master/docker/ssl/Makefile]. In order to create a valid certificate, you need to create Mainflux thing using the process described in the [provisioning section](provisioning.md). After that, you need to fetch created thing key. Thing key will be used to create x.509 certificate for corresponding thing. TO create certificate, execute following commands:

```bash
cd docker/ssl
make ca
make server_cert
make thing_cert KEY=<thing_key> CRT_FILE_NAME=<cert_name>
```
These commands use (OpenSSL)[https://www.openssl.org/] tool, so please make sure that you have it installed and set up before running these commands.

    - Command `make ca` wil generate self-signed certificate that will later be used as a CA to sign other generated certificates. CA will expire in 3 years.
    - Command `make server_cert` will generated and sign (with previously created CA) server cert, which will expire after 1000 days. This cert is used as a Mainflux server-side certificate in usual TLS flow to establish HTTPS, WSS, or MQTTS connection.
    - Command `make thing_cert` wil finally generate and sign client-side certificate and private key for the thing.

In this example `<thing_key>` represents key of the thing, and `<cert_name>` represents name of the certificate and key file which will be saved in `docker/ssl/certs` directory. Generated Certificate will expire after 2 years. The key must be stored in the x.509 certificate "CN" field.  This script is created for the testing purposes and is not meant to be used in production. We strongly recommend avoiding self-signed certificates and using certificate management tool such as (Vault)[https://www.vaultproject.io/] for the production.

Once you have created CA and server-side cert, you can spin the composition using:

```bash
AUTH=x509 docker-compose -f docker/docker-compose.yml up -d
```

Then, you can create user and provision things and channels. Now, in order to send a message from the specific thing to the channel, you need to connect thing to the channel and generate corresponding client certificate using aforementioned commands. To publish a message to the channel, thing should send following request:

_HTTPS:_
```bash
curl -s -S -i --cacert docker/ssl/certs/ca.crt --cert docker/ssl/certs/<thing_cert_name>.crt --key docker/ssl/certs/<thing_cert_key>.key --insecure -X POST -H "Content-Type: application/senml+json" https://localhost/http/channels/<channel_id>/messages -d '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```

_MQTTS_:

###### PUBLISH
```bash
mosquitto_pub -u <thing_id> -P <thing_key> -t channels/<channel_id>/messages -h localhost  --cafile docker/ssl/certs/ca.crt --cert docker/ssl/certs/<thing_cert_name>.crt --key docker/ssl/certs/<thing_cert_key>.key -m '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```
###### SUBSCRIBE
```
mosquitto_sub -u <thing_id> -P <thing_key> --cafile docker/ssl/certs/ca.crt --cert docker/ssl/certs/<thing_cert_name>.crt --key docker/ssl/certs/<thing_cert_key>.key -t channels/<channel_id>/messages -h localhost
```

_WSS:_
```javascript
const WebSocket = require('ws');

// Do not verify self-signed certificates if you are using one.
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0'

// Replace <channel_id> and <thing_key> with real values.
const ws = new WebSocket('wss://localhost/ws/channels/<channel_id>/messages?authorization=<thing_key>',
// This is ClientOptions object that contains client cert and client key in the form of string. You can easily load these strings from cert and key files.
{
    cert: `-----BEGIN CERTIFICATE-----....`,
    key: `-----BEGIN RSA PRIVATE KEY-----.....`
})

ws.on('open', () => {
    ws.send('something')
})

ws.on('message', (data) => {
    console.log(data)
})
ws.on('error', (e) => {
    console.log(e)
})
```

As you can see, `Authorization` header does not have to be present in the the HTTP request, since the key is present in the certificate. However, if yoy pass `Authorization` header, it _must be the same as the key in the cert_. In the case of MQTTS, `password` filed in CONNECT message _must match the key from the certificate_. In the case of WSS, `Authorization` header or `authorization` query parameter _must match cert key_.