// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package transformers

import "github.com/absmach/magistrala/pkg/messaging"

// Transformer specifies API form Message transformer.
type Transformer interface {
	// Transform Magistrala message to any other format.
	Transform(msg *messaging.Message) (interface{}, error)
}

type number interface {
	uint64 | int64 | float64
}

// ToUnixNano converts time to UnixNano time format.
func ToUnixNano[N number](t N) N {
	switch {
	case t == 0:
		return 0
	case t >= 1e18: // Check if the value is in nanoseconds
		return t
	case t >= 1e15 && t < 1e18: // Check if the value is in milliseconds
		return t * 1e3
	case t >= 1e12 && t < 1e15: // Check if the value is in microseconds
		return t * 1e6
	default: // Assume it's in seconds (Unix time)
		return t * 1e9
	}
}
