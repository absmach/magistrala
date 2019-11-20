// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "github.com/mainflux/mainflux/errors"

// Emailer wrapper around the email
type Emailer interface {
	SendPasswordReset(To []string, host, token string) errors.Error
}
