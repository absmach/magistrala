Provisioning is a process of configuration of an IoT platform in which system operator creates and sets-up different entities
used in the platform - users, channels and things.

## User management

### Account creation

Use the Mainflux API to create user account:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/users -d '{"email":"john.doe@email.com", "password":"123"}'
```

Note that when using official `docker-compose`, all services are behind `nginx`
proxy and all traffic is `TLS` encrypted.

### Obtaining an authorization key

In order for this user to be able to authenticate to the system, you will have
to create an authorization token for him:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/tokens -d '{"email":"john.doe@email.com", "password":"123"}'
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

Devices are provisioned by executing request `POST /things`, with a
`"type":"device"` specified in JSON payload. Note that you will also need
`user_auth_token` in order to provision things (both devices and application)
that belong to this particular user.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/things -d '{"type":"device", "name":"weio"}'
```

Response will contain `Location` header whose value represents path to newly
created thing:

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /things/81380742-7116-4f6f-9800-14fe464f6773
Date: Tue, 10 Apr 2018 10:02:59 GMT
Content-Length: 0
```

### Provisioning applications

Applications are provisioned by executing HTTP request `POST /things`, with
`"type":"app"` specified in JSON payload.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/things -d '{"type":"app", "name":"myapp"}'
```

Response will contain `Location` header whose value represents path to newly
created thing (same as for devices):

```
HTTP/1.1 201 Created
Content-Type: application/json
Location: /things/cb63f852-2d48-44f0-a0cf-e450496c6c92
Date: Tue, 10 Apr 2018 10:33:17 GMT
Content-Length: 0
```

### Retrieving provisioned things

In order to retrieve data of provisioned things that is written in database, you
can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things
```

Notice that you will receive only those things that were provisioned by
`user_auth_token` owner.

```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 10 Apr 2018 10:50:12 GMT
Content-Length: 1105

{
  "things": [
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
      "key": "cbf02d60-72f2-4180-9f82-2c957db929d1"
    }
  ]
}
```

You can specify `offset` and `limit` parameters in order to fetch specific
group of things. In that case, your request should look like:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things?offset=0&limit=5
```

If you don't provide them, default values will be used instead: 0 for `offset`,
and 10 for `limit`. Note that `limit` cannot be set to values greater than 100. Providing
invalid values will be considered malformed request.

### Removing things

In order to remove you own thing you can send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/things/<thing_id>
```

### Provisioning channels

Channels are provisioned by executing request `POST /channels`:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/channels -d '{"name":"mychan"}'
```

After sending request you should receive response with `Location` header that
contains path to newly created channel:

```
HTTP/1.1 201 Created
Content-Type: application/json
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

Note that you will receive only those channels that were created by authorization
token's owner.

```
HTTP/1.1 200 OK
Content-Type: application/json
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

You can specify  `offset` and  `limit` parameters in order to fetch specific
group of channels. In that case, your request should look like:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels?offset=0&limit=5
```

If you don't provide them, default values will be used instead: 0 for `offset`,
and 10 for `limit`. Note that `limit` cannot be set to values greater than 100. Providing
invalid values will be considered malformed request.

### Removing channels

In order to remove specific channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

## Access control

Channel can be observed as a communication group of things. Only things that
are connected to the channel can send and receive messages from other things
in this channel. things that are not connected to this channel are not allowed
to communicate over it.

Only user, who is the owner of a channel and of the things, can connect the
things to the channel (which is equivalent of giving permissions to these things
to communicate over given communication group).

To connect thing to the channel you should send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X PUT -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things/<thing_id>
```

You can observe which things are connected to specific channel:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>
```

You should receive response with the lists of connected things in `connected` field
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
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things/<thing_id>
```