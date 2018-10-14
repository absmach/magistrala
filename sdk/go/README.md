# Mainflux Go SDK

Go SDK, a Go driver for Mainflux HTTP API.

Does both system administration (provisioning) and messaging.

## Installation
Import `"github.com/mainflux/mainflux/sdk/go"` in your Go package.

```
import "github.com/mainflux/mainflux/sdk/go"
```

Then call SDK Go functions to interact with the system.

## API Reference

```go
FUNCTIONS

func NewMfxSDK(host, port string, tls bool) *MfxSDK

func (sdk *MfxSDK) Channel(id, token string) (things.Channel, error)
    Channel - gets channel by ID

func (sdk *MfxSDK) Channels(token string) ([]things.Channel, error)
    Channels - gets all channels

func (sdk *MfxSDK) ConnectThing(thingID, chanID, token string) error
    ConnectThing - connect thing to a channel

func (sdk *MfxSDK) CreateChannel(data, token string) (string, error)
    CreateChannel - creates new channel and generates UUID

func (sdk *MfxSDK) CreateThing(data, token string) (string, error)
    CreateThing - creates new thing and generates thing UUID

func (sdk *MfxSDK) CreateToken(user, pwd string) (string, error)
    CreateToken - create user token

func (sdk *MfxSDK) CreateUser(user, pwd string) error
    CreateUser - create user

func (sdk *MfxSDK) DeleteChannel(id, token string) error
    DeleteChannel - removes channel

func (sdk *MfxSDK) DeleteThing(id, token string) error
    DeleteThing - removes thing

func (sdk *MfxSDK) DisconnectThing(thingID, chanID, token string) error
    DisconnectThing - connect thing to a channel

func (sdk *MfxSDK) SendMessage(id, msg, token string) error
    SendMessage - send message on Mainflux channel

func (sdk *MfxSDK) SetContentType(ct string) error
    SetContentType - set message content type. Available options are SenML
    JSON, custom JSON and custom binary (octet-stream).

func (sdk *MfxSDK) Thing(id, token string) (things.Thing, error)
    Thing - gets thing by ID

func (sdk *MfxSDK) Things(token string) ([]things.Thing, error)
    Things - gets all things

func (sdk *MfxSDK) UpdateChannel(id, data, token string) error
    UpdateChannel - update a channel

func (sdk *MfxSDK) UpdateThing(id, data, token string) error
    UpdateThing - updates thing by ID

func (sdk *MfxSDK) Version() (mainflux.VersionInfo, error)
    Version - server health check
```
