# Mainflux CLI
## Build
From the project root:
```
make cli
```

## Usage
### Service
#### Get the version of Mainflux services
```
mainflux-cli version
```

### Users management
#### Create User
```
mainflux-cli users create john.doe@email.com password
```

#### Login User
```
mainflux-cli users token john.doe@email.com password
```

### System Provisioning
#### Create Thing (type Device)
```
mainflux-cli things create '{"name":"myDevice"}' <user_auth_token>
```

#### Create Thing (type Application)
```
mainflux-cli things create '{"name":"myDevice"}' <user_auth_token>
```

#### Update Thing
```
mainflux-cli things update '{"id":"<thing_id>", "name":"myNewName"}' <user_auth_token>
```

#### Remove Thing
```
mainflux-cli things delete <thing_id> <user_auth_token>
```

#### Retrieve a subset list of provisioned Things
```
mainflux-cli things get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Thing By ID
```
mainflux-cli things get <thing_id> <user_auth_token>
```

#### Create Channel
```
mainflux-cli channels create '{"name":"myChannel"}' <user_auth_token>
```

#### Update Channel
```
mainflux-cli channels update '{"id":"<channel_id>","name":"myNewName"}' <user_auth_token>

```
#### Remove Channel
```
mainflux-cli channels delete <channel_id> <user_auth_token>
```

#### Retrieve a subset list of provisioned Channels
```
mainflux-cli channels get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Channel By ID
```
mainflux-cli channels get <channel_id> <user_auth_token>
```

### Access control
#### Connect Thing to Channel
```
mainflux-cli things connect <thing_id> <channel_id> <user_auth_token>
```

#### Disconnect Thing from Channel
```
mainflux-cli things disconnect <thing_id> <channel_id> <user_auth_token>

```

#### Retrieve a subset list of Channels connected to Thing
```
mainflux-cli things connections <thing_id> <user_auth_token>
```

#### Retrieve a subset list of Things connected to Channel
```
mainflux-cli channels connections <channel_id> <user_auth_token>
```

### Messaging
#### Send a message over HTTP
```
mainflux-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_auth_token>
```
