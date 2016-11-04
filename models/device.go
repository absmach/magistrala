/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package models

type (
	Device struct {
		Id   string `json:"id"`
		Name string `json:"name"`

		Description string `json:"description"`

		Online        bool   `json:"online"`
		ConnectedAt   string `json:"connected_at"`
		DisonnectedAt string `json:"disconnected_at"`

		Channels []Channel `json:"channels"`

		Created string `json:"created"`
		Updated string `json:"updated"`

		Metadata map[string]interface{} `json:"metadata"`
	}
)
