// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const errorKey = "error"

var (
	// Failed to read response body.
	errRespBody = New("failed to read response body")
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewSDKErrorWithStatus(Wrap(errRespBody, err), resp.StatusCode)
	}
	var content map[string]interface{}
	_ = json.Unmarshal(body, &content)

	if msg, ok := content[errorKey]; ok {
		if v, ok := msg.(string); ok {
			return NewSDKErrorWithStatus(New(v), resp.StatusCode)
		}
		return NewSDKErrorWithStatus(fmt.Errorf("%v", msg), resp.StatusCode)
	}

	return NewSDKErrorWithStatus(New(string(body)), resp.StatusCode)
}
