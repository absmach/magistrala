package api

import (
	"net/http"

	sdk "github.com/mainflux/mainflux/provision/sdk"
)

type provisionRes struct {
	Thing       sdk.Thing     `json:"thing"`
	Channels    []sdk.Channel `json:"channels"`
	ClientCert  string        `json:"client_cert,omitempty"`
	ClientKey   string        `json:"client_key,omitempty"`
	CACert      string        `json:"ca_cert,omitempty"`
	Whitelisted bool          `json:"whitelisted,omitempty"`
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
