// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import "github.com/absmach/magistrala/pkg/transformers/senml"

type PageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

// MessagesPage contains list of messages in a page with proper metadata.
type MessagesPage struct {
	Messages []senml.Message `json:"messages,omitempty"`
	PageRes
}

// GatewayDeviceStat is one device observed publishing through a gateway
// (MG-15). DeviceID is the raw stored serial, verbatim.
type GatewayDeviceStat struct {
	DeviceID     string  `json:"device_id"`
	LastSeen     float64 `json:"last_seen"`
	MessageCount uint64  `json:"message_count"`
}

// GatewayDevicesPage contains a page of the devices observed publishing
// through a gateway (MG-15).
type GatewayDevicesPage struct {
	Devices []GatewayDeviceStat `json:"devices,omitempty"`
	PageRes
}

// DeviceGatewayStat is one gateway observed relaying for a device (MG-15).
type DeviceGatewayStat struct {
	Publisher    string  `json:"publisher"`
	LastSeen     float64 `json:"last_seen"`
	MessageCount uint64  `json:"message_count"`
}

// DeviceGatewaysPage contains a page of the gateways observed relaying for a
// device (MG-15).
type DeviceGatewaysPage struct {
	Publishers []DeviceGatewayStat `json:"publishers,omitempty"`
	PageRes
}

// BootstrapPage contains list of bootstrap configs in a page with proper metadata.
type BootstrapPage struct {
	Configs []BootstrapConfig `json:"configs"`
	PageRes
}

// BootstrapProfilesPage contains list of bootstrap profiles in a page with proper metadata.
type BootstrapProfilesPage struct {
	Profiles []BootstrapProfile `json:"profiles"`
	PageRes
}

