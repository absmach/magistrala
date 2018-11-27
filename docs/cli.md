## CLI

Mainflux CLI makes it easy to manage users, things, channels and messages.

CLI can be downloaded as separate asset from [project realeses](https://github.com/mainflux/mainflux/releases) or it can be built with `GNU Make` tool:

```
make cli
```

which will build `mainflux-cli` in `<project_root>/build` folder.

Executing `build/mainflux-cli` without any arguments will output help with all available commands and flags:

```
Usage:
  mainflux-cli [command]

Available Commands:
  channels    Manipulation with channels
  help        Help about any command
  msg         Send or retrieve messages
  things      things <options>
  users       users create/token <email> <password>
  version     Get version of Mainflux Things Service

Flags:
  -c, --content-type string    Mainflux message content type (default "application/senml+json")
  -h, --help                   help for mainflux-cli
  -a, --http-prefix string     Mainflux http adapter prefix (default "http")
  -i, --insecure               Do not check for TLS cert
  -l, --limit uint             limit query parameter (default 100)
  -m, --mainflux-url string    Mainflux host URL (default "http://localhost")
  -o, --offset uint            offset query parameter
  -t, --things-prefix string   Mainflux things service prefix
  -u, --users-prefix string    Mainflux users service prefix

Use "mainflux-cli [command] --help" for more information about a command.
```

You can execute each command with `-h` flag for more information about that command, e.g.

```
./mainflux-cli channels -h
```

will get you usage info:

```
Manipulation with channels: create, delete or update channels

Usage:
  mainflux-cli channels [flags]
  mainflux-cli channels [command]

Available Commands:
  create      create <JSON_channel> <user_auth_token>
  delete      delete <channel_id> <user_auth_token>
  get         get all/<channel_id> <user_auth_token>
  update      update <JSON_string> <user_auth_token>

```

### Service
#### Get the service verison

```
./mainflux-cli version
```

### User management
#### Create User

```
./mainflux-cli users create john.doe@email.com password
```

#### Login User

```
./mainflux-cli users token john.doe@email.com password
```

### System Provisioning
#### Provision Device

```
./mainflux-cli things create '{"type":"device", "name":"nyDevice"}' <user_auth_token>
```

#### Provision Application

```
./mainflux-cli things create '{"type":"app", "name":"nyDevice"}' <user_auth_token>
```

#### Retrieve All Things

```
./mainflux-cli things get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Thing By ID

```
./mainflux-cli things get <thing_id> <user_auth_token>
```

#### Remove Thing

```
./mainflux-cli things delete <thing_id> <user_auth_token>
```

#### Provision Channel

```
./mainflux-cli channels create '{"name":"nyChannel"}' <user_auth_token>
```

#### Retrieve All Channels

```
./mainflux-cli channels get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrievie Channel By ID

```
./mainflux-cli channels get <channel_id> <user_auth_token>
```

#### Remove Channel

```
./mainflux-cli channels delete <channel_id> <user_auth_token>
```

### Access control
#### Connect Thing to a Channel

```
./mainflux-cli things connect <thing_id> <channel_id> <user_auth_token>
```

#### Disconnect Things from a Channel

```
./mainflux-cli things disconnect <thing_id> <channel_id> <user_auth_token>
```

### Messaging
#### Send a message over HTTP

```
./mainflux-cli msg send <channel_id> '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]' <thing_auth_token>
```


