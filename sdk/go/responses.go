// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

type createThingsRes struct {
	Things []Thing `json:"things"`
}

type createChannelsRes struct {
	Channels []Channel `json:"channels"`
}
