// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"time"
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
	ThingID    string    `json:"thing_id"`
	ClientCert string    `json:"client_cert"`
	ClientKey  string    `json:"client_key"`
	CertSerial string    `json:"cert_serial"`
	Expiration time.Time `json:"expiration"`
	created    bool
}

func (res certsPageRes) Code() int {
	return http.StatusOK
}

func (res certsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsPageRes) Empty() bool {
	return false
}

func (res certsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res certsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res certsRes) Empty() bool {
	return false
}
