package message

import "errors"

var (
	ErrTooSmall                     = errors.New("too small bytes buffer")
	ErrInvalidOptionHeaderExt       = errors.New("invalid option header ext")
	ErrInvalidTokenLen              = errors.New("invalid token length")
	ErrInvalidValueLength           = errors.New("invalid value length")
	ErrShortRead                    = errors.New("invalid short read")
	ErrOptionTruncated              = errors.New("option truncated")
	ErrOptionUnexpectedExtendMarker = errors.New("option unexpected extend marker")
	ErrOptionsTooSmall              = errors.New("too small options buffer")
	ErrInvalidEncoding              = errors.New("invalid encoding")
	ErrOptionNotFound               = errors.New("option not found")
	ErrOptionDuplicate              = errors.New("duplicated option")
)
