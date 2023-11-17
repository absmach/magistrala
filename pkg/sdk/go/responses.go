// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"time"

	"github.com/absmach/magistrala/pkg/transformers/senml"
)

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
	Channels []Channel `json:"groups"`
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

type MembersPage struct {
	Members []User `json:"members"`
	pageRes
}

// MembershipsPage contains page related metadata as well as list of memberships that
// belong to this page.
type MembershipsPage struct {
	pageRes
	Memberships []Group `json:"memberships"`
}

type revokeCertsRes struct {
	RevocationTime time.Time `json:"revocation_time"`
}

// BoostrapsPage contains list of boostrap configs in a page with proper metadata.
type BootstrapPage struct {
	Configs []BootstrapConfig `json:"configs"`
	pageRes
}

type CertSerials struct {
	Certs []Cert `json:"certs"`
	pageRes
}

type SubscriptionPage struct {
	Subscriptions []Subscription `json:"subscriptions"`
	pageRes
}

type identifyThingResp struct {
	ID string `json:"id,omitempty"`
}

type DomainsPage struct {
	Domains []Domain `json:"domains"`
	pageRes
}
