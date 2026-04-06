// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
)

var (
	_ magistrala.Response = (*issueKeyRes)(nil)
	_ magistrala.Response = (*revokeKeyRes)(nil)
	_ magistrala.Response = (*retrieveKeyRes)(nil)
	_ magistrala.Response = (*retrieveJWKSRes)(nil)
)

type issueKeyRes struct {
	ID        string     `json:"id,omitempty"`
	Value     string     `json:"value,omitempty"`
	IssuedAt  time.Time  `json:"issued_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (res issueKeyRes) Code() int {
	return http.StatusCreated
}

func (res issueKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res issueKeyRes) Empty() bool {
	return res.Value == ""
}

type retrieveKeyRes struct {
	ID        string       `json:"id,omitempty"`
	IssuerID  string       `json:"issuer_id,omitempty"`
	Subject   string       `json:"subject,omitempty"`
	Type      auth.KeyType `json:"type,omitempty"`
	IssuedAt  time.Time    `json:"issued_at,omitempty"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
}

func (res retrieveKeyRes) Code() int {
	return http.StatusOK
}

func (res retrieveKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res retrieveKeyRes) Empty() bool {
	return false
}

type revokeKeyRes struct{}

func (res revokeKeyRes) Code() int {
	return http.StatusNoContent
}

func (res revokeKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeKeyRes) Empty() bool {
	return true
}

type retrieveJWKSRes struct {
	Keys                      []auth.PublicKeyInfo `json:"-"`
	CacheMaxAge               int                  `json:"-"`
	CacheStaleWhileRevalidate int                  `json:"-"`
}

func (res retrieveJWKSRes) MarshalJSON() ([]byte, error) {
	type jwksResponse struct {
		Keys []auth.PublicKeyInfo `json:"keys"`
	}

	return json.Marshal(jwksResponse{Keys: res.Keys})
}

func (res retrieveJWKSRes) Code() int {
	return http.StatusOK
}

func (res retrieveJWKSRes) Headers() map[string]string {
	cacheControl := fmt.Sprintf("public, max-age=%d, stale-while-revalidate=%d", res.CacheMaxAge, res.CacheStaleWhileRevalidate)
	headers := map[string]string{
		"Cache-Control": cacheControl,
	}

	return headers
}

func (res retrieveJWKSRes) Empty() bool {
	return false
}
