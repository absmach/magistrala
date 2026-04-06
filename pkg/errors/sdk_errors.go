// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type errorRes struct {
	Err string `json:"error"`
	Msg string `json:"message"`
}

// Failed to read response body.
var errRespBody = New("failed to read response body")

// SDKError is an error type for Magistrala SDK.
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
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return &sdkError{
			statusCode: 0,
			customError: &customError{
				msg: e.Msg(),
				err: cast(e.Err()),
			},
		}
	}
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
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return &sdkError{
			statusCode: statusCode,
			customError: &customError{
				msg: e.Msg(),
				err: cast(e.Err()),
			},
		}
	}
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
	if resp == nil {
		return nil
	}

	for _, expectedStatusCode := range expectedStatusCodes {
		if resp.StatusCode == expectedStatusCode {
			return nil
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewSDKErrorWithStatus(Wrap(errRespBody, err), resp.StatusCode)
	}
	var content errorRes
	if err := json.Unmarshal(body, &content); err != nil {
		return NewSDKErrorWithStatus(err, resp.StatusCode)
	}
	if content.Err == "" {
		return NewSDKErrorWithStatus(New(content.Msg), resp.StatusCode)
	}

	return NewSDKErrorWithStatus(Wrap(New(content.Msg), New(content.Err)), resp.StatusCode)
}
