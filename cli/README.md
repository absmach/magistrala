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
mainflux-cli users create <user_name> <user_email> <user_password>

mainflux-cli users create <user_name> <user_email> <user_password> <user_token>
```

#### Login User

```bash
mainflux-cli users token <user_email> <user_password>
```

#### Get User

```bash
mainflux-cli users get <user_id> <user_token>
```

#### Get Users

```bash
mainflux-cli users get all <user_token>
```

#### Update User Metadata

```bash
mainflux-cli users update <user_id> '{"name":"value1", "metadata":{"value2": "value3"}}' <user_token>
```

#### Update User Password

```bash
mainflux-cli users password <old_password> <password> <user_token>
```

#### Enable User

```bash
mainflux-cli users enable <user_id> <user_token>
```

#### Disable User

```bash
mainflux-cli users disable <user_id> <user_token>
```

### System Provisioning

#### Create Thing

```bash
mainflux-cli things create '{"name":"myThing"}' <user_token>
```

#### Create Thing with metadata

```bash
mainflux-cli things create '{"name":"myThing", "metadata": {"key1":"value1"}}' <user_token>
```

#### Bulk Provision Things

```bash
mainflux-cli provision things <file> <user_token>
```

* `file` - A CSV or JSON file containing thing names (must have extension `.csv` or `.json`)
* `user_token` - A valid user auth token for the current system

An example CSV file might be:

```csv
thing1,
thing2,
thing3,
```

in which the first column is the thing's name.

A comparable JSON file would be

```json
[{
        "name": "<thing1_name>",
        "status": "enabled"
    },
    {
        "name": "<thing2_name>",
        "status": "disabled"
    }, {

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
mainflux-cli things update <thing_id> '{"name":"value1", "metadata":{"key1": "value2"}}' <user_token>
```

#### Identify Thing

```bash
mainflux-cli things identify <thing_key>
```

#### Enable Thing

```bash
mainflux-cli things enable <thing_id> <user_token>
```

#### Disable Thing

```bash
mainflux-cli things disable <thing_id> <user_token>
```

#### Get Thing

```bash
mainflux-cli things get <thing_id> <user_token>
```

#### Get Things

```bash
mainflux-cli things get all <user_token>
```

#### Get a subset list of provisioned Things

```bash
mainflux-cli things get all --offset=1 --limit=5 <user_token>
```

#### Create Channel

```bash
mainflux-cli channels create '{"name":"myChannel"}' <user_token>
```

#### Bulk Provision Channels

```bash
mainflux-cli provision channels <file> <user_token>
```

* `file` - A CSV or JSON file containing channel names (must have extension `.csv` or `.json`)
* `user_token` - A valid user auth token for the current system

An example CSV file might be:

```csv
<channel1_name>,
<channel2_name>,
<channel3_name>,
```

in which the first column is channel names.

A comparable JSON file would be

```json
[{
        "name": "<channel1_name>",
        "description": "<channel1_description>",
        "status": "enabled"
    },
    {
        "name": "<channel2_name>",
        "description": "<channel2_description>",
        "status": "disabled"
    }, {

        "name": "<channel3_name>",
        "description": "<channel3_description>",
        "status": "enabled"
    }
]
```

With JSON you can be able to specify more fields of the channels you want to create

#### Update Channel

```bash
mainflux-cli channels update '{"id":"<channel_id>","name":"myNewName"}' <user_token>
```

#### Enable Channel

```bash
mainflux-cli channels enable <channel_id> <user_token>
```

#### Disable Channel

```bash
mainflux-cli channels disable <channel_id> <user_token>
```

#### Get Channel

```bash
mainflux-cli channels get <channel_id> <user_token>
```

#### Get Channels

```bash
mainflux-cli channels get all <user_token>
```

#### Get a subset list of provisioned Channels

```bash
mainflux-cli channels get all --offset=1 --limit=5 <user_token>
```

### Access control

#### Connect Thing to Channel

```bash
mainflux-cli things connect <thing_id> <channel_id> <user_token>
```

#### Bulk Connect Things to Channels

```bash
mainflux-cli provision connect <file> <user_token>
```

* `file` - A CSV or JSON file containing thing and channel ids (must have extension `.csv` or `.json`)
* `user_token` - A valid user auth token for the current system

An example CSV file might be

```csv
<thing_id1>,<channel_id1>
<thing_id2>,<channel_id2>
```

in which the first column is thing IDs and the second column is channel IDs.  A connection will be created for each thing to each channel.  This example would result in 4 connections being created.

A comparable JSON file would be

```json
{
    "client_ids": [
        "<thing_id1>",
        "<thing_id2>"
    ],
    "group_ids": [
        "<channel_id1>",
        "<channel_id2>"
    ]
}
```

#### Disconnect Thing from Channel

```bash
mainflux-cli things disconnect <thing_id> <channel_id> <user_token>
```

#### Get a subset list of Channels connected to Thing

```bash
mainflux-cli things connections <thing_id> <user_token>
```

#### Get a subset list of Things connected to Channel

```bash
mainflux-cli channels connections <channel_id> <user_token>
```

### Messaging

#### Send a message over HTTP

```bash
mainflux-cli messages send <channel_id> '[{"bn":"Dev1","n":"temp","v":20}, {"n":"hum","v":40}, {"bn":"Dev2", "n":"temp","v":20}, {"n":"hum","v":40}]' <thing_secret>
```

#### Read messages over HTTP

```bash
mainflux-cli messages read <channel_id> <user_token> -R <reader_url>
```

### Bootstrap

#### Add configuration

```bash
mainflux-cli bootstrap create '{"external_id": "myExtID", "external_key": "myExtKey", "name": "myName", "content": "myContent"}' <user_token> -b <bootstrap-url>
```

#### View configuration

```bash
mainflux-cli bootstrap get <thing_id> <user_token> -b <bootstrap-url>
```

#### Update configuration

```bash
mainflux-cli bootstrap update '{"thing_id":"<thing_id>", "name": "newName", "content": "newContent"}' <user_token> -b <bootstrap-url>
```

#### Remove configuration

```bash
mainflux-cli bootstrap remove <thing_id> <user_token> -b <bootstrap-url>
```

#### Bootstrap configuration

```bash
mainflux-cli bootstrap bootstrap <external_id> <external_key> -b <bootstrap-url>
```

### Groups

#### Create Group

```bash
mainflux-cli groups create '{"name":"<group_name>","description":"<description>","parentID":"<parent_id>","metadata":"<metadata>"}' <user_token>
```

#### Get Group

```bash
mainflux-cli groups get <group_id> <user_token>
```

#### Get Groups

```bash
mainflux-cli groups get all <user_token>
```

#### Get Group Members

```bash
mainflux-cli groups members <group_id> <user_token>
```

#### Get Memberships

```bash
mainflux-cli groups membership <member_id> <user_token>
```

#### Assign Members to Group

```bash
mainflux-cli groups assign <member_ids> <member_type> <group_id> <user_token>
```

#### Unassign Members to Group

```bash
mainflux-cli groups unassign <member_ids> <group_id>  <user_token>
```

#### Enable Group

```bash
mainflux-cli groups enable <group_id> <user_token>
```

#### Disable Group

```bash
mainflux-cli groups disable <group_id> <user_token>
```
