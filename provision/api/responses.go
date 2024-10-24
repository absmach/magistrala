// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
)

var _ magistrala.Response = (*provisionRes)(nil)

type provisionRes struct {
	Clients     []sdk.Client      `json:"clients"`
	Channels    []sdk.Channel     `json:"channels"`
	ClientCert  map[string]string `json:"client_cert,omitempty"`
	ClientKey   map[string]string `json:"client_key,omitempty"`
	CACert      string            `json:"ca_cert,omitempty"`
	Whitelisted map[string]bool   `json:"whitelisted,omitempty"`
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

type mappingRes struct {
	Data interface{}
}

func (res mappingRes) Code() int {
	return http.StatusOK
}

func (res mappingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res mappingRes) Empty() bool {
	return false
}

func (res mappingRes) MarshalJSON() ([]byte, error) {
	return json.Marshal(res.Data)
}
