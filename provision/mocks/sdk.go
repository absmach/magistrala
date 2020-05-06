package mocks

import (
	"sync"

	"github.com/gofrs/uuid"
	mfSDK "github.com/mainflux/mainflux/sdk/go"
)

const (
	validEmail = "test@example.com"
	validPass  = "test"
	invalid    = "invalid"
	validToken = "valid_token"
)

// SDK is fake sdk for mocking
type mockSDK struct {
	things      map[string]mfSDK.Thing
	channels    map[string]mfSDK.Channel
	connections map[string][]string
	configs     map[string]mfSDK.BootstrapConfig
	mu          sync.Mutex
}

// NewSDK returns new mock SDK for testing purposes.
func NewSDK() mfSDK.SDK {
	sdk := &mockSDK{}
	sdk.channels = make(map[string]mfSDK.Channel)
	sdk.connections = make(map[string][]string)
	sdk.configs = make(map[string]mfSDK.BootstrapConfig)

	th := mfSDK.Thing{ID: "predefined", Name: "ID"}
	sdk.things = map[string]mfSDK.Thing{"predefined": th}
	sdk.mu = sync.Mutex{}

	return sdk
}

func (s *mockSDK) CreateUser(u mfSDK.User) error {
	panic("CreatUser not implemented")
}

func (s *mockSDK) User(token string) (mfSDK.User, error) {
	panic("User not implemented")
}
func (s *mockSDK) UpdateUser(u mfSDK.User, token string) error {
	panic("UpdateUser not implemented")
}

func (s *mockSDK) UpdatePassword(oldPass, newPass, token string) error {
	panic("UpdatePassword not implemented")
}

// CreateThings registers new things and returns their ids.
func (s *mockSDK) CreateThings(things []mfSDK.Thing, token string) ([]mfSDK.Thing, error) {
	panic("CreateThings not implemented")
}

// Things returns page of things.
func (s *mockSDK) Things(token string, offset, limit uint64, name string) (mfSDK.ThingsPage, error) {
	panic("Things not implemented")
}

// ThingsByChannel returns page of things that are connected to specified
// channel.
func (s *mockSDK) ThingsByChannel(token, chanID string, offset, limit uint64) (mfSDK.ThingsPage, error) {
	panic("ThingsByChannel not implemented")
}

// UpdateThing updates existing thing.
func (s *mockSDK) UpdateThing(thing mfSDK.Thing, token string) error {
	panic("UpdateThing not implemented")
}

// DisconnectThing disconnect thing from specified channel by id.
func (s *mockSDK) DisconnectThing(thingID, chanID, token string) error {
	panic("UpdatePassword not implemented")
}

// CreateChannels registers new channels and returns their ids.
func (s *mockSDK) CreateChannels(channels []mfSDK.Channel, token string) ([]mfSDK.Channel, error) {
	panic("CreateChannels not implemented")
}

// Channels returns page of channels.
func (s *mockSDK) Channels(token string, offset, limit uint64, name string) (mfSDK.ChannelsPage, error) {
	panic("Channels not implemented")
}

// ChannelsByThing returns page of channels that are connected to specified
// thing.
func (s *mockSDK) ChannelsByThing(token, thingID string, offset, limit uint64) (mfSDK.ChannelsPage, error) {
	panic("ChannelsByThing not implemented")
}

// UpdateChannel updates existing channel.
func (s *mockSDK) UpdateChannel(channel mfSDK.Channel, token string) error {
	panic("UpdateChannel not implemented")
}

// SendMessage send message to specified channel.
func (s *mockSDK) SendMessage(chanID, msg, token string) error {
	panic("SendMessage not implemented")
}

// ReadMessages read messages of specified channel.
func (s *mockSDK) ReadMessages(chanID, token string) (mfSDK.MessagesPage, error) {
	panic("ReadMessages not implemented")
}

// SetContentType sets message content type.
func (s *mockSDK) SetContentType(ct mfSDK.ContentType) error {
	panic("SetContentType not implemented")
}

// Version returns used mainflux version.
func (s *mockSDK) Version() (string, error) {
	panic("Version not implemented")
}

// Update updates editable fields of the provided Config.
func (s *mockSDK) UpdateBootstrap(key string, cfg mfSDK.BootstrapConfig) error {
	panic("UpdatePassword not implemented")
}

// View returns Thing Config with given ID belonging to the user identified by the given key.
func (s *mockSDK) Bootstrap(key, id string) (mfSDK.BootstrapConfig, error) {
	panic("UpdatePassword not implemented")
}

