# Mainflux CLI
## Build
From the project root:
```
make cli
```

## Usage
### Service
#### Get the service verison
```
mainflux-cli version
```

### User management
#### Create User
```
mainflux-cli users create john.doe@email.com password
```

#### Login User
```
mainflux-cli users token john.doe@email.com password
```

### System Provisioning
#### Provision Device
```
mainflux-cli things create '{"type":"device", "name":"nyDevice"}' <user_auth_token>
```

#### Provision Application
```
mainflux-cli things create '{"type":"app", "name":"nyDevice"}' <user_auth_token>
```

#### Retrieve All Things
```
mainflux-cli things get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Thing By ID
```
mainflux-cli things get <thing_id> <user_auth_token>
```

#### Remove Thing
```
mainflux-cli things delete <thing_id> <user_auth_token>
```

#### Provision Channel
```
mainflux-cli channels create '{"name":"nyChannel"}' <user_auth_token>
```

#### Retrieve All Channels
```
mainflux-cli channels get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrievie Channel By ID
```
mainflux-cli channels get <channel_id> <user_auth_token>
```

#### Remove Channel
```
mainflux-cli channels delete <channel_id> <user_auth_token>
```

### Access control
#### Connect Thing to a Channel
```
mainflux-cli things connect <thing_id> <channel_id> <user_auth_token>
```

#### Disconnect Things from a Channel
```
mainflux-cli things disconnect <thing_id> <channel_id> <user_auth_token>
```

### Messaging
#### Send a message over HTTP
```
mainflux-cli msg send <channel_id> '[{"bn":"some-base-name:","bt":1.276020076001e+09, "bu":"A","bver":5, "n":"voltage","u":"V","v":120.1}, {"n":"current","t":-5,"v":1.2}, {"n":"current","t":-4,"v":1.3}]' <thing_auth_token>
```
