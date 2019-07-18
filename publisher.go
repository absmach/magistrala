//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mainflux

import context "context"

// MessagePublisher specifies a message publishing API.
type MessagePublisher interface {
	// Publishes message to the stream. A non-nil error is returned to indicate
	// operation failure.
	Publish(context.Context, string, RawMessage) error
}