// Whitelist updates Thing state Config with given ID belonging to the user identified by the given key.
func (s *mockSDK) Whitelist(key string, cfg mfSDK.BootstrapConfig) error {
	if cfg.ThingID == invalid {
		return mfSDK.ErrFailedWhitelist
	}
	return nil
}

func (s *mockSDK) CreateToken(u mfSDK.User) (string, error) {
	if u.Email != validEmail || u.Password != validPass {
		return "", mfSDK.ErrUnauthorized
	}
	return validToken, nil
}

func (s *mockSDK) CreateThing(t mfSDK.Thing, token string) (string, error) {
	if token != validToken {
		return "", mfSDK.ErrUnauthorized
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	key, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	newThing := mfSDK.Thing{ID: id.String(), Name: t.Name, Key: key.String(), Metadata: t.Metadata}
	s.things[newThing.ID] = newThing

	return newThing.ID, nil
}

func (s *mockSDK) Thing(id, token string) (mfSDK.Thing, error) {
	t := mfSDK.Thing{}

	if token != validToken {
		return t, mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.things[id]; ok {
		return t, nil
	}

	return t, mfSDK.ErrFailedFetch

}

// Channel returns channel data by id.
func (s *mockSDK) Channel(id, token string) (mfSDK.Channel, error) {
	c := mfSDK.Channel{}

	if token != validToken {
		return c, mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.channels[id]; ok {
		return c, nil
	}

	return c, mfSDK.ErrFailedFetch
}

func (s *mockSDK) DeleteThing(id string, token string) error {
	if id == invalid {
		return mfSDK.ErrFailedRemoval
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.things, id)
	return nil
}

func (s *mockSDK) CreateChannel(channel mfSDK.Channel, token string) (string, error) {
	if token != validToken {
		return "", mfSDK.ErrUnauthorized
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newChan := mfSDK.Channel{ID: id.String(), Name: channel.Name, Metadata: channel.Metadata}
	s.channels[newChan.ID] = newChan

	return newChan.ID, nil
}

func (s *mockSDK) DeleteChannel(id string, token string) error {
	if id == invalid {
		return mfSDK.ErrFailedRemoval
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.channels, id)
	return nil
}

// ConnectThing connects thing to specified channel by id.
func (s *mockSDK) Connect(connIDs mfSDK.ConnectionIDs, token string) error {
	if token != validToken {
		return mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, thingID := range connIDs.ThingIDs {
		if _, ok := s.things[thingID]; !ok {
			return mfSDK.ErrFailedFetch
		}
	}

	for _, channelID := range connIDs.ChannelIDs {
		if _, ok := s.channels[channelID]; !ok {
			return mfSDK.ErrFailedFetch
		}

	}

	for _, thingID := range connIDs.ThingIDs {
		for _, chanID := range connIDs.ChannelIDs {
			conns := s.connections[thingID]
			conns = append(conns, chanID)
			s.connections[thingID] = conns
		}
	}

	return nil
}

func (s *mockSDK) AddBootstrap(token string, cfg mfSDK.BootstrapConfig) (string, error) {
	if token != validToken {
		return "", mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, val := range s.configs {
		if val.ExternalID == cfg.ExternalID {
			return "", mfSDK.ErrFailedCreation
		}
	}

	mfid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	cfg.MFThing = mfid.String()
	s.configs[string(mfid.String())] = cfg
	return mfid.String(), nil
}

func (s *mockSDK) ViewBootstrap(token string, id string) (mfSDK.BootstrapConfig, error) {
	if token != validToken {
		return mfSDK.BootstrapConfig{}, mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.configs[id]; !ok {
		return mfSDK.BootstrapConfig{}, mfSDK.ErrFailedFetch
	}

	return s.configs[id], nil

}

func (s *mockSDK) RemoveBootstrap(token, id string) error {
	if token != validToken {
		return mfSDK.ErrUnauthorized
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.configs[id]; !ok {
		return mfSDK.ErrFailedFetch
	}
	delete(s.configs, id)
	return nil
}

func (s *mockSDK) Cert(thingID, thingKey string, token string) (mfSDK.Cert, error) {
	if thingID == invalid || thingKey == invalid {
		return mfSDK.Cert{}, mfSDK.ErrCerts
	}
	return mfSDK.Cert{}, nil
}

func (s *mockSDK) RemoveCert(key string, token string) error {
	if key == invalid {
		return mfSDK.ErrCertsRemove
	}
	return nil
}
