package gompare

import "errors"

var (
	// ErrTypeMismatch Compared types do not match
	ErrTypeMismatch = errors.New("types do not match")
	// ErrDuplicateIdentifier an identifier appears more than once within one slice
	ErrDuplicateIdentifier = errors.New("duplicate identifier")
	// ErrIdentifierTemplate an identifier template could not be parsed or executed
	ErrIdentifierTemplate = errors.New("invalid identifier template")
	// ErrUnsupportedType a value of a kind that cannot be compared (func, chan, complex, unsafe pointer)
	ErrUnsupportedType = errors.New("unsupported type")
	// ErrInvalidOption an option was given an invalid value
	ErrInvalidOption = errors.New("invalid option")
	// ErrUnexportedField an unexported field was found but access to unexported fields is not possible
	// on this Go version (see checkUnexportedAccess)
	ErrUnexportedField = errors.New("cannot access unexported field")
)
