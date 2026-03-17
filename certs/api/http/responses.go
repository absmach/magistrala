// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/absmach/supermq/certs"
)

var (
	_ Response = (*revokeCertRes)(nil)
	_ Response = (*issueCertRes)(nil)
	_ Response = (*renewCertRes)(nil)
	_ Response = (*ocspRawRes)(nil)
)

type renewCertRes struct {
	renewed     bool
	Certificate certs.Certificate `json:"certificate,omitempty"`
}

func (res renewCertRes) Code() int {
	if res.renewed {
		return http.StatusOK
	}

	return http.StatusBadRequest
}

func (res renewCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res renewCertRes) Empty() bool {
	return false
}

type revokeCertRes struct {
	revoked bool
}

func (res revokeCertRes) Code() int {
	if res.revoked {
		return http.StatusNoContent
	}

	return http.StatusUnprocessableEntity
}

func (res revokeCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeCertRes) Empty() bool {
	return true
}

type deleteCertRes struct {
	deleted bool
}

func (res deleteCertRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusUnprocessableEntity
}

func (res deleteCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteCertRes) Empty() bool {
	return true
}

type issueCertRes struct {
	SerialNumber string    `json:"serial_number"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked"`
	ExpiryTime   time.Time `json:"expiry_time"`
	EntityID     string    `json:"entity_id"`
	issued       bool
}

func (res issueCertRes) Code() int {
	if res.issued {
		return http.StatusCreated
	}

	return http.StatusBadRequest
}

func (res issueCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res issueCertRes) Empty() bool {
	return false
}

type listCertsRes struct {
	Total        uint64        `json:"total"`
	Offset       uint64        `json:"offset,omitempty"`
	Limit        uint64        `json:"limit,omitempty"`
	Certificates []viewCertRes `json:"certificates,omitempty"`
}

func (res listCertsRes) Code() int {
	return http.StatusOK
}

func (res listCertsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listCertsRes) Empty() bool {
	return false
}

type viewCertRes struct {
	SerialNumber string    `json:"serial_number,omitempty"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked"`
	ExpiryTime   time.Time `json:"expiry_time,omitempty"`
	EntityID     string    `json:"entity_id,omitempty"`
}

func (res viewCertRes) Code() int {
	return http.StatusOK
}

func (res viewCertRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewCertRes) Empty() bool {
	return false
}

type crlRes struct {
	CrlBytes []byte `json:"crl"`
}

func (res crlRes) Code() int {
	return http.StatusOK
}

func (res crlRes) Headers() map[string]string {
	return map[string]string{}
}

func (res crlRes) Empty() bool {
	return false
}

type ocspRawRes struct {
	Data []byte `json:"-"`
}

func (res ocspRawRes) Code() int {
	return http.StatusOK
}

func (res ocspRawRes) Headers() map[string]string {
	return map[string]string{}
}

func (res ocspRawRes) Empty() bool {
	return false
}

type fileDownloadRes struct {
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
	CA          []byte `json:"ca"`
	Filename    string
	ContentType string
}

type issueFromCSRRes struct {
	SerialNumber string    `json:"serial_number"`
	Certificate  string    `json:"certificate,omitempty"`
	Revoked      bool      `json:"revoked"`
	ExpiryTime   time.Time `json:"expiry_time"`
	EntityID     string    `json:"entity_id"`
}

func (res issueFromCSRRes) Code() int {
	return http.StatusOK
}

func (res issueFromCSRRes) Headers() map[string]string {
	return map[string]string{}
}

func (res issueFromCSRRes) Empty() bool {
	return false
}
