// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package transformers

import "github.com/absmach/magistrala/pkg/messaging"

// Transformer specifies API form Message transformer.
type Transformer interface {
	// Transform Magistrala message to any other format.
	Transform(msg *messaging.Message) (interface{}, error)
}
