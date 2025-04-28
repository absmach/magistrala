// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

type Emailer interface {
	// SendEmailNotification sends an email to the recipients based on a trigger.
	SendEmailNotification(to []string, from, subject, header, user, content, footer string, attachments map[string][]byte) error
}
