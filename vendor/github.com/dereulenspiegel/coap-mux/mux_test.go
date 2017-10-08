package mux

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	coap "github.com/dustin/go-coap"
)

func TestBasicRouting(t *testing.T) {
	assert := assert.New(t)

	router := NewRouter()
	router.Handle("/a/{id1}/{id2}/b/{id3}", nil).Methods(coap.GET)

	validMsg := &coap.Message{}
	validMsg.SetPathString("/a/1234/abcd/b/23")
	validMsg.Code = coap.GET

	match := &RouteMatch{}
	assert.True(router.Match(validMsg, nil, match))

	invalidMsg := &coap.Message{}
	invalidMsg.SetPathString("/a/1234/abcd/b/23")
	invalidMsg.Code = coap.PUT

	match = &RouteMatch{}
	assert.False(router.Match(invalidMsg, nil, match))
}

type TestHandler struct {
	assert *assert.Assertions
}

var handlerCalled bool

func (t TestHandler) ServeCOAP(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	handlerCalled = true

	id1 := Var(m, "id1")
	t.assert.Equal("1234", id1)

	id2 := Var(m, "id2")
	t.assert.Equal("abcd", id2)

	id3 := Var(m, "id3")
	t.assert.Equal("23", id3)

	return nil
}

func TestVariableAccess(t *testing.T) {
	assert := assert.New(t)
	handlerCalled = false
	router := NewRouter()
	router.Handle("/a/{id1}/{id2}/b/{id3}", TestHandler{assert: assert}).Methods(coap.GET)

	validMsg := &coap.Message{}
	validMsg.SetPathString("/a/1234/abcd/b/23")
	validMsg.Code = coap.GET

	router.ServeCOAP(nil, nil, validMsg)
	assert.True(handlerCalled)

	assert.Equal("", Var(validMsg, "id1"))
}
