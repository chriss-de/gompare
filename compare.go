package gompare

import (
	"fmt"
	"reflect"
	"text/template"
	"time"
)

// compareType is an enum of all supported compare types
type compareType uint8

const (
	typeUnsupported compareType = iota
	typeTime
	typeStruct
	typeSlice
	typeArray
	typeString
	typeBool
	typeInt
	typeUint
	typeFloat
	typeMap
	typePtr
	typeInterface
)

// compareFunc is the signature of the built-in compare functions
type compareFunc func([]string, reflect.Value, reflect.Value) error

// config holds config options of how to compare and present Differences
type config struct {
	tagName                   string // tagName defines wat struct tag name is used for our settings
	combinedIdentifierJoinSep rune   // combinedIdentifierJoinSep is the strings.Join separator for
	summarizeMissingStructs   bool   // summarizeMissingStructs adds the whole struct rather than every field to Differences
	sliceOrdering             bool   // sliceOrdering indicates if the slices we compare are ordered and elements for comparison are at the same index
	structMapKeys             bool   // structMapKeys allows complex struct keys to be encoded and represented in Differences path
	embeddedStructsAsFields   bool   // embeddedStructsAsFields if true will add EmbeddedStructs by struct name to Differences
	allowTypeMismatch         bool   // allowTypeMismatch reports values of different kinds as CHANGED instead of returning ErrTypeMismatch
	allowDifferentStructs     bool   // allowDifferentStructs compares structs of different types field by field instead of returning ErrTypeMismatch
	summarizeMissing          bool   // summarizeMissing adds one entry for a missing struct/slice/map instead of one per field/element
}

// Comparer a configurable compare instance. It is safe to use one Comparer from multiple goroutines.
type Comparer struct {
	config      config
	differences Differences
	// inProgress tracks the pointer/map/slice pairs that are currently being compared up the call stack.
	// It is shared between a comparison and its clones and is used to detect cycles.
	inProgress map[visitKey]struct{}
	// templates caches parsed identifier templates for one comparison run, shared with clones
	templates map[string]*template.Template
}

// visitKey identifies a pair of referencing values (pointer, map, slice) that is being compared
type visitKey struct {
	typ      reflect.Type
	left     uintptr
	right    uintptr
	leftLen  int
	rightLen int
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

// clone creates a new Comparer with the same config and an empty result. It takes part in the
// same comparison run as c and therefore shares the cycle detection state.
func (c *Comparer) clone() *Comparer {
	nc := &Comparer{
		config:     c.config,
		inProgress: c.inProgress,
		templates:  c.templates,
	}
	return nc
}

// getCompareFunc returns the fitting function to compare left and right
func (c *Comparer) getCompareFunc(left, right reflect.Value) (compareType, compareFunc) {
	switch {
	case areType(left, right, reflect.TypeOf(time.Time{})):
		return typeTime, c.cmpTime
	case areKind(left, right, reflect.Struct, reflect.Invalid):
		return typeStruct, c.cmpStruct
	case areKind(left, right, reflect.Slice, reflect.Invalid):
		return typeSlice, c.cmpSlice
	case areKind(left, right, reflect.Array, reflect.Invalid):
		return typeArray, c.cmpSlice
	case areKind(left, right, reflect.String, reflect.Invalid):
		return typeString, c.cmpString
	case areKind(left, right, reflect.Bool, reflect.Invalid):
		return typeBool, c.cmpBool
	case areKind(left, right, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Invalid):
		return typeInt, c.cmpInt
	case areKind(left, right, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Invalid):
		return typeUint, c.cmpUint
	case areKind(left, right, reflect.Float32, reflect.Float64, reflect.Invalid):
		return typeFloat, c.cmpFloat
	case areKind(left, right, reflect.Map, reflect.Invalid):
		return typeMap, c.cmpMap
	case areKind(left, right, reflect.Ptr, reflect.Invalid):
		return typePtr, c.cmpPtr
	case areKind(left, right, reflect.Interface, reflect.Invalid):
		return typeInterface, c.cmpInterface
	default:
		return typeUnsupported, nil
	}
}

// Compare returns Differences of all mutated values between left and right
func (c *Comparer) Compare(left, right any) (Differences, error) {
	// every call gets its own state so one Comparer can be used concurrently
	run := &Comparer{
		config:      c.config,
		differences: Differences{},
		inProgress:  make(map[visitKey]struct{}),
		templates:   make(map[string]*template.Template),
	}

	err := run.compare([]string{}, reflect.ValueOf(left), reflect.ValueOf(right))

	return run.differences, err
}

// compare is the internal compare functions. It compares left with right. This function gets also called
// from the internal compare functions and from struct and slices compares
func (c *Comparer) compare(path []string, left, right reflect.Value) error {
	// nothing on both sides - nothing to compare
	if left.Kind() == reflect.Invalid && right.Kind() == reflect.Invalid {
		return nil
	}

	// check if types match or areKind
	if !isValid(left, right) {
		if c.config.allowTypeMismatch {
			c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
			return nil
		}
		return pathError(ErrTypeMismatch, path)
	}

	// one side is missing and the other is a container: one entry for the whole value if configured
	if c.config.summarizeMissing && (left.Kind() == reflect.Invalid || right.Kind() == reflect.Invalid) {
		if left.Kind() == reflect.Invalid && isComposite(right) {
			c.differences.add(ADDED, path, nil, getAsAny(right))
			return nil
		}
		if right.Kind() == reflect.Invalid && isComposite(left) {
			c.differences.add(REMOVED, path, getAsAny(left), nil)
			return nil
		}
	}

	// cycle detection: if this pair of references is already being compared further up the
	// call stack we treat it as equal, otherwise we would recurse forever
	if key, ok := getVisitKey(left, right); ok {
		if _, seen := c.inProgress[key]; seen {
			return nil
		}
		c.inProgress[key] = struct{}{}
		defer delete(c.inProgress, key)
	}

	cmpType, compareFunc := c.getCompareFunc(left, right)

	if cmpType == typeUnsupported {
		kind := left.Kind()
		if kind == reflect.Invalid {
			kind = right.Kind()
		}
		return pathError(fmt.Errorf("%w: %s", ErrUnsupportedType, kind), path)
	}

	return compareFunc(path, left, right)
}

// isComposite reports if v is a struct (other than time.Time), slice, array or map - a value that is
// reported by its fields/elements when it is missing on one side
func isComposite(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct:
		return v.Type() != reflect.TypeOf(time.Time{})
	case reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

// cmpFromNil handles a nil pointer or interface on one side with a value on the other side.
// The present side is passed as is (pointer or interface content) so compare can detect cycles.
// A composite target is reported by its fields/elements (or as one entry with WithSummarizeMissing),
// any other target is one CHANGED entry holding the dereferenced value.
func (c *Comparer) cmpFromNil(path []string, left, right reflect.Value) error {
	present := left
	if present.Kind() == reflect.Invalid {
		present = right
	}

	target := getFinalValue(present)
	if isComposite(target) {
		return c.compare(path, left, right)
	}

	var leftVal, rightVal any
	if left.Kind() != reflect.Invalid {
		leftVal = getAsAny(getFinalValue(left))
	}
	if right.Kind() != reflect.Invalid {
		rightVal = getAsAny(getFinalValue(right))
	}
	c.differences.add(CHANGED, path, leftVal, rightVal)
	return nil
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

	return false, nil
}
