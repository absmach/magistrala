// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package apiutil

import (
	"net/http"
	"strings"
)

// BearerPrefix represents the token prefix for Bearer authentication scheme.
const BearerPrefix = "Bearer "

// ExtractBearerToken returns value of the bearer token. If there is no bearer token - an empty value is returned.
func ExtractBearerToken(r *http.Request) string {
	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, BearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(token, BearerPrefix)
}
