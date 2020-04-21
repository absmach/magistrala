package mocks

//
import (
	"sync"

	"github.com/gofrs/uuid"
	provsdk "github.com/mainflux/mainflux/provision/sdk"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
)

const (
	validEmail   = "test@example.com"
	validPass    = "test"
	invalid      = "invalid"
	validToken   = "valid_token"
	invalidToken = "invalid_token"
)

var thingIDs = []string{"ids"}

// SDK is fake sdk for mocking
type mockSDK struct {
	things      map[string]provsdk.Thing
	channels    map[string]provsdk.Channel
	connections map[string][]string
	configs     map[string]provsdk.BSConfig
	mu          sync.Mutex
}

// NewSDK returns new mock SDK for testing purposes.
func NewSDK() provsdk.SDK {
	sdk := &mockSDK{}
	sdk.channels = make(map[string]provsdk.Channel)
	sdk.connections = make(map[string][]string)
	sdk.configs = make(map[string]provsdk.BSConfig)

	th := provsdk.Thing{ID: "predefined", Name: "ID"}
	sdk.things = map[string]provsdk.Thing{"predefined": th}
	sdk.mu = sync.Mutex{}

	return sdk
}

// CreateToken receives credentials and returns user token.
func (s *mockSDK) CreateToken(email, pass string) (string, error) {
	if email != validEmail || pass != validPass {
		return "", mfsdk.ErrFailedCreation
	}
	return validToken, nil
}

func (s *mockSDK) Cert(thingID, thingKey string, token string) (provsdk.Cert, error) {
	if thingID == invalid || thingKey == invalid {
		return provsdk.Cert{}, provsdk.ErrCerts
	}
	return provsdk.Cert{}, nil
}

func (s *mockSDK) SaveConfig(data provsdk.BSConfig, token string) error {
	if data.ThingID == invalid {
		return mfsdk.ErrFailedCreation
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[data.ExternalID]; ok {
		return provsdk.ErrConflict
	}
	s.configs[data.ExternalID] = data
	return nil
}

func (s *mockSDK) Whitelist(thingID string, data map[string]int, token string) error {
	if thingID == invalid {
		return provsdk.ErrWhitelist
	}
	return nil
}

func (s *mockSDK) RemoveConfig(id string, token string) error {
	if id == invalid {
		return provsdk.ErrConfigRemove
	}
	return nil
}

func (s *mockSDK) RemoveCert(key string, token string) error {
	if key == invalid {
		return provsdk.ErrCertsRemove
	}
	return nil
}

func (s *mockSDK) CreateThing(externalID string, name string, token string) (string, error) {
	if token != validToken {
		return "", mfsdk.ErrFailedCreation
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

	newThing := provsdk.Thing{ID: id.String(), Name: name, Key: key.String(), Metadata: map[string]interface{}{"ExternalID": externalID}}
	s.things[newThing.ID] = newThing

	return newThing.ID, nil
}

func (s *mockSDK) Thing(id, token string) (provsdk.Thing, error) {
	t := provsdk.Thing{}

	if token != validToken {
		return t, mfsdk.ErrFailedFetch
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.things[id]; ok {
		return t, nil
	}

	return t, mfsdk.ErrFailedFetch
}

func (s *mockSDK) DeleteThing(id string, token string) error {
	if id == invalid {
		return mfsdk.ErrFailedRemoval
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.things, id)
	return nil
}

func (s *mockSDK) CreateChannel(name string, chantype string, token string) (provsdk.Channel, error) {
	if token != validToken {
		return provsdk.Channel{}, mfsdk.ErrFailedCreation
	}

	id, err := uuid.NewV4()
	if err != nil {
		return provsdk.Channel{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newChan := provsdk.Channel{ID: id.String(), Name: name, Metadata: map[string]interface{}{"Type": chantype}}
	s.channels[newChan.ID] = newChan

	return newChan, nil
}

func (s *mockSDK) DeleteChannel(id string, token string) error {
	if id == invalid {
		return mfsdk.ErrFailedRemoval
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.channels, id)
	return nil
}

// ConnectThing connects thing to specified channel by id.
func (s *mockSDK) Connect(thingID, chanID, token string) error {
	if token != validToken {
		return mfsdk.ErrFailedCreation
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.things[thingID]; !ok {
		return mfsdk.ErrFailedFetch
	}
	if _, ok := s.channels[chanID]; !ok {
		return mfsdk.ErrFailedFetch
	}

	conns := s.connections[thingID]
	conns = append(conns, chanID)
	s.connections[thingID] = conns
	return nil
}
