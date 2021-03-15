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
	Certs []certsRes `json:"certs"`
}

type certsRes struct {
	ThingID    string `json:"thing_id"`
	Cert       string `json:"cert"`
	CertKey    string `json:"cert_key"`
	CertSerial string `json:"cert_serial"`
	CACert     string `json:"ca_cert"`
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

func (res certsRes) Code() int {
	return http.StatusCreated
}

func (res certsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsRes) Empty() bool {
	return false
}
