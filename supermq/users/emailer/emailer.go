// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package emailer

import (
	"fmt"

	"github.com/absmach/supermq/internal/email"
	"github.com/absmach/supermq/users"
)

var _ users.Emailer = (*emailer)(nil)

type emailer struct {
	resetURL string
	agent    *email.Agent
}

// New creates new emailer utility.
func New(url string, c *email.Config) (users.Emailer, error) {
	e, err := email.New(c)
	return &emailer{resetURL: url, agent: e}, err
}

func (e *emailer) SendPasswordReset(to []string, host, user, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", host, e.resetURL, token)
	return e.agent.Send(to, "", "Password Reset Request", "", user, url, "")
}
