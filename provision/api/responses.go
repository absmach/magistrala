package api

import (
	"net/http"

	SDK "github.com/mainflux/mainflux/pkg/sdk/go"
)

type provisionRes struct {
	Things      []SDK.Thing       `json:"things"`
	Channels    []SDK.Channel     `json:"channels"`
	ClientCert  map[string]string `json:"client_cert,omitempty"`
	ClientKey   map[string]string `json:"client_key,omitempty"`
	CACert      string            `json:"ca_cert,omitempty"`
	Whitelisted map[string]bool   `json:"whitelisted,omitempty"`
	Error       string            `json:"error,omitempty"`
}

func (res provisionRes) Code() int {
	return http.StatusCreated
}

func (res provisionRes) Headers() map[string]string {
	return map[string]string{}
}

func (res provisionRes) Empty() bool {
	return false
}
