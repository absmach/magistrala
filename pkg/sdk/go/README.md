# Magistrala Go SDK

Go SDK, a Go driver for Magistrala HTTP API.

Does both system administration (provisioning) and messaging.

## Installation

Import `"github.com/absmach/magistrala/sdk/go"` in your Go package.

````
import "github.com/absmach/magistrala/pkg/sdk/go"```

Then call SDK Go functions to interact with the system.

## API Reference

```go
FUNCTIONS

func NewMgxSDK(host, port string, tls bool) *MgxSDK

func (sdk *MgxSDK) Channel(id, token string) (things.Channel, error)
    Channel - gets channel by ID

func (sdk *MgxSDK) Channels(token string) ([]things.Channel, error)
    Channels - gets all channels

func (sdk *MgxSDK) Connect(struct{[]string, []string}, token string) error
    Connect - connect things to channels

func (sdk *MgxSDK) CreateChannel(data, token string) (string, error)
    CreateChannel - creates new channel and generates UUID

func (sdk *MgxSDK) CreateThing(data, token string) (string, error)
    CreateThing - creates new thing and generates thing UUID

func (sdk *MgxSDK) CreateToken(user, pwd string) (string, error)
    CreateToken - create user token

func (sdk *MgxSDK) CreateUser(user, pwd string) error
    CreateUser - create user

func (sdk *MgxSDK) User(pwd string) (user, error)
    User - gets user

func (sdk *MgxSDK) UpdateUser(user, pwd string) error
    UpdateUser - update user

func (sdk *MgxSDK) UpdatePassword(user, pwd string) error
    UpdatePassword - update user password

func (sdk *MgxSDK) DeleteChannel(id, token string) error
    DeleteChannel - removes channel

func (sdk *MgxSDK) DeleteThing(id, token string) error
    DeleteThing - removes thing

func (sdk *MgxSDK) DisconnectThing(thingID, chanID, token string) error
    DisconnectThing - connect thing to a channel

func (sdk *MgxSDK) SendMessage(chanID, msg, token string) error
    SendMessage - send message on Magistrala channel

func (sdk *MgxSDK) SetContentType(ct ContentType) error
    SetContentType - set message content type. Available options are SenML
    JSON, custom JSON and custom binary (octet-stream).

func (sdk *MgxSDK) Thing(id, token string) (Thing, error)
    Thing - gets thing by ID

func (sdk *MgxSDK) Things(token string) ([]Thing, error)
    Things - gets all things

func (sdk *MgxSDK) UpdateChannel(channel Channel, token string) error
    UpdateChannel - update a channel

func (sdk *MgxSDK) UpdateThing(thing Thing, token string) error
    UpdateThing - updates thing by ID

func (sdk *MgxSDK) Health() (magistrala.Health, error)
    Health - things service health check
````
