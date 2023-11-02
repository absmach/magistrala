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

#### Create Thing

```bash
magistrala-cli things create '{"name":"myThing"}' <user_token>
```

#### Create Thing with metadata

```bash
magistrala-cli things create '{"name":"myThing", "metadata": {"key1":"value1"}}' <user_token>
```

#### Bulk Provision Things

```bash
magistrala-cli provision things <file> <user_token>
```

- `file` - A CSV or JSON file containing thing names (must have extension `.csv` or `.json`)
- `user_token` - A valid user auth token for the current system

An example CSV file might be:

```csv
thing1,
thing2,
thing3,
```

in which the first column is the thing's name.

A comparable JSON file would be

```json
[
  {
    "name": "<thing1_name>",
    "status": "enabled"
  },
  {
    "name": "<thing2_name>",
    "status": "disabled"
  },
  {
    "name": "<thing3_name>",
    "status": "enabled",
    "credentials": {
      "identity": "<thing3_identity>",
      "secret": "<thing3_secret>"
    }
  }
]
```

With JSON you can be able to specify more fields of the channels you want to create

#### Update Thing

```bash
magistrala-cli things update <thing_id> '{"name":"value1", "metadata":{"key1": "value2"}}' <user_token>
```

#### Identify Thing

```bash
magistrala-cli things identify <thing_key>
```

#### Enable Thing

```bash
magistrala-cli things enable <thing_id> <user_token>
```

#### Disable Thing

```bash
magistrala-cli things disable <thing_id> <user_token>
```

#### Get Thing

```bash
magistrala-cli things get <thing_id> <user_token>
```

#### Get Things

```bash
magistrala-cli things get all <user_token>
```

#### Get a subset list of provisioned Things

```bash
magistrala-cli things get all --offset=1 --limit=5 <user_token>
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

#### Connect Thing to Channel

```bash
magistrala-cli things connect <thing_id> <channel_id> <user_token>
```

#### Bulk Connect Things to Channels

```bash
magistrala-cli provision connect <file> <user_token>
```

- `file` - A CSV or JSON file containing thing and channel ids (must have extension `.csv` or `.json`)
- `user_token` - A valid user auth token for the current system

An example CSV file might be

```csv
<thing_id1>,<channel_id1>
<thing_id2>,<channel_id2>
```

in which the first column is thing IDs and the second column is channel IDs. A connection will be created for each thing to each channel. This example would result in 4 connections being created.

A comparable JSON file would be

```json
{
  "client_ids": ["<thing_id1>", "<thing_id2>"],
  "group_ids": ["<channel_id1>", "<channel_id2>"]
}
```

#### Disconnect Thing from Channel

```bash
magistrala-cli things disconnect <thing_id> <channel_id> <user_token>
```

#### Get a subset list of Channels connected to Thing

```bash
magistrala-cli things connections <thing_id> <user_token>
```

#### Get a subset list of Things connected to Channel

```bash
magistrala-cli channels connections <channel_id> <user_token>
```

### Messaging

#### Send a message over HTTP

```bash
magistrala-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_secret>
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
magistrala-cli bootstrap get <thing_id> <user_token> -b <bootstrap-url>
```

#### Update configuration

```bash
magistrala-cli bootstrap update '{"thing_id":"<thing_id>", "name": "newName", "content": "newContent"}' <user_token> -b <bootstrap-url>
```

#### Remove configuration

```bash
magistrala-cli bootstrap remove <thing_id> <user_token> -b <bootstrap-url>
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
