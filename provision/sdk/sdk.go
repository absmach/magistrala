package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	mfsdk "github.com/mainflux/mainflux/sdk/go"
)

// Thing is Mainflux SDK thing.
type Thing mfsdk.Thing

// Channel is Mainflux SDK channel.
type Channel mfsdk.Channel

var (
	// ErrGetThing indicates error when fetching new Thing.
	ErrGetThing = errors.New("failed to get created thing")

	// ErrCreateCtrl indicates error when creating control channel.
	ErrCreateCtrl = errors.New("failed to create control channel")

	// ErrCreateData indicates error when creating data channel.
	ErrCreateData = errors.New("failed to create data channel")

	// ErrConn indicates error when connecting channel to proxy thing.
	ErrConn = errors.New("failed to connect proxy to control or data channel")

	// ErrCerts indicates error fetching certificates.
	ErrCerts = errors.New("failed to fetch certs data")

	// ErrConfig indicates error when saving configuration on the Bootstrap service.
	ErrConfig = errors.New("failed to save bootstrap config")

	// ErrWhitelist indicates error when whitelisting thing stored in Bootstrap service.
	ErrWhitelist = errors.New("failed to whitelist")

	// ErrConfigRemove indicates failure while cleaning up from the Bootstrap service.
	ErrConfigRemove = errors.New("failed to remove bootstrap config")

	// ErrCertsRemove indicates failure while cleaning up from the Certs service.
	ErrCertsRemove = errors.New("failed to remove certificate")

	// ErrConflict indicates duplicate unique field.
	ErrConflict = errors.New("duplicate unique field")

	// ErrUnauthorized indicates forbidden access.
	ErrUnauthorized = errors.New("unauthorized access")

	// ErrMalformedEntity indicates malformed request data.
	ErrMalformedEntity = errors.New("malformed data")

	// ErrNotFound indicates that entity doesn't exist.
	ErrNotFound = errors.New("entity not found")
)

