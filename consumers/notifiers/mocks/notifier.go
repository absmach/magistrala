// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/absmach/magistrala/consumers/notifiers"
	"github.com/absmach/magistrala/pkg/messaging"
)

var _ notifiers.Notifier = (*notifier)(nil)

const InvalidSender = "invalid@example.com"

type notifier struct{}

// NewNotifier returns a new Notifier mock.
func NewNotifier() notifiers.Notifier {
	return notifier{}
}

func (n notifier) Notify(from string, to []string, msg *messaging.Message) error {
	for _, t := range to {
		if t == InvalidSender {
			return notifiers.ErrNotify
		}
	}
	return nil
}
