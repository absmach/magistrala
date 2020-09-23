// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import "github.com/mainflux/mainflux/pkg/transformers/senml"

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
