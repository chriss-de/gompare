package gompare

import (
	"errors"
	"reflect"
	"time"
)

// Type represents an enum with all the supported compare types
type Type uint8

const (
	UNSUPPORTED Type = iota
	TIME
	STRUCT
	SLICE
	ARRAY
	STRING
	BOOL
	INT
	UINT
	FLOAT
	MAP
	PTR
	INTERFACE
)

// CompareFunc represents the built-in compare functions
type CompareFunc func([]string, reflect.Value, reflect.Value) error

// config holds config options of how to compare and present Differences
type config struct {
	tagName                   string // tagName defines wat struct tag name is used for our settings
	combinedIdentifierJoinSep rune   // combinedIdentifierJoinSep is the strings.Join separator for
	summarizeMissingStructs   bool   // summarizeMissingStructs adds the whole struct rather than every field to Differences
	sliceOrdering             bool   // sliceOrdering indicates if the slices we compare are ordered and elements for comparison are at the same index
	structMapKeys             bool   // structMapKeys allows complex struct keys to be encoded and represented in Differences path
	embeddedStructsAsFields   bool   // embeddedStructsAsFields if true will add EmbeddedStructs by struct name to Differences
}

// Comparer a configurable compare instance
type Comparer struct {
	config      config
	differences Differences
}

// defaultConfig set the default config
func defaultConfig() config {
	return config{
		tagName:                   "cmp",
		combinedIdentifierJoinSep: '|',
		summarizeMissingStructs:   false,
		sliceOrdering:             false,
		structMapKeys:             false,
		embeddedStructsAsFields:   false,
	}
}

// NewComparer creates a new configurable diffing object
func NewComparer(opts ...CompareOptsFunc) (*Comparer, error) {
	d := Comparer{config: defaultConfig()}

	for _, opt := range opts {
		err := opt(&d)
		if err != nil {
			return nil, err
		}
	}

	return &d, nil
}

// clone clones Comparer to a new reference
func (c *Comparer) clone() *Comparer {
	nc := &Comparer{
		config: c.config,
	}
	return nc
}

// getCompareFunc returns the fitting function to compare left and right
func (c *Comparer) getCompareFunc(left, right reflect.Value) (Type, CompareFunc) {
	switch {
	case areType(left, right, reflect.TypeOf(time.Time{})):
		return TIME, c.cmpTime
	case areKind(left, right, reflect.Struct, reflect.Invalid):
		return STRUCT, c.cmpStruct
	case areKind(left, right, reflect.Slice, reflect.Invalid):
		return SLICE, c.cmpSlice
	case areKind(left, right, reflect.Array, reflect.Invalid):
		return ARRAY, c.cmpSlice
	case areKind(left, right, reflect.String, reflect.Invalid):
		return STRING, c.cmpString
	case areKind(left, right, reflect.Bool, reflect.Invalid):
		return BOOL, c.cmpBool
	case areKind(left, right, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Invalid):
		return INT, c.cmpInt
	case areKind(left, right, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Invalid):
		return UINT, c.cmpUint
	case areKind(left, right, reflect.Float32, reflect.Float64, reflect.Invalid):
		return FLOAT, c.cmpFloat
	case areKind(left, right, reflect.Map, reflect.Invalid):
		return MAP, c.cmpMap
	case areKind(left, right, reflect.Ptr, reflect.Invalid):
		return PTR, c.cmpPtr
	case areKind(left, right, reflect.Interface, reflect.Invalid):
		return INTERFACE, c.cmpInterface
	default:
		return UNSUPPORTED, nil
	}
}

// Compare returns Differences of all mutated values between left and right
func (c *Comparer) Compare(left, right any) (Differences, error) {
	// reset the state of the compare
	c.differences = Differences{}

	return c.differences, c.compare([]string{}, reflect.ValueOf(left), reflect.ValueOf(right))
}

// compare is the internal compare functions. It compares left with right. This function gets also called
// from the internal compare functions and from struct and slices compares
func (c *Comparer) compare(path []string, left, right reflect.Value) error {
	// check if types match or areKind
	if !isValid(left, right) {
		//if c.AllowTypeMismatch {
		//	c.differences.Add(CHANGED, path, left.Interface(), right.Interface())
		//	return nil
		//}
		return ErrTypeMismatch
	}

	cmpType, compareFunc := c.getCompareFunc(left, right)

	if cmpType == UNSUPPORTED {
		return errors.New("unsupported type: " + left.Kind().String())
	}

	return compareFunc(path, left, right)
}

// cmpDefault does basic compare operations and gets called from type specific compare functions
func (c *Comparer) cmpDefault(path []string, left, right reflect.Value) (changed bool, err error) {
	if left.Kind() == reflect.Invalid {
		c.differences.add(ADDED, path, nil, getAsAny(right))
		return true, nil
	}

	if right.Kind() == reflect.Invalid {
		c.differences.add(REMOVED, path, getAsAny(left), nil)
		return true, nil
	}

	if left.Kind() != right.Kind() {
		return false, ErrTypeMismatch
	}

	return false, nil
}
