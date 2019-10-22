package errors

import (
	pkg_errors "github.com/pkg/errors"
)

// Prefix is the default error string prefix
const Prefix = "opcua: "

// Errorf is a wrapper for `errors.Errorf`
func Errorf(format string, a ...interface{}) error {
	return pkg_errors.Errorf(Prefix+format, a...)
}

// New is a wrapper for `errors.New`
func New(text string) error {
	return pkg_errors.New(Prefix + text)
}
