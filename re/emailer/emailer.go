// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer

import (
	"github.com/absmach/magistrala/internal/email"
	"github.com/absmach/magistrala/re"
)

var _ re.Emailer = (*emailer)(nil)

type emailer struct {
	agent *email.Agent
}

func New(a *email.Config) (re.Emailer, error) {
	e, err := email.New(a)
	return &emailer{agent: e}, err
}

func (e *emailer) SendEmailNotification(to []string, from, subject, header, user, content, footer string) error {
	return e.agent.Send(to, from, subject, header, user, content, footer)
}
