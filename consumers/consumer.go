// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumers

import "context"

// AsyncConsumer specifies a non-blocking message-consuming API,
// which can be used for writing data to the DB, publishing messages
// to broker, sending notifications, or any other asynchronous job.
type AsyncConsumer interface {
	// ConsumeAsync method is used to asynchronously consume received messages.
	ConsumeAsync(ctx context.Context, messages interface{})

	// Errors method returns a channel for reading errors which occur during async writes.
	// Must be  called before performing any writes for errors to be collected.
	// The channel is buffered(1) so it allows only 1 error without blocking if not drained.
	// The channel may receive nil error to indicate success.
	Errors() <-chan error
}

// BlockingConsumer specifies a blocking message-consuming API,
// which can be used for writing data to the DB, publishing messages
// to broker, sending notifications... BlockingConsumer implementations
// might also support concurrent use, but consult implementation for more details.
type BlockingConsumer interface {
	// ConsumeBlocking method is used to consume received messages synchronously.
	// A non-nil error is returned to indicate operation failure.
	ConsumeBlocking(ctx context.Context, messages interface{}) error
}
