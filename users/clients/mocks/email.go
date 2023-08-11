// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/mainflux/mainflux/users/clients"
)

type emailerMock struct{}

// NewEmailer provides emailer instance for  the test.
func NewEmailer() clients.Emailer {
	return &emailerMock{}
}

func (e *emailerMock) SendPasswordReset([]string, string, string, string) error {
	return nil
}
