// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

// UserPasswordReq contains old and new passwords
type UserPasswordReq struct {
	OldPassword string `json:"old_password,omitempty"`
	Password    string `json:"password,omitempty"`
}

// ConnectionIDs contains ID lists of things and channels to be connected
type ConnectionIDs struct {
	ChannelIDs []string `json:"channel_ids"`
	ThingIDs   []string `json:"thing_ids"`
}
