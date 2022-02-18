// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

// BearerPrefix represents the token prefix for Bearer authentication scheme.
const BearerPrefix = "Bearer "

// ExtractAuthToken reads the value of request Authorization and removes the Bearer substring or returns error if it does not exist
func ExtractAuthToken(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")

	if !strings.HasPrefix(token, BearerPrefix) {
		return token, errors.ErrAuthentication
	}

	return strings.TrimPrefix(token, BearerPrefix), nil
}
