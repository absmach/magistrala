package ws_test

import (
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/ws"
	"github.com/stretchr/testify/assert"
)

func TestClose(t *testing.T) {
	channel := ws.Channel{make(chan mainflux.RawMessage), make(chan bool)}
	channel.Close()
	_, closed := <-channel.Closed
	_, messagesClosed := <-channel.Messages
	assert.False(t, closed, "channel closed stayed open")
	assert.False(t, messagesClosed, "channel messages stayed open")
}
