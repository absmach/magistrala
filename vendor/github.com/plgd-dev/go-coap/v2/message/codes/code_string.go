package codes

import (
	"fmt"
	"strconv"
)

var codeToString = map[Code]string{
	Empty:                 "Empty",
	GET:                   "GET",
	POST:                  "POST",
	PUT:                   "PUT",
	DELETE:                "DELETE",
	Created:               "Created",
	Deleted:               "Deleted",
	Valid:                 "Valid",
	Changed:               "Changed",
	Content:               "Content",
	BadRequest:            "BadRequest",
	Unauthorized:          "Unauthorized",
	BadOption:             "BadOption",
	Forbidden:             "Forbidden",
	NotFound:              "NotFound",
	MethodNotAllowed:      "MethodNotAllowed",
	NotAcceptable:         "NotAcceptable",
	PreconditionFailed:    "PreconditionFailed",
	RequestEntityTooLarge: "RequestEntityTooLarge",
	UnsupportedMediaType:  "UnsupportedMediaType",
	InternalServerError:   "InternalServerError",
	NotImplemented:        "NotImplemented",
	BadGateway:            "BadGateway",
	ServiceUnavailable:    "ServiceUnavailable",
	GatewayTimeout:        "GatewayTimeout",
	ProxyingNotSupported:  "ProxyingNotSupported",
	CSM:                   "Capabilities and Settings Messages",
	Ping:                  "Ping",
	Pong:                  "Pong",
	Release:               "Release",
	Abort:                 "Abort",
}

func (c Code) String() string {
	val, ok := codeToString[c]
	if ok {
		return val
	}
	return "Code(" + strconv.FormatInt(int64(c), 10) + ")"
}

func ToCode(v string) (Code, error) {
	for key, val := range codeToString {
		if v == val {
			return key, nil
		}
	}
	return 0, fmt.Errorf("not found")
}
