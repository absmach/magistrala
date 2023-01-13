// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"net/http"
	"time"

	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

type createThingsRes struct {
	Things []Thing `json:"things"`
}

type createChannelsRes struct {
	Channels []Channel `json:"channels"`
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

// ThingsPage contains list of things in a page with proper metadata.
type ThingsPage struct {
	Things []Thing `json:"things"`
	pageRes
}

// ChannelsPage contains list of channels in a page with proper metadata.
type ChannelsPage struct {
	Channels []Channel `json:"channels"`
	pageRes
}

// MessagesPage contains list of messages in a page with proper metadata.
type MessagesPage struct {
	Messages []senml.Message `json:"messages,omitempty"`
	pageRes
}

type GroupsPage struct {
	Groups []Group `json:"groups"`
	pageRes
}

type UsersPage struct {
	Users []User `json:"users"`
	pageRes
}

type MembersPage struct {
	Members []string `json:"members"`
	pageRes
}

type KeyRes struct {
	ID        string     `json:"id,omitempty"`
	Value     string     `json:"value,omitempty"`
	IssuedAt  time.Time  `json:"issued_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (res KeyRes) Code() int {
	return http.StatusCreated
}

func (res KeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res KeyRes) Empty() bool {
	return res.Value == ""
}

type retrieveKeyRes struct {
	ID        string     `json:"id,omitempty"`
	IssuerID  string     `json:"issuer_id,omitempty"`
	Subject   string     `json:"subject,omitempty"`
	IssuedAt  time.Time  `json:"issued_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (res retrieveKeyRes) Code() int {
	return http.StatusOK
}

func (res retrieveKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res retrieveKeyRes) Empty() bool {
	return false
}

type revokeCertsRes struct {
	RevocationTime time.Time `json:"revocation_time"`
}
