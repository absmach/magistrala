Provisioning is a process of configuration of an IoT platform in which system operator creates and sets-up different entities
used in the platform - users, channels and things.

## Users management

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

### Provisioning things

Things are provisioned by executing request `POST /things` with a JSON payload.
Note that you will also need `user_auth_token` in order to provision things
that belong to this particular user.

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: <user_auth_token>" https://localhost/things -d '{"name":"weio"}'
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
  "total": 2,
  "offset": 0,
  "limit": 10,
  "things": [
    {
      "id": "81380742-7116-4f6f-9800-14fe464f6773",
      "name": "weio",
      "key": "7aa91f7a-cbea-4fed-b427-07e029577590"
    },
    {
      "id": "cb63f852-2d48-44f0-a0cf-e450496c6c92",
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

You can specify `name` and/or `metadata` parameters in order to fetch specific
group of things. When specifiying metadata you can specify just a part of the metadata json you want to match

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things?offset=0&limit=5&metadata={"serial":"123456"}
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
  "total": 1,
  "offset": 0,
  "limit": 10,
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
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things
```

Response that you'll get should look like this:

```
{
  "total": 2,
  "offset": 0,
  "limit": 10,
  "things": [
    {
      "id": "3ffb3880-d1e6-4edd-acd9-4294d013f35b",
      "name": "d0",
      "key": "b1996995-237a-4552-94b2-83ec2e92a040",
      "metadata": "{}"
    },
    {
      "id": "94d166d6-6477-43dc-93b7-5c3707dbef1e",
      "name": "d1",
      "key": "e4588a68-6028-4740-9f12-c356796aebe8",
      "metadata": "{}"
    }
  ]
}
```

You can also observe to which channels is specified thing connected:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: <user_auth_token>" https://localhost/things/<thing_id>/channels
```

Response that you'll get should look like this:

```
{
  "total": 2,
  "offset": 0,
  "limit": 10,
  "channels": [
    {
      "id": "5e62eb13-2695-4860-8d87-85b8a2f80fd4",
      "name": "c1",
      "metadata": "{}"
    },
    {
      "id": "c4b5e19a-7ffe-4172-b2c5-c8b9d570a165",
      "name": "c0",
      "metadata":"{}"
    }
  ]
}
```

If you want to disconnect your thing from the channel, send following request:

```
curl -s -S -i --cacert docker/ssl/certs/mainflux-server.crt --insecure -X DELETE -H "Authorization: <user_auth_token>" https://localhost/channels/<channel_id>/things/<thing_id>
```
