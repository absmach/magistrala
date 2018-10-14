//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk

import (
	"github.com/mainflux/mainflux/things"
)

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

type listThingsRes struct {
	Things []things.Thing `json:"things,omitempty"`
}

type listChannelsRes struct {
	Channels []things.Channel `json:"channels,omitempty"`
}
