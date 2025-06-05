// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer

import (
	"github.com/absmach/magistrala/internal/email"
)

var _ Emailer = (*emailer)(nil)

type Emailer interface {
	// SendEmailNotification sends an email to the recipients based on a trigger.
	SendEmailNotification(to []string, from, subject, header, user, content, footer string, attachments map[string][]byte) error
}

type emailer struct {
	agent *email.Agent
}

func New(a *email.Config) (Emailer, error) {
	e, err := email.New(a)
	return &emailer{agent: e}, err
}

func (e *emailer) SendEmailNotification(to []string, from, subject, header, user, content, footer string, attachments map[string][]byte) error {
	return e.agent.Send(to, from, subject, header, user, content, footer, attachments)
}
