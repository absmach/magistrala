// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/absmach/supermq/ws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

const expectedCount = uint64(1)

var (
	msgChan = make(chan []byte)
	c       *ws.Client
	count   uint64

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		atomic.AddUint64(&count, 1)
		msgChan <- message
	}
}

func TestHandle(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(handler))
	defer s.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := strings.Replace(s.URL, "http", "ws", 1)

	// Connect to the server
	wsConn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer wsConn.Close()

	c = ws.NewClient(wsConn)

	cases := []struct {
		desc            string
		publisher       string
		expectedPayload []byte
		expectMsg       bool
	}{
		{
			desc:            "handling with different id from ws.Client",
			publisher:       msg.Publisher,
			expectedPayload: msg.Payload,
			expectMsg:       true,
		},
		{
			desc:            "handling with same id as ws.Client (empty by default) drops message",
			publisher:       "",
			expectedPayload: []byte{},
			expectMsg:       false,
		},
	}

	for _, tc := range cases {
		msg.Publisher = tc.publisher
		err = c.Handle(&msg)
		assert.Nil(t, err, fmt.Sprintf("expected nil error from handle, got: %s", err))
		receivedMsg := []byte{}
		switch tc.expectMsg {
		case true:
			rec := <-msgChan // Wait for the message to be received.
			receivedMsg = rec
		case false:
			time.Sleep(100 * time.Millisecond) // Give time to server to process c.Handle call.
		}
		assert.Equal(t, tc.expectedPayload, receivedMsg, fmt.Sprintf("%s: expected %+v, got %+v", tc.desc, &msg, receivedMsg))
	}
	c := atomic.LoadUint64(&count)
	assert.Equal(t, expectedCount, c, fmt.Sprintf("expected message count %d, got %d", expectedCount, c))
}
