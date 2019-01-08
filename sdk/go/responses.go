//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

type tokenRes struct {
	Token string `json:"token,omitempty"`
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

type listMessagesRes struct {
	Messages []Message `json:"messages,omitempty"`
}
