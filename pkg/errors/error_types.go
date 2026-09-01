// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

const internalServiceError = "internal server error"

type NestError interface {
	Error
	Embed(e error) error
}

var _ NestError = (*customError)(nil)

func (e *customError) Embed(err error) error {
	if err == nil {
		return e
	}

	return &customError{
		msg: e.msg,
		err: Wrap(err, e.err),
	}
}

type nestableError interface {
	NestError
	isNestable()
}

type RequestError struct {
	customError
}

var _ nestableError = (*RequestError)(nil)

func NewRequestError(message string) NestError {
	return &RequestError{
		customError: newCustomError(message),
	}
}

func NewRequestErrorWithErr(message string, err error) NestError {
	return &RequestError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (e *RequestError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &RequestError{
		customError: *embedded.(*customError),
	}
}

func (*RequestError) isNestable() {}

type AuthNError struct {
	customError
}

var _ nestableError = (*AuthNError)(nil)

func NewAuthNError(message string) NestError {
	return &AuthNError{
		customError: newCustomError(message),
	}
}

func NewAuthNErrorWithErr(message string, err error) NestError {
	return &AuthNError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (e *AuthNError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &AuthNError{
		customError: *embedded.(*customError),
	}
}

func (*AuthNError) isNestable() {}

var _ nestableError = (*AuthZError)(nil)

type AuthZError struct {
	customError
}

func (e *AuthZError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &AuthZError{
		customError: *embedded.(*customError),
	}
}

func NewAuthZError(message string) NestError {
	return &AuthZError{
		customError: newCustomError(message),
	}
}

func NewAuthZErrorWithErr(message string, err error) NestError {
	return &AuthZError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (*AuthZError) isNestable() {}

type InternalError struct {
	customError
}

var _ nestableError = (*InternalError)(nil)

func NewInternalError() error {
	return &InternalError{
		customError: newCustomError(internalServiceError),
	}
}

func NewInternalErrorWithErr(err error) NestError {
	return &InternalError{
		customError: newCustomErrorWithError(internalServiceError, err),
	}
}

func (e *InternalError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &InternalError{
		customError: *embedded.(*customError),
	}
}

func (*InternalError) isNestable() {}

type ServiceError struct {
	customError
}

var _ nestableError = (*ServiceError)(nil)

func NewServiceError(message string) NestError {
	return &ServiceError{
		customError: newCustomError(message),
	}
}

func NewServiceErrorWithErr(message string, err error) NestError {
	return &ServiceError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (e *ServiceError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &ServiceError{
		customError: *embedded.(*customError),
	}
}

func (*ServiceError) isNestable() {}

type MediaTypeError struct {
	customError
}

var _ nestableError = (*MediaTypeError)(nil)

func NewMediaTypeError(message string) NestError {
	return &MediaTypeError{
		customError: newCustomError(message),
	}
}

func NewMediaTypeErrorWithErr(message string, err error) NestError {
	return &MediaTypeError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (e *MediaTypeError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &MediaTypeError{
		customError: *embedded.(*customError),
	}
}

func (*MediaTypeError) isNestable() {}

type NotFoundError struct {
	customError
}

var _ nestableError = (*NotFoundError)(nil)

func NewNotFoundError(message string) NestError {
	return &NotFoundError{
		customError: newCustomError(message),
	}
}

func NewNotFoundErrorWithErr(message string, err error) NestError {
	return &NotFoundError{
		customError: newCustomErrorWithError(message, err),
	}
}

func (e *NotFoundError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &NotFoundError{
		customError: *embedded.(*customError),
	}
}

func (*NotFoundError) isNestable() {}

type ConflictError struct {
	customError
}

var _ nestableError = (*ConflictError)(nil)

func NewConflictError(message string) NestError {
	return &ConflictError{
		customError: newCustomError(message),
	}
}

func (e *ConflictError) Embed(err error) error {
	embedded := e.customError.Embed(err)
	return &ConflictError{
		customError: *embedded.(*customError),
	}
}

func (*ConflictError) isNestable() {}
