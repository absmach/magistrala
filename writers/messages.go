//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package writers

import "github.com/mainflux/mainflux"

// MessageRepository specifies message writing API.
type MessageRepository interface {

	// Save method is used to save published message. A non-nil
	// error is returned to indicate  operation failure.
	Save(mainflux.Message) error
}
