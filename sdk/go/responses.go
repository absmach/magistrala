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

type listThingsRes struct {
	Things []Thing `json:"things,omitempty"`
}

type listChannelsRes struct {
	Channels []Channel `json:"channels,omitempty"`
}