// BSConfig represents Config entity to be stored by Bootstrap service.
type BSConfig struct {
	ThingID     string   `json:"thing_id,omitempty"`
	ExternalID  string   `json:"external_id,omitempty"`
	ExternalKey string   `json:"external_key,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Content     string   `json:"content,omitempty"`
	ClientCert  string   `json:"client_cert,omitempty"`
	ClientKey   string   `json:"client_key,omitempty"`
	CACert      string   `json:"ca_cert,omitempty"`
}

// Cert represents certs data.
type Cert struct {
	CACert     string `json:"ca_cert,omitempty"`
	ClientKey  string `json:"client_key,omitempty"`
	ClientCert string `json:"client_cert,omitempty"`
}

// SDK is wrapper around Mainflux SDK that adds some new features
// related to device booststrapping and certs management.
type SDK interface {
	// CreateToken receives credentials and returns user token.
	CreateToken(email, pass string) (string, error)

	// CreateThing registers new thing and returns its id.
	CreateThing(externalID, name, token string) (string, error)

	// Thing returns thing object by id.
	Thing(id, token string) (Thing, error)

	// DeleteThing removes existing thing.
	DeleteThing(id, token string) error

	// CreateChannel creates a new Mainflux Channel.
	CreateChannel(name, chantype, token string) (Channel, error)

	// DeleteChannel removes existing channel.
	DeleteChannel(id, token string) error

	// Connect connects thing to specified channel by id.
	Connect(thingID, chanID, token string) error

	// Cert creates cert using external PKI provider and Certs service.
	Cert(thingID, thingKey, token string) (Cert, error)

	// SaveConfig saves config to Bootstrap service.
	SaveConfig(data BSConfig, token string) error

	// Whitelist sets config state to 1.
	Whitelist(thingID string, data map[string]int, token string) error

	// RemoveConfig removes config associated with the given thing.
	RemoveConfig(id, token string) error

	// RemoveCert revokes and removes cert from Certs service.
	RemoveCert(key, token string) error
}

type provisionSDK struct {
	sdk          mfsdk.SDK
	users        mfsdk.SDK
	certsURL     string
	bsURL        string
	whitelistURL string
}

// New creates new Provision SDK.
func New(certsURL, bsURL, whitelistURL string, thingsSDK, usersSDK mfsdk.SDK) SDK {
	return &provisionSDK{
		sdk:          thingsSDK,
		users:        usersSDK,
		certsURL:     certsURL,
		bsURL:        bsURL,
		whitelistURL: whitelistURL,
	}
}

func (ps *provisionSDK) CreateToken(email, pass string) (string, error) {
	user := mfsdk.User{Email: email, Password: pass}
	return ps.users.CreateToken(user)
}

func (ps *provisionSDK) CreateThing(externalID, name, token string) (string, error) {
	thing := mfsdk.Thing{
		Name:     "",
		Metadata: map[string]interface{}{"ExternalID": externalID},
	}
	return ps.sdk.CreateThing(thing, token)
}

func (ps *provisionSDK) Thing(id, token string) (Thing, error) {
	t, err := ps.sdk.Thing(id, token)
	if err != nil {
		return Thing{}, err
	}
	return Thing(t), nil
}

func (ps *provisionSDK) DeleteThing(id, token string) error {
	return ps.sdk.DeleteThing(id, token)
}

func (ps *provisionSDK) CreateChannel(name, chantype, token string) (Channel, error) {
	retChannel := mfsdk.Channel{}
	retChannel = mfsdk.Channel{Name: name, Metadata: map[string]interface{}{"Type": chantype}}
	chanID, err := ps.sdk.CreateChannel(retChannel, token)
	if err != nil {
		return Channel{}, err
	}

	retChannel.ID = chanID
	return Channel(retChannel), nil
}

func (ps *provisionSDK) DeleteChannel(id, token string) error {
	return ps.sdk.DeleteChannel(id, token)
}

func (ps *provisionSDK) Connect(thingID, chanID, token string) error {
	connIDs := mfsdk.ConnectionIDs{
		ThingIDs:   []string{thingID},
		ChannelIDs: []string{chanID},
	}
	return ps.sdk.Connect(connIDs, token)
}

func (ps *provisionSDK) Cert(thingID, thingKey, token string) (Cert, error) {
	var c Cert
	r := certReq{
		ThingID:  thingID,
		ThingKey: thingKey,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Cert{}, err
	}
	res, err := request(http.MethodPost, token, ps.certsURL, d)
	if err != nil {
		return Cert{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return Cert{}, ErrCerts
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		println(err.Error())
		return Cert{}, err
	}
	if err := json.Unmarshal(body, &c); err != nil {
		return Cert{}, err
	}
	return c, nil
}

func (ps *provisionSDK) SaveConfig(data BSConfig, token string) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}
	res, err := request(http.MethodPost, token, ps.bsURL, d)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusConflict:
		return ErrConflict
	case http.StatusBadRequest:
		return ErrMalformedEntity
	case http.StatusCreated:
		return nil
	default:
		return fmt.Errorf("Failed to save Bootstrap configuration, response status code: %v", res.StatusCode)
	}
}

func (ps *provisionSDK) Whitelist(thingID string, data map[string]int, token string) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}

	res, err := request(http.MethodPut, token, fmt.Sprintf("%s/%s", ps.whitelistURL, thingID), d)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return ErrMalformedEntity
	case http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("Failed to whitelist thing in Bootstrap, response status code: %v", res.StatusCode)
	}
}

func (ps *provisionSDK) RemoveConfig(id, token string) error {
	res, err := request(http.MethodDelete, token, fmt.Sprintf("%s/%s", ps.bsURL, id), nil)
	if res != nil {
		res.Body.Close()
	}
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusForbidden:
		return ErrUnauthorized
	default:
		return ErrConfigRemove
	}
}

func (ps *provisionSDK) RemoveCert(id, token string) error {
	res, err := request(http.MethodDelete, token, fmt.Sprintf("%s/%s", ps.certsURL, id), nil)
	if res != nil {
		res.Body.Close()
	}
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusForbidden:
		return ErrUnauthorized
	default:
		return ErrCertsRemove
	}
}

func request(method, jwt, url string, data []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", jwt)
	c := &http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

type certReq struct {
	ThingID  string `json:"id,omitempty"`
	ThingKey string `json:"key,omitempty"`
}
