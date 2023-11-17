// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/absmach/magistrala/users"
)

type emailerMock struct{}

// NewEmailer provides emailer instance for  the test.
func NewEmailer() users.Emailer {
	return &emailerMock{}
}

func (e *emailerMock) SendPasswordReset([]string, string, string, string) error {
	return nil
}
