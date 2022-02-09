// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"errors"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	protocol = "mqtt"
	username = "mainflux-mqtt"
	qos      = 2
)

var errConnect = errors.New("failed to connect to MQTT broker")

func newClient(address string, timeout time.Duration) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		SetUsername(username).
		AddBroker(address)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Error() != nil {
		return nil, token.Error()
	}

	ok := token.WaitTimeout(timeout)
	if !ok {
		return nil, errConnect
	}

	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}
