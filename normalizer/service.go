//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package normalizer

import "github.com/mainflux/mainflux"

// Service specifies API for normalizing messages.
type Service interface {
	// Normalizes raw message to array of standard SenML messages.
	Normalize(mainflux.RawMessage) (NormalizedData, error)
}

// NormalizedData contains normalized messages and their content type.
type NormalizedData struct {
	ContentType string
	Messages    []mainflux.Message
}
