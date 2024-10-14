// Copyright (c) Abstract Machines
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
	ThingID      string    `json:"thing_id"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	SerialNumber string    `json:"serial_number"`
	ExpiryTime   time.Time `json:"expiry_time"`
	Revoked      bool      `json:"revoked"`
	issued       bool
}

type revokeCertsRes struct {
	RevocationTime time.Time `json:"revocation_time"`
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
	if res.issued {
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

func (res revokeCertsRes) Code() int {
	return http.StatusOK
}

func (res revokeCertsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeCertsRes) Empty() bool {
	return false
}
