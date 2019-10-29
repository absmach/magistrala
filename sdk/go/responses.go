// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import "github.com/mainflux/mainflux"

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

type createThingsRes struct {
	Things []Thing `json:"things"`
}

type createChannelsRes struct {
	Channels []Channel `json:"channels"`
}

type thingsPageRes struct {
	Things []Thing `json:"things,omitempty"`
	Total  uint64  `json:"total"`
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
}

type channelsPageRes struct {
	Channels []Channel `json:"channels,omitempty"`
	Total    uint64    `json:"total"`
	Offset   uint64    `json:"offset"`
	Limit    uint64    `json:"limit"`
}

type messagesPageRes struct {
	Total    uint64             `json:"total"`
	Offset   uint64             `json:"offset"`
	Limit    uint64             `json:"limit"`
	Messages []mainflux.Message `json:"messages,omitempty"`
}
