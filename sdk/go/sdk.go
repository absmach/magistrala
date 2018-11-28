//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
)

const (
	// CTJSON represents JSON content type.
	CTJSON ContentType = "application/json"

	// CTJSONSenML represents JSON SenML content type.
	CTJSONSenML ContentType = "application/senml+json"

	// CTBinary represents binary content type.
	CTBinary ContentType = "application/octet-stream"
)

var (
	// ErrConflict indicates that create or update of entity failed because
	// entity with same name already exists.
	ErrConflict = errors.New("entity already exists")

	// ErrFailedCreation indicates that entity creation failed.
	ErrFailedCreation = errors.New("failed to create entity")

	// ErrFailedUpdate indicates that entity update failed.
	ErrFailedUpdate = errors.New("failed to update entity")

	// ErrFailedPublish indicates that publishing message failed.
	ErrFailedPublish = errors.New("failed to publish message")

	// ErrFailedRemoval indicates that entity removal failed.
	ErrFailedRemoval = errors.New("failed to remove entity")

	// ErrFailedConnection indicates that connecting thing to channel failed.
	ErrFailedConnection = errors.New("failed to connect thing to channel")

	// ErrFailedDisconnect indicates that disconnecting thing from a channel failed.
	ErrFailedDisconnect = errors.New("failed to disconnect thing from channel")

	// ErrInvalidArgs indicates that invalid argument was passed.
	ErrInvalidArgs = errors.New("invalid argument passed")

	// ErrFetchFailed indicates that fetching of entity data failed.
	ErrFetchFailed = errors.New("failed to fetch entity")

	// ErrUnauthorized indicates unauthorized access.
	ErrUnauthorized = errors.New("unauthorized access")

	// ErrNotFound indicates that entity doesn't exist.
	ErrNotFound = errors.New("entity not found")

	// ErrInvalidContentType indicates that nonexistent message content type
	// was passed.
	ErrInvalidContentType = errors.New("Unknown Content Type")
)

// ContentType represents all possible content types.
type ContentType string

var _ SDK = (*mfSDK)(nil)

// User represents mainflux user its credentials.
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Thing represents mainflux thing.
type Thing struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Key      string `json:"key,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

// Channel represents mainflux channel.
type Channel struct {
	ID       string  `json:"id,omitempty"`
	Name     string  `json:"name"`
	Things   []Thing `json:"connected,omitempty"`
	Metadata string  `json:"metadata,omitempty"`
}

// SDK contains Mainflux API.
type SDK interface {
	// CreateUser registers mainflux user.
	CreateUser(user User) error

	// CreateToken receives credentials and returns user token.
	CreateToken(user User) (string, error)

	// CreateThing registers new thing and returns its id.
	CreateThing(thing Thing, token string) (string, error)

	// Things returns page of things.
	Things(token string, offset, limit uint64) ([]Thing, error)

	// Thing returns thing object by id.
	Thing(id, token string) (Thing, error)

	// UpdateThing updates existing thing.
	UpdateThing(thing Thing, token string) error

	// DeleteThing removes existing thing.
	DeleteThing(id, token string) error

	// ConnectThing connects thing to specified channel by id.
	ConnectThing(thingID, chanID, token string) error

	// DisconnectThing disconnect thing from specified channel by id.
	DisconnectThing(thingID, chanID, token string) error

	// CreateChannel creates new channel and returns its id.
	CreateChannel(channel Channel, token string) (string, error)

	// Channels returns page of channels.
	Channels(token string, offset, limit uint64) ([]Channel, error)

	// Channel returns channel data by id.
	Channel(id, token string) (Channel, error)

	// UpdateChannel updates existing channel.
	UpdateChannel(channel Channel, token string) error

	// DeleteChannel removes existing channel.
	DeleteChannel(id, token string) error

	// SendMessage send message to specified channel.
	SendMessage(chanID, msg, token string) error

	// SetContentType sets message content type.
	SetContentType(ct ContentType) error

	// Version returns used mainflux version.
	Version() (string, error)
}

type mfSDK struct {
	url               string
	usersPrefix       string
	thingsPrefix      string
	httpAdapterPrefix string
	msgContentType    ContentType
	client            *http.Client
}

// Config contains sdk configuration parameters.
type Config struct {
	BaseURL           string
	UsersPrefix       string
	ThingsPrefix      string
	HTTPAdapterPrefix string
	MsgContentType    ContentType
	TLSVerification   bool
}

// NewSDK returns new mainflux SDK instance.
func NewSDK(conf Config) SDK {
	return &mfSDK{
		url:               conf.BaseURL,
		usersPrefix:       conf.UsersPrefix,
		thingsPrefix:      conf.ThingsPrefix,
		httpAdapterPrefix: conf.HTTPAdapterPrefix,
		msgContentType:    conf.MsgContentType,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: !conf.TLSVerification,
				},
			},
		},
	}
}

func (sdk mfSDK) sendRequest(req *http.Request, token, contentType string) (*http.Response, error) {
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	if contentType != "" {
		req.Header.Add("Content-Type", contentType)
	}

	return sdk.client.Do(req)
}

func createURL(baseURL, prefix, endpoint string) string {
	if prefix == "" {
		return fmt.Sprintf("%s/%s", baseURL, endpoint)
	}

	return fmt.Sprintf("%s/%s/%s", baseURL, prefix, endpoint)
}
