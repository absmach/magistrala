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
		Id     string `json:"id"`
		Device string `json:"device"`

		// Name is optional. If present, it is pre-pended to `bn` member of SenML.
		Name string `json:"name"`
		// Unit is optional. If present, it is pre-pended to `bu` member of SenML.
		Unit string `json:"unit"`

		Values []gosenml.Entry `json:"values"`

		Created string `json:"created"`
		Updated string `json:"updated"`

		Metadata map[string]interface{} `json:"metadata"`
	}
)
