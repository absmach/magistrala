// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/gorilla/websocket"

type connReq struct {
	thingKey string
	chanID   string
	subtopic string
	conn     *websocket.Conn
}
