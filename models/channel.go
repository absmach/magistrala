/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package models

import (
	"github.com/krylovsk/gosenml"
)

type (
	Channel struct {
		Id      string `json:"id"`
		Device  string `json:"device"`
		Created string `json:"created"`
		Updated string `json:"updated"`

		Values  []gosenml.Entry `json:"values"`

		Metadata  map[string]interface{} `json:"metadata"`
	}
)
