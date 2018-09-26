package logger

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshalText(t *testing.T) {
	cases := map[string]struct {
		input  string
		output Level
		err    error
	}{
		"select log level Not_A_Level": {"Not_A_Level", 0, ErrInvalidLogLevel},
		"select log level Bad_Input":   {"Bad_Input", 0, ErrInvalidLogLevel},

		"select log level debug": {"debug", Debug, nil},
		"select log level DEBUG": {"DEBUG", Debug, nil},
		"select log level info":  {"info", Info, nil},
		"select log level INFO":  {"INFO", Info, nil},
		"select log level warn":  {"warn", Warn, nil},
		"select log level WARN":  {"WARN", Warn, nil},
		"select log level Error": {"Error", Error, nil},
		"select log level ERROR": {"ERROR", Error, nil},
	}

	for desc, tc := range cases {
		var logLevel Level
		err := logLevel.UnmarshalText(tc.input)
		assert.Equal(t, tc.output, logLevel, fmt.Sprintf("%s: expected %s got %d", desc, tc.output, logLevel))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %d", desc, tc.err, err))

	}

}

func TestLevelIsAllowed(t *testing.T) {
	cases := map[string]struct {
		requestedLevel Level
		allowedLevel   Level
		output         bool
	}{
		"log debug when level debug": {Debug, Debug, true},
		"log info when level debug":  {Info, Debug, true},
		"log warn when level debug":  {Warn, Debug, true},
		"log error when level debug": {Error, Debug, true},
		"log warn when level info":   {Warn, Info, true},
		"log error when level warn":  {Error, Warn, true},
		"log error when level error": {Error, Error, true},

		"log debug when level error": {Debug, Error, false},
		"log info when level error":  {Info, Error, false},
		"log warn when level error":  {Warn, Error, false},
		"log debug when level warn":  {Debug, Warn, false},
		"log info when level warn":   {Info, Warn, false},
		"log debug when level info":  {Debug, Info, false},
	}
	for desc, tc := range cases {
		result := tc.requestedLevel.isAllowed(tc.allowedLevel)
		assert.Equal(t, tc.output, result, fmt.Sprintf("%s: expected %t got %t", desc, tc.output, result))
	}
}
