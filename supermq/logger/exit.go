// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger

import "os"

// ExitWithError closes the current process with error code.
func ExitWithError(code *int) {
	os.Exit(*code)
}
