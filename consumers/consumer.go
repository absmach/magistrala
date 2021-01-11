// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

// Consumer specifies message consuming API.
type Consumer interface {
	// Consume method is used to consumed received messages.
	// A non-nil error is returned to indicate operation failure.
	Consume(messages interface{}) error
}
