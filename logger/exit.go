package logger

import "os"

// ExitWithError closes the current process with error code.
func ExitWithError(code *int) {
	os.Exit(*code)
}
