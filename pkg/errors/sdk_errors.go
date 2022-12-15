// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const err = "error"

var (
	// ErrJSONErrKey indicates response body did not contain erorr message.
	errJSONKey = New("response body expected error message json key not found")

	// ErrUnknown indicates that an unknown error was found in the response body.
	errUnknown = New("unknown error")
)

// SDKError is an error type for Mainflux SDK.
type SDKError interface {
	Error
	StatusCode() int
}

var _ SDKError = (*sdkError)(nil)

type sdkError struct {
	*customError
	statusCode int
}

func (ce *sdkError) Error() string {
	if ce == nil {
		return ""
	}
	if ce.customError == nil {
		return http.StatusText(ce.statusCode)
	}
	return fmt.Sprintf("Status: %s: %s", http.StatusText(ce.statusCode), ce.customError.Error())
}

func (ce *sdkError) StatusCode() int {
	return ce.statusCode
}

// NewSDKError returns an SDK Error that formats as the given text.
func NewSDKError(err error) SDKError {
	return &sdkError{
		customError: &customError{
			msg: err.Error(),
			err: nil,
		},
		statusCode: 0,
	}
}

// NewSDKErrorWithStatus returns an SDK Error setting the status code.
func NewSDKErrorWithStatus(err error, statusCode int) SDKError {
	return &sdkError{
		statusCode: statusCode,
		customError: &customError{
			msg: err.Error(),
			err: nil,
		},
	}
}

// CheckError will check the HTTP response status code and matches it with the given status codes.
// Since multiple status codes can be valid, we can pass multiple status codes to the function.
// The function then checks for errors in the HTTP response.
func CheckError(resp *http.Response, expectedStatusCodes ...int) SDKError {
	for _, expectedStatusCode := range expectedStatusCodes {
		if resp.StatusCode == expectedStatusCode {
			return nil
		}
	}

	var content map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return NewSDKErrorWithStatus(err, resp.StatusCode)
	}

	if msg, ok := content[err]; ok {
		if v, ok := msg.(string); ok {
			return NewSDKErrorWithStatus(errors.New(v), resp.StatusCode)
		}
		return NewSDKErrorWithStatus(errUnknown, resp.StatusCode)
	}

	return NewSDKErrorWithStatus(errJSONKey, resp.StatusCode)
}
