// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/mainflux/mainflux/coap"
	go_coap "github.com/plgd-dev/go-coap/v2"
	"github.com/plgd-dev/go-coap/v2/mux"
	udp "github.com/plgd-dev/go-coap/v2/udp"
	"github.com/stretchr/testify/assert"
)

const expectedCount = uint64(1)

var (
	msgChan = make(chan []byte)
	c       *coap.Client
	count   uint64
)

func handler(w mux.ResponseWriter, r *mux.Message) {

}

func TestHandle(t *testing.T) {
	// server code
	r := mux.NewRouter()
	r.Handle("/", mux.HandlerFunc(handler))
	log.Fatal(go_coap.ListenAndServe("udp", ":5688", r))

	// client code
	conn, err := udp.Dial("localhost:5688")
	assert.Nil(t, err, fmt.Sprintf("expected nil error, got: %s\n", err))

	resp, err := conn.Client().Get(context.Background(), "")
	assert.Nil(t, err, fmt.Sprintf("expected nil error, got: %s\n", err))

	log.Printf("Response Payload: %v\n", resp.String())
}
