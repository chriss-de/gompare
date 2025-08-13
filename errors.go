package gompare

import "errors"

var (
	// ErrTypeMismatch Compared types do not match
	ErrTypeMismatch = errors.New("types do not match")
	// ErrInvalidChangeType The specified change values areKind not unsupported
	ErrInvalidChangeType = errors.New("diff type must be one of 'ADDED' or 'REMOVED'")
)
