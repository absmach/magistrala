# Mainflux Go SDK

Go SDK, a Go driver for Mainflux HTTP API.

Does both system administration (provisioning) and messaging.

## Installation
Import `"github.com/mainflux/mainflux/sdk/go"` in your Go package.

```
import "github.com/mainflux/mainflux/pkg/sdk/go"```

Then call SDK Go functions to interact with the system.

## API Reference

```go
FUNCTIONS

func NewMfxSDK(host, port string, tls bool) *MfxSDK

func (sdk *MfxSDK) Channel(id, token string) (things.Channel, error)
    Channel - gets channel by ID

func (sdk *MfxSDK) Channels(token string) ([]things.Channel, error)
    Channels - gets all channels

func (sdk *MfxSDK) Connect(struct{[]string, []string}, token string) error
    Connect - connect things to channels

func (sdk *MfxSDK) CreateChannel(data, token string) (string, error)
    CreateChannel - creates new channel and generates UUID

func (sdk *MfxSDK) CreateThing(data, token string) (string, error)
    CreateThing - creates new thing and generates thing UUID

func (sdk *MfxSDK) CreateToken(user, pwd string) (string, error)
    CreateToken - create user token

func (sdk *MfxSDK) CreateUser(user, pwd string) error
    CreateUser - create user

func (sdk *MfxSDK) User(pwd string) (user, error)
    User - gets user

func (sdk *MfxSDK) UpdateUser(user, pwd string) error
    UpdateUser - update user

func (sdk *MfxSDK) UpdatePassword(user, pwd string) error
    UpdatePassword - update user password

func (sdk *MfxSDK) DeleteChannel(id, token string) error
    DeleteChannel - removes channel

func (sdk *MfxSDK) DeleteThing(id, token string) error
    DeleteThing - removes thing

func (sdk *MfxSDK) DisconnectThing(thingID, chanID, token string) error
    DisconnectThing - connect thing to a channel

func (sdk mfSDK) SendMessage(chanID, msg, token string) error
    SendMessage - send message on Mainflux channel

func (sdk mfSDK) SetContentType(ct ContentType) error
    SetContentType - set message content type. Available options are SenML
    JSON, custom JSON and custom binary (octet-stream).

func (sdk mfSDK) Thing(id, token string) (Thing, error)
    Thing - gets thing by ID

func (sdk mfSDK) Things(token string) ([]Thing, error)
    Things - gets all things

func (sdk mfSDK) UpdateChannel(channel Channel, token string) error
    UpdateChannel - update a channel

func (sdk mfSDK) UpdateThing(thing Thing, token string) error
    UpdateThing - updates thing by ID

func (sdk mfSDK) Health() (mainflux.Health, error)
    Health - things service health check
```
