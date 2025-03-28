// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

//go:generate mockery --name Emailer --output=./mocks --filename emailer.go --quiet --note "Copyright (c) Abstract Machines"
type Emailer interface {
	// SendEmailNotification sends an email to the recipients based on a trigger.
	SendEmailNotification(to []string, from, subject, header, user, content, footer string) error
}
