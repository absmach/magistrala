// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package lora

// RxInfo receiver parameters.
type RxInfo []struct {
	Mac       string  `json:"mac"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Time      string  `json:"time"`
	Rssi      float64 `json:"rssi"`
	LoRaSNR   float64 `json:"loRaSNR"`
}

// DataRate lora data rate.
type DataRate struct {
	Modulation   string  `json:"modulation"`
	Bandwidth    float64 `json:"bandwidth"`
	SpreadFactor int64   `json:"spreadFactor"`
}

// TxInfo transmeter parameters.
type TxInfo struct {
	Frequency float64  `json:"frequency"`
	DataRate  DataRate `json:"dataRate"`
	Adr       bool     `json:"adr"`
	CodeRate  string   `json:"codeRate"`
}

// Message lora msg (https://www.chirpstack.io/application-server/integrations/events).
type Message struct {
	ApplicationID       string      `json:"applicationID"`
	ApplicationName     string      `json:"applicationName"`
	DeviceName          string      `json:"deviceName"`
	DevEUI              string      `json:"devEUI"`
	DeviceStatusBattery string      `json:"deviceStatusBattery"`
	DeviceStatusMrgin   string      `json:"deviceStatusMargin"`
	RxInfo              RxInfo      `json:"rxInfo"`
	TxInfo              TxInfo      `json:"txInfo"`
	FCnt                int         `json:"fCnt"`
	FPort               int         `json:"fPort"`
	Data                string      `json:"data"`
	Object              interface{} `json:"object"`
}
