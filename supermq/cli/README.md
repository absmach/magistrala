# SuperMQ CLI

## Build

From the project root:

```bash
make cli
```

## Usage

### Service

#### Get SuperMQ Services Health Check

```bash
supermq-cli health <service>
```

### Users management

#### Create User

```bash
supermq-cli users create <user_name> <user_email> <user_password>

supermq-cli users create <user_name> <user_email> <user_password> <user_token>
```

#### Login User

```bash
supermq-cli users token <user_email> <user_password>
```

#### Get User

```bash
supermq-cli users get <user_id> <user_token>
```

#### Get Users

```bash
supermq-cli users get all <user_token>
```

#### Update User Metadata

```bash
supermq-cli users update <user_id> '{"name":"value1", "metadata":{"value2": "value3"}}' <user_token>
```

#### Update User Password

```bash
supermq-cli users password <old_password> <password> <user_token>
```

#### Enable User

```bash
supermq-cli users enable <user_id> <user_token>
```

#### Disable User

```bash
supermq-cli users disable <user_id> <user_token>
```

### System Provisioning

#### Create Client

```bash
supermq-cli clients create '{"name":"myClient"}' <user_token>
```

#### Create Client with metadata

```bash
supermq-cli clients create '{"name":"myClient", "metadata": {"key1":"value1"}}' <user_token>
```

#### Update Client

```bash
supermq-cli clients update <client_id> '{"name":"value1", "metadata":{"key1": "value2"}}' <user_token>
```

#### Identify Client

```bash
supermq-cli clients identify <client_key>
```

#### Enable Client

```bash
supermq-cli clients enable <client_id> <user_token>
```

#### Disable Client

```bash
supermq-cli clients disable <client_id> <user_token>
```

#### Get Client

```bash
supermq-cli clients get <client_id> <user_token>
```

#### Get Clients

```bash
supermq-cli clients get all <user_token>
```

#### Get a subset list of provisioned Clients

```bash
supermq-cli clients get all --offset=1 --limit=5 <user_token>
```

#### Create Channel

```bash
supermq-cli channels create '{"name":"myChannel"}' <user_token>
```

#### Bulk Provision Channels

```bash
supermq-cli provision channels <file> <user_token>
```

- `file` - A CSV or JSON file containing channel names (must have extension `.csv` or `.json`)
- `user_token` - A valid user auth token for the current system

An example CSV file might be:

```csv
<channel1_name>,
<channel2_name>,
<channel3_name>,
```

in which the first column is channel names.

A comparable JSON file would be

```json
[
  {
    "name": "<channel1_name>",
    "description": "<channel1_description>",
    "status": "enabled"
  },
  {
    "name": "<channel2_name>",
    "description": "<channel2_description>",
    "status": "disabled"
  },
  {
    "name": "<channel3_name>",
    "description": "<channel3_description>",
    "status": "enabled"
  }
]
```

With JSON you can be able to specify more fields of the channels you want to create

#### Update Channel

```bash
supermq-cli channels update '{"id":"<channel_id>","name":"myNewName"}' <user_token>
```

#### Enable Channel

```bash
supermq-cli channels enable <channel_id> <user_token>
```

#### Disable Channel

```bash
supermq-cli channels disable <channel_id> <user_token>
```

#### Get Channel

```bash
supermq-cli channels get <channel_id> <user_token>
```

#### Get Channels

```bash
supermq-cli channels get all <user_token>
```

#### Get a subset list of provisioned Channels

```bash
supermq-cli channels get all --offset=1 --limit=5 <user_token>
```

### Access control

#### Connect Client to Channel

```bash
supermq-cli clients connect <client_id> <channel_id> <user_token>
```

#### Bulk Connect Clients to Channels

```bash
supermq-cli provision connect <file> <user_token>
```

- `file` - A CSV or JSON file containing client and channel ids (must have extension `.csv` or `.json`)
- `user_token` - A valid user auth token for the current system

An example CSV file might be

```csv
<client_id1>,<channel_id1>
<client_id2>,<channel_id2>
```

in which the first column is client IDs and the second column is channel IDs. A connection will be created for each client to each channel. This example would result in 4 connections being created.

A comparable JSON file would be

```json
{
  "client_ids": ["<client_id1>", "<client_id2>"],
  "group_ids": ["<channel_id1>", "<channel_id2>"]
}
```

#### Disconnect Client from Channel

```bash
supermq-cli clients disconnect <client_id> <channel_id> <user_token>
```

#### Get a subset list of Channels connected to Client

```bash
supermq-cli clients connections <client_id> <user_token>
```

#### Get a subset list of Clients connected to Channel

```bash
supermq-cli channels connections <channel_id> <user_token>
```

### Messaging

#### Send a message over HTTP

```bash
supermq-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <client_secret>
```

#### Read messages over HTTP

```bash
supermq-cli messages read <channel_id> <user_token> -R <reader_url>
```

### Groups

#### Create Group

```bash
supermq-cli groups create '{"name":"<group_name>","description":"<description>","parentID":"<parent_id>","metadata":"<metadata>"}' <user_token>
```

#### Get Group

```bash
supermq-cli groups get <group_id> <user_token>
```

#### Get Groups

```bash
supermq-cli groups get all <user_token>
```

#### Get Group Members

```bash
supermq-cli groups members <group_id> <user_token>
```

#### Get Memberships

```bash
supermq-cli groups membership <member_id> <user_token>
```

#### Assign Members to Group

```bash
supermq-cli groups assign <member_ids> <member_type> <group_id> <user_token>
```

#### Unassign Members to Group

```bash
supermq-cli groups unassign <member_ids> <group_id>  <user_token>
```

#### Enable Group

```bash
supermq-cli groups enable <group_id> <user_token>
```

#### Disable Group

```bash
supermq-cli groups disable <group_id> <user_token>
```
