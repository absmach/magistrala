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
	resetURL        string
	verificationURL string
	resetAgent      *email.Agent
	verifyAgent     *email.Agent
}

// New creates new emailer utility.
func New(resetURL, verificationURL string, resetConfig, verifyConfig *email.Config) (users.Emailer, error) {
	resetAgent, err := email.New(resetConfig)
	if err != nil {
		return nil, err
	}

	verifyAgent, err := email.New(verifyConfig)
	if err != nil {
		return nil, err
	}

	return &emailer{
		resetURL:        resetURL,
		verificationURL: verificationURL,
		resetAgent:      resetAgent,
		verifyAgent:     verifyAgent,
	}, nil
}

func (e *emailer) SendPasswordReset(to []string, user, token string) error {
	url := fmt.Sprintf("%s?token=%s", e.resetURL, token)
	return e.resetAgent.Send(to, "", "Password Reset Request", "", user, url, "", nil)
}

func (e *emailer) SendVerification(to []string, user, verificationToken string) error {
	url := fmt.Sprintf("%s?token=%s", e.verificationURL, verificationToken)
	return e.verifyAgent.Send(to, "", "Email Verification", "", user, url, "", nil)
}
