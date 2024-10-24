# Magistrala CLI

## Build

From the project root:

```bash
make cli
```

## Usage

### Service

#### Get Magistrala Services Health Check

```bash
magistrala-cli health <service>
```

### Users management

#### Create User

```bash
magistrala-cli users create <user_name> <user_email> <user_password>

magistrala-cli users create <user_name> <user_email> <user_password> <user_token>
```

#### Login User

```bash
magistrala-cli users token <user_email> <user_password>
```

#### Get User

```bash
magistrala-cli users get <user_id> <user_token>
```

#### Get Users

```bash
magistrala-cli users get all <user_token>
```

#### Update User Metadata

```bash
magistrala-cli users update <user_id> '{"name":"value1", "metadata":{"value2": "value3"}}' <user_token>
```

#### Update User Password

```bash
magistrala-cli users password <old_password> <password> <user_token>
```

#### Enable User

```bash
magistrala-cli users enable <user_id> <user_token>
```

#### Disable User

```bash
magistrala-cli users disable <user_id> <user_token>
```

### System Provisioning

#### Create Client

```bash
magistrala-cli clients create '{"name":"myClient"}' <user_token>
```

#### Create Client with metadata

```bash
magistrala-cli clients create '{"name":"myClient", "metadata": {"key1":"value1"}}' <user_token>
```

#### Bulk Provision Clients

```bash
magistrala-cli provision clients <file> <user_token>
```

- `file` - A CSV or JSON file containing client names (must have extension `.csv` or `.json`)
- `user_token` - A valid user auth token for the current system

An example CSV file might be:

```csv
client1,
client2,
client3,
```

in which the first column is the client's name.

A comparable JSON file would be

```json
[
  {
    "name": "<client1_name>",
    "status": "enabled"
  },
  {
    "name": "<client2_name>",
    "status": "disabled"
  },
  {
    "name": "<client3_name>",
    "status": "enabled",
    "credentials": {
      "identity": "<client3_identity>",
      "secret": "<client3_secret>"
    }
  }
]
```

With JSON you can be able to specify more fields of the channels you want to create

#### Update Client

```bash
magistrala-cli clients update <client_id> '{"name":"value1", "metadata":{"key1": "value2"}}' <user_token>
```

#### Identify Client

```bash
magistrala-cli clients identify <client_key>
```

#### Enable Client

```bash
magistrala-cli clients enable <client_id> <user_token>
```

#### Disable Client

```bash
magistrala-cli clients disable <client_id> <user_token>
```

#### Get Client

```bash
magistrala-cli clients get <client_id> <user_token>
```

#### Get Clients

```bash
magistrala-cli clients get all <user_token>
```

#### Get a subset list of provisioned Clients

```bash
magistrala-cli clients get all --offset=1 --limit=5 <user_token>
```

#### Create Channel

```bash
magistrala-cli channels create '{"name":"myChannel"}' <user_token>
```

#### Bulk Provision Channels

```bash
magistrala-cli provision channels <file> <user_token>
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
magistrala-cli channels update '{"id":"<channel_id>","name":"myNewName"}' <user_token>
```

#### Enable Channel

```bash
magistrala-cli channels enable <channel_id> <user_token>
```

#### Disable Channel

```bash
magistrala-cli channels disable <channel_id> <user_token>
```

#### Get Channel

```bash
magistrala-cli channels get <channel_id> <user_token>
```

#### Get Channels

```bash
magistrala-cli channels get all <user_token>
```

#### Get a subset list of provisioned Channels

```bash
magistrala-cli channels get all --offset=1 --limit=5 <user_token>
```

### Access control

#### Connect Client to Channel

```bash
magistrala-cli clients connect <client_id> <channel_id> <user_token>
```

#### Bulk Connect Clients to Channels

```bash
magistrala-cli provision connect <file> <user_token>
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
magistrala-cli clients disconnect <client_id> <channel_id> <user_token>
```

#### Get a subset list of Channels connected to Client

```bash
magistrala-cli clients connections <client_id> <user_token>
```

#### Get a subset list of Clients connected to Channel

```bash
magistrala-cli channels connections <channel_id> <user_token>
```

### Messaging

#### Send a message over HTTP

```bash
magistrala-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <client_secret>
```

#### Read messages over HTTP

```bash
magistrala-cli messages read <channel_id> <user_token> -R <reader_url>
```

### Bootstrap

#### Add configuration

```bash
magistrala-cli bootstrap create '{"external_id": "myExtID", "external_key": "myExtKey", "name": "myName", "content": "myContent"}' <user_token> -b <bootstrap-url>
```

#### View configuration

```bash
magistrala-cli bootstrap get <client_id> <user_token> -b <bootstrap-url>
```

#### Update configuration

```bash
magistrala-cli bootstrap update '{"client_id":"<client_id>", "name": "newName", "content": "newContent"}' <user_token> -b <bootstrap-url>
```

#### Remove configuration

```bash
magistrala-cli bootstrap remove <client_id> <user_token> -b <bootstrap-url>
```

#### Bootstrap configuration

```bash
magistrala-cli bootstrap bootstrap <external_id> <external_key> -b <bootstrap-url>
```

### Groups

#### Create Group

```bash
magistrala-cli groups create '{"name":"<group_name>","description":"<description>","parentID":"<parent_id>","metadata":"<metadata>"}' <user_token>
```

#### Get Group

```bash
magistrala-cli groups get <group_id> <user_token>
```

#### Get Groups

```bash
magistrala-cli groups get all <user_token>
```

#### Get Group Members

```bash
magistrala-cli groups members <group_id> <user_token>
```

#### Get Memberships

```bash
magistrala-cli groups membership <member_id> <user_token>
```

#### Assign Members to Group

```bash
magistrala-cli groups assign <member_ids> <member_type> <group_id> <user_token>
```

#### Unassign Members to Group

```bash
magistrala-cli groups unassign <member_ids> <group_id>  <user_token>
```

#### Enable Group

```bash
magistrala-cli groups enable <group_id> <user_token>
```

#### Disable Group

```bash
magistrala-cli groups disable <group_id> <user_token>
```
