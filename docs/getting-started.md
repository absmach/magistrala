## Prerequisites

Before proceeding, install the following prerequisites:

- [Docker](https://docs.docker.com/install/)
- [Docker compose](https://docs.docker.com/compose/install/)
- [jsonpp](https://jmhodges.github.io/jsonpp/) (optional)

Once everything is installed, execute the following commands from project root:

```bash
docker-compose up -f docker/docker-compose.yml -d
```

## User management

### Account creation

Use the Mainflux API to create user account:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json; charset=utf-8" https://localhost/users -d '{"email":"john.doe@email.com", "password":"123"}'
```

Note that when using official `docker-compose`, all services are behind `nginx` 
proxy and all traffic is `TLS` encrypted.

### Obtaining an authorization key

In order for this user to be able to authenticate to the system, you will have
to create an authorization token for him:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json; charset=utf-8" https://localhost/tokens -d '{"email":"john.doe@email.com", "password":"123"}'
```

Response should look like this:
```
{
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MjMzODg0NzcsImlhdCI6MTUyMzM1MjQ3NywiaXNzIjoibWFpbmZsdXgiLCJzdWIiOiJqb2huLmRvZUBlbWFpbC5jb20ifQ.cygz9zoqD7Rd8f88hpQNilTCAS1DrLLgLg4PRcH-iAI"
}
```

## System provisioning

Before proceeding, make sure that you have created a new account, and obtained
an authorization key.

### Provisioning devices

Devices are provisioned by executing request `POST /clients`, with a 
`"type":"device"` specified in JSON payload. Note that you will also need 
`user_auth_token` in order to provision clients (both devices and application) 
that belong to this particular user.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json; charset=utf-8" -H "Authorization: <user_auth_token>" https://localhost/clients -d '{"type":"device", "name":"weio"}'
```

Response will contain `Location` header whose value represents path to newly 
created client:

```
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Location: /clients/81380742-7116-4f6f-9800-14fe464f6773
Date: Tue, 10 Apr 2018 10:02:59 GMT
Content-Length: 0
```

### Provisioning applications

Applications are provisioned by executing HTTP request `POST /clients`, with 
`"type":"app"` specified in JSON payload.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json; charset=utf-8" -H "Authorization: <user_auth_token>" https://localhost/clients -d '{"type":"app", "name":"myapp"}'
```

Response will contain `Location` header whose value represents path to newly 
created client (same as for devices):

```
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Location: /clients/cb63f852-2d48-44f0-a0cf-e450496c6c92
Date: Tue, 10 Apr 2018 10:33:17 GMT
Content-Length: 0
```

### Retrieving provisioned clients

In order to retrieve data of provisioned clients that is written in database, you 
can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/clients
```

Notice that you will receive only those clients that were provisioned by 
`user_auth_token` owner.

```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Count: 4
Date: Tue, 10 Apr 2018 10:50:12 GMT
Content-Length: 1105

{
  "clients": [
    {
      "id": "81380742-7116-4f6f-9800-14fe464f6773",
      "type": "device",
      "name": "weio",
      "key": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1MjMzNTQ1NzksImlzcyI6Im1haW5mbHV4Iiwic3ViIjoiODEzODA3NDItNzExNi00ZjZmLTk4MDAtMTRmZTQ2NGY2NzczIn0.5s8s1hlK-l30kQAyHxEZO_M2NIQw53MQuy7b3Wf3OOE"
    },
    {
      "id": "cb63f852-2d48-44f0-a0cf-e450496c6c92",
      "type": "app",
      "name": "myapp",
      "key": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1MjMzNTYzOTcsImlzcyI6Im1haW5mbHV4Iiwic3ViIjoiY2I2M2Y4NTItMmQ0OC00NGYwLWEwY2YtZTQ1MDQ5NmM2YzkyIn0.FE6DWB3yJmBb8uojpQJaKUEbD0Elrjx0HhJA28bVzkU"
    }
  ]
}
```

### Removing clients

In order to remove you own client you can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/clients/<client_id>
```

### Provisioning channels

Channels are provisioned by executing request `POST /channels`:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json; charset=utf-8" -H "Authorization: <user_auth_token>" https://localhost/channels -d '{"name":"mychan"}'
```

After sending request you should receive response with `Location` header that 
contains path to newly created channel:

```
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
Location: /channels/19daa7a8-a489-4571-8714-ef1a214ed914
Date: Tue, 10 Apr 2018 11:30:07 GMT
Content-Length: 0
```

### Retrieving provisioned channels

To retreve provisioned channels you should send request to `/channels` with
authorization token in `Authorization` header:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels
```

You should receive response similar to this:

```
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Count: 2
Date: Tue, 10 Apr 2018 11:38:06 GMT
Content-Length: 139

{
  "channels": [
    {
      "id": "19daa7a8-a489-4571-8714-ef1a214ed914",
      "name": "mychan"
    }
  ]
}
```

Note that you will receive only those channels that were created by authorization
token's owner.

### Removing channels

In order to remove specific channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

## Access control

Channel can be observed as a communication group of clients. Only clients that
are connected to the channel can send and receive messages from other clients
in this channel. Clients that are not connected to this channel are not allowed
to communicate over it.

Only user, who is the owner of a channel and of the clients, can connect the 
clients to the channel (which is equivalent of giving permissions to these clients 
to communicate over given communication group).

To connect client to the channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X PUT -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/clients/<client_id>
```

You can observe which clients are connected to specific channel:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

You should receive response with the lists of connected clients in `connected` field
similar to this one:

```
{
  "id": "19daa7a8-a489-4571-8714-ef1a214ed914",
  "name": "mychan",
  "connected": [
    {
      "id": "81380742-7116-4f6f-9800-14fe464f6773",
      "type": "device",
      "name": "weio",
      "key": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1MjMzNTQ1NzksImlzcyI6Im1haW5mbHV4Iiwic3ViIjoiODEzODA3NDItNzExNi00ZjZmLTk4MDAtMTRmZTQ2NGY2NzczIn0.5s8s1hlK-l30kQAyHxEZO_M2NIQw53MQuy7b3Wf3OOE"
    }
  ]
}
```

If you want to disconnect your device from the channel, send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/clients/<client_id>
```

## Sending messages

Once a channel is provisioned and client is connected to it, it can start to
publish messages on the channel. The following sections will provide an example
of message publishing for each of the supported protocols.

### HTTP

To publish message over channel, client should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/senml+json" -H "Authorization: <client_token>" https://localhost/channels/<channel_id>/messages -d -d '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]'
```

Note that you should always send array of messages in senML format.