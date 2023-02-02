// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"errors"

	"github.com/mainflux/mainflux/pkg/messaging"
)

// ErrNotify wraps sending notification errors,
var ErrNotify = errors.New("Error sending notification")

// Notifier represents an API for sending notification.
type Notifier interface {
	// Notify method is used to send notification for the
	// received message to the provided list of receivers.
	Notify(from string, to []string, msg *messaging.Message) error
}
