// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type certsPageRes struct {
	pageRes
	Certs []certsResponse `json:"certs"`
	Error string          `json:"error,omitempty"`
}

type certsResponse struct {
	ClientCert map[string]string `json:"client_cert"`
	ClientKey  map[string]string `json:"client_key"`
	Serial     string            `json:"serial"`
	ThingID    string            `json:"thing_id"`
	CACert     string            `json:"ca_cert"`
	Error      string            `json:"error"`
}

func (res certsPageRes) Code() int {
	return http.StatusCreated
}

func (res certsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsPageRes) Empty() bool {
	return false
}

func (res certsResponse) Code() int {
	return http.StatusCreated
}

func (res certsResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res certsResponse) Empty() bool {
	return false
}
