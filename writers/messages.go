// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writers

// MessageRepository specifies message writing API.
type MessageRepository interface {
	// Save method is used to save published message. A non-nil
	// error is returned to indicate  operation failure.
	Save(messages interface{}) error
}
