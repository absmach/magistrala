package mux

import (
	"testing"

	"github.com/stretchr/testify/assert"

	coap "github.com/dustin/go-coap"
)

func TestPathMatcherSingle(t *testing.T) {
	assert := assert.New(t)
	pathRoute := &Route{}
	pathRoute.Path("/stats/{id}/clients")

	testMsg := &coap.Message{}
	testMsg.SetPathString("/stats/1234/clients")
	match := &RouteMatch{}
	assert.True(pathRoute.Match(testMsg, nil, match))
	assert.Equal(1, len(match.Vars))
}

func TestPathMatcherMultiple(t *testing.T) {
	assert := assert.New(t)
	pathRoute := &Route{}
	pathRoute.Path("/stats/{id}/{subid}/clients")

	testMsg := &coap.Message{}
	testMsg.SetPathString("/stats/1234/abcd/clients")
	match := &RouteMatch{}
	assert.True(pathRoute.Match(testMsg, nil, match))
	assert.Equal(2, len(match.Vars))
}

func TestMethodMatcher(t *testing.T) {
	assert := assert.New(t)

	methodRoute := &Route{}
	methodRoute.Methods(coap.GET)

	testMsg := &coap.Message{}
	testMsg.Code = coap.GET

	match := &RouteMatch{}
	assert.True(methodRoute.Match(testMsg, nil, match))
}

func TestCOAPTypeMatcher(t *testing.T) {
	assert := assert.New(t)

	typeRoute := &Route{}
	typeRoute.COAPType(coap.Confirmable)

	testMsg := &coap.Message{}
	testMsg.Type = coap.Confirmable

	match := &RouteMatch{}
	assert.True(typeRoute.Match(testMsg, nil, match))
}
