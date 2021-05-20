package codes

import (
	"fmt"
	"strconv"
)

// A Code is an unsigned 16-bit coap code as defined in the coap spec.
type Code uint16

// Request Codes
const (
	GET    Code = 1
	POST   Code = 2
	PUT    Code = 3
	DELETE Code = 4
)

// Response Codes
const (
	Empty                   Code = 0
	Created                 Code = 65
	Deleted                 Code = 66
	Valid                   Code = 67
	Changed                 Code = 68
	Content                 Code = 69
	Continue                Code = 95
	BadRequest              Code = 128
	Unauthorized            Code = 129
	BadOption               Code = 130
	Forbidden               Code = 131
	NotFound                Code = 132
	MethodNotAllowed        Code = 133
	NotAcceptable           Code = 134
	RequestEntityIncomplete Code = 136
	PreconditionFailed      Code = 140
	RequestEntityTooLarge   Code = 141
	UnsupportedMediaType    Code = 143
	InternalServerError     Code = 160
	NotImplemented          Code = 161
	BadGateway              Code = 162
	ServiceUnavailable      Code = 163
	GatewayTimeout          Code = 164
	ProxyingNotSupported    Code = 165
)

//Signaling Codes for TCP
const (
	CSM     Code = 225
	Ping    Code = 226
	Pong    Code = 227
	Release Code = 228
	Abort   Code = 229
)

const _maxCode = 255

var strToCode = map[string]Code{
	`"GET"`:                                GET,
	`"POST"`:                               POST,
	`"PUT"`:                                PUT,
	`"DELETE"`:                             DELETE,
	`"Created"`:                            Created,
	`"Deleted"`:                            Deleted,
	`"Valid"`:                              Valid,
	`"Changed"`:                            Changed,
	`"Content"`:                            Content,
	`"BadRequest"`:                         BadRequest,
	`"Unauthorized"`:                       Unauthorized,
	`"BadOption"`:                          BadOption,
	`"Forbidden"`:                          Forbidden,
	`"NotFound"`:                           NotFound,
	`"MethodNotAllowed"`:                   MethodNotAllowed,
	`"NotAcceptable"`:                      NotAcceptable,
	`"PreconditionFailed"`:                 PreconditionFailed,
	`"RequestEntityTooLarge"`:              RequestEntityTooLarge,
	`"UnsupportedMediaType"`:               UnsupportedMediaType,
	`"InternalServerError"`:                InternalServerError,
	`"NotImplemented"`:                     NotImplemented,
	`"BadGateway"`:                         BadGateway,
	`"ServiceUnavailable"`:                 ServiceUnavailable,
	`"GatewayTimeout"`:                     GatewayTimeout,
	`"ProxyingNotSupported"`:               ProxyingNotSupported,
	`"Capabilities and Settings Messages"`: CSM,
	`"Ping"`:                               Ping,
	`"Pong"`:                               Pong,
	`"Release"`:                            Release,
	`"Abort"`:                              Abort,
}

// UnmarshalJSON unmarshals b into the Code.
func (c *Code) UnmarshalJSON(b []byte) error {
	// From json.Unmarshaler: By convention, to approximate the behavior of
	// Unmarshal itself, Unmarshalers implement UnmarshalJSON([]byte("null")) as
	// a no-op.
	if string(b) == "null" {
		return nil
	}
	if c == nil {
		return fmt.Errorf("nil receiver passed to UnmarshalJSON")
	}

	if ci, err := strconv.ParseUint(string(b), 10, 32); err == nil {
		if ci >= _maxCode {
			return fmt.Errorf("invalid code: %q", ci)
		}

		*c = Code(ci)
		return nil
	}

	if jc, ok := strToCode[string(b)]; ok {
		*c = jc
		return nil
	}
	return fmt.Errorf("invalid code: %q", string(b))
}
