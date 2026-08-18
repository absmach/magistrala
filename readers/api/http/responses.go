// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/readers"
)

var _ magistrala.Response = (*pageRes)(nil)

type pageRes struct {
	readers.PageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
}

func (res pageRes) MarshalJSON() ([]byte, error) {
	pm := res.PageMetadata
	pm.DeviceScope = nil

	type pageResponse pageRes
	return json.Marshal(pageResponse{
		PageMetadata: pm,
		Total:        res.Total,
		Messages:     res.Messages,
	})
}

func (res pageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res pageRes) Code() int {
	return http.StatusOK
}

func (res pageRes) Empty() bool {
	return false
}

var _ magistrala.Response = (*gatewayDevicesRes)(nil)

// gatewayDeviceRes is one device observed publishing through a gateway
// (MG-15): device_id is the raw stored serial, verbatim.
type gatewayDeviceRes struct {
	DeviceID     string  `json:"device_id"`
	LastSeen     float64 `json:"last_seen"`
	MessageCount uint64  `json:"message_count"`
}

type gatewayDevicesRes struct {
	Total   uint64             `json:"total"`
	Offset  uint64             `json:"offset"`
	Limit   uint64             `json:"limit"`
	Devices []gatewayDeviceRes `json:"devices"`
}

func (res gatewayDevicesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res gatewayDevicesRes) Code() int {
	return http.StatusOK
}

func (res gatewayDevicesRes) Empty() bool {
	return false
}

var _ magistrala.Response = (*deviceGatewaysRes)(nil)

// deviceGatewayRes is one gateway observed relaying for a device (MG-15).
type deviceGatewayRes struct {
	Publisher    string  `json:"publisher"`
	LastSeen     float64 `json:"last_seen"`
	MessageCount uint64  `json:"message_count"`
}

type deviceGatewaysRes struct {
	Total      uint64             `json:"total"`
	Offset     uint64             `json:"offset"`
	Limit      uint64             `json:"limit"`
	Publishers []deviceGatewayRes `json:"publishers"`
}

func (res deviceGatewaysRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deviceGatewaysRes) Code() int {
	return http.StatusOK
}

func (res deviceGatewaysRes) Empty() bool {
	return false
}
