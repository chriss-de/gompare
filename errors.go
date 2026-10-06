package gompare

import "errors"

var (
	// ErrTypeMismatch Compared types do not match
	ErrTypeMismatch = errors.New("types do not match")
	// ErrInvalidChangeType The specified change values areKind not unsupported
	ErrInvalidChangeType = errors.New("diff type must be one of 'ADDED' or 'REMOVED'")
	// ErrDuplicateIdentifier an identifier appears more than once within one slice
	ErrDuplicateIdentifier = errors.New("duplicate identifier")
	// ErrIdentifierTemplate an identifier template could not be parsed or executed
	ErrIdentifierTemplate = errors.New("invalid identifier template")
	// ErrUnsupportedType a value of a kind that cannot be compared (func, chan, complex, unsafe pointer)
	ErrUnsupportedType = errors.New("unsupported type")
	// ErrUnexportedField an unexported field was found but access to unexported fields is not possible
	// on this Go version (see checkUnexportedAccess)
	ErrUnexportedField = errors.New("cannot access unexported field")
)
