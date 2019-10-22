// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

// Emailer wrapper around the email
type Emailer interface {
	SendPasswordReset(To []string, host, token string) error
}
