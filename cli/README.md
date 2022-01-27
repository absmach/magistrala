# Mainflux CLI
## Build
From the project root:
```bash
make cli
```

## Usage
### Service
#### Get Mainflux Things services Health Check
```bash
mainflux-cli health
```

### Users management
#### Create User
```bash
mainflux-cli users create <user_email> <user_password>
```

#### Login User
```bash
mainflux-cli users token <user_email> <user_password>
```

#### Retrieve User
```bash
mainflux-cli users get <user_auth_token>
```

#### Update User Metadata
```bash
mainflux-cli users update '{"key1":"value1", "key2":"value2"}' <user_auth_token>
```

#### Update User Password
```bash
mainflux-cli users password <old_password> <password> <user_auth_token>
```

### System Provisioning
#### Create Thing
```bash
mainflux-cli things create '{"name":"myThing"}' <user_auth_token>
```

#### Create Thing with metadata
```bash
mainflux-cli things create '{"name":"myThing", "metadata": {\"key1\":\"value1\"}}' <user_auth_token>
```

#### Bulk Provision Things
```bash
mainflux-cli provision things <file> <user_auth_token>
```

* `file` - A CSV or JSON file containing things
* `user_auth_token` - A valid user auth token for the current system

#### Update Thing
```bash
mainflux-cli things update '{"id":"<thing_id>", "name":"myNewName"}' <user_auth_token>
```

#### Remove Thing
```bash
mainflux-cli things delete <thing_id> <user_auth_token>
```

#### Retrieve a subset list of provisioned Things
```bash
mainflux-cli things get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Thing By ID
```bash
mainflux-cli things get <thing_id> <user_auth_token>
```

#### Create Channel
```bash
mainflux-cli channels create '{"name":"myChannel"}' <user_auth_token>
```

#### Bulk Provision Channels
```bash
mainflux-cli provision channels <file> <user_auth_token>
```

* `file` - A CSV or JSON file containing channels
* `user_auth_token` - A valid user auth token for the current system

#### Update Channel
```bash
mainflux-cli channels update '{"id":"<channel_id>","name":"myNewName"}' <user_auth_token>
```

#### Remove Channel
```bash
mainflux-cli channels delete <channel_id> <user_auth_token>
```

#### Retrieve a subset list of provisioned Channels
```bash
mainflux-cli channels get all --offset=1 --limit=5 <user_auth_token>
```

#### Retrieve Channel By ID
```bash
mainflux-cli channels get <channel_id> <user_auth_token>
```

### Access control
#### Connect Thing to Channel
```bash
mainflux-cli things connect <thing_id> <channel_id> <user_auth_token>
```

#### Bulk Connect Things to Channels
```bash
mainflux-cli provision connect <file> <user_auth_token>
```

* `file` - A CSV or JSON file containing thing and channel ids
* `user_auth_token` - A valid user auth token for the current system

An example CSV file might be

```csv
<thing_id>,<channel_id>
<thing_id>,<channel_id>
```

in which the first column is thing IDs and the second column is channel IDs.  A connection will be created for each thing to each channel.  This example would result in 4 connections being created.

A comparable JSON file would be

```json
{
    "thing_ids": [
        "<thing_id>",
        "<thing_id>"
    ],
    "channel_ids": [
        "<channel_id>",
        "<channel_id>"
    ]
}
```

#### Disconnect Thing from Channel
```bash
mainflux-cli things disconnect <thing_id> <channel_id> <user_auth_token>

```

#### Retrieve a subset list of Channels connected to Thing
```bash
mainflux-cli things connections <thing_id> <user_auth_token>
```

#### Retrieve a subset list of Things connected to Channel
```bash
mainflux-cli channels connections <channel_id> <user_auth_token>
```


### Messaging
#### Send a message over HTTP
```bash
mainflux-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_auth_token>
```

#### Read messages over HTTP
```bash
mainflux-cli messages read <channel_id> <thing_auth_token>
```

### Bootstrap

#### Add configuration
```bash
mainflux-cli bootstrap add '{"external_id": "myExtID", "external_key": "myExtKey", "name": "myName", "content": "myContent"}' <user_auth_token>
```

#### View configuration
```bash
mainflux-cli bootstrap view <thing_id> <user_auth_token>
```

#### Update configuration
```bash
mainflux-cli bootstrap update '{"MFThing":"<thing_id>", "name": "newName", "content": "newContent"}' <user_auth_token>
```

#### Remove configuration
```bash
mainflux-cli bootstrap remove <thing_id> <user_auth_token>
```

#### Bootstrap configuration
```bash
mainflux-cli bootstrap bootstrap <external_id> <external_key>
```

### Groups
#### Create new group
```bash
mainflux-cli groups create '{"name":"<group_name>","parent_id":"<parent_group_id>","description":"<description>","metadata":{"key":"value",...}}' <user_auth_token>
```
#### Delete group
```bash
mainflux-cli groups delete <group_id> <user_auth_token>
```
#### Get group with id
```bash
mainflux-cli groups get <group_id> <user_auth_token>
```
#### List all groups
```bash
mainflux-cli groups get all <user_auth_token>
```
#### List children groups for some group
```bash
mainflux-cli groups get children <parent_group_id> <user_auth_token>
```
#### Assign user to a group
```bash
mainflux-cli groups assign <user_id> <group_id> <user_auth_token>
```
#### Unassign user from group
```bash
mainflux-cli groups unassign <user_id> <group_id> <user_auth_token>
```
#### List users for a group
```bash
mainflux-cli groups members <group_id> <user_auth_token>
```
#### List groups that user belongs to
```bash
mainflux-cli groups membership <user_id> <user_auth_token>
```

### Keys management
#### Issue a new Key
```bash
mainflux-cli keys issue <duration> <user_auth_token>
```
#### Remove API key from database
```bash
mainflux-cli keys revoke <key_id> <user_auth_token>
```
#### Retrieve API key with given id
```bash
mainflux-cli keys retrieve <key_id> <user_auth_token>
```
