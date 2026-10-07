package gompare

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unsafe"
)

// isExportFlag flag value to 'export' a struct field even if private
var isExportFlag uintptr = (1 << 5) | (1 << 6)

// getAsAny relies on reflect.Value being exactly three words: type, pointer, flag.
// If a Go release changes that, these two declarations fail to compile (negative array length).
var (
	_ [unsafe.Sizeof(reflect.Value{}) - 3*unsafe.Sizeof(uintptr(0))]struct{}
	_ [3*unsafe.Sizeof(uintptr(0)) - unsafe.Sizeof(reflect.Value{})]struct{}
)

// unexportedAccessOK is the result of the startup self test of getAsAny.
// If it is false, Compare returns ErrUnexportedField when it meets an unexported field.
var unexportedAccessOK = checkUnexportedAccess(forceExport)

// checkUnexportedAccess verifies at startup that export (forceExport) makes an unexported field readable
func checkUnexportedAccess(export func(reflect.Value) reflect.Value) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	type probe struct{ v int }
	field := reflect.ValueOf(probe{v: 42}).Field(0)

	val, isInt := export(field).Interface().(int)
	return isInt && val == 42
}

// forceExport returns a copy of v with the read-only flags cleared so Interface() works on unexported fields
func forceExport(v reflect.Value) reflect.Value {
	flagTmp := (*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + 2*unsafe.Sizeof(uintptr(0))))
	*flagTmp = (*flagTmp) & (^isExportFlag)
	return v
}

// getAsAny returns v's current value as any. It is equivalent to:
// var i any = (v's underlying value)
func getAsAny(v reflect.Value) any {
	// check if we can access the field
	// fake export it if it is unexported
	if !v.CanInterface() {
		if !unexportedAccessOK {
			return nil
		}
		v = forceExport(v)
	}
	return v.Interface()
}

// pathError wraps err with the path where it happened
func pathError(err error, path []string) error {
	return fmt.Errorf("%w at path %q", err, strings.Join(path, "/"))
}

// areKind checks if left and right are of reflect.Kind of the kinds listed
func areKind(left, right reflect.Value, kinds ...reflect.Kind) bool {
	var leftMatch, rightMatch bool

	for _, kind := range kinds {
		if left.Kind() == kind {
			leftMatch = true
		}
		if right.Kind() == kind {
			rightMatch = true
		}
	}

	return leftMatch && rightMatch
}

// areType checks if left and right are of reflect.Type of the types listed.
// One side may be reflect.Invalid (missing) as long as the other side is of a listed type.
func areType(left, right reflect.Value, types ...reflect.Type) bool {
	var leftMatch, rightMatch bool

	for _, t := range types {
		if left.Kind() != reflect.Invalid {
			if left.Type() == t {
				leftMatch = true
			}
		}
		if right.Kind() != reflect.Invalid {
			if right.Type() == t {
				rightMatch = true
			}
		}
	}

	if left.Kind() == reflect.Invalid {
		return rightMatch
	}
	if right.Kind() == reflect.Invalid {
		return leftMatch
	}

	return leftMatch && rightMatch
}

// getVisitKey returns a key identifying the pair left/right if they are non-nil references
// (pointer, map or slice) of the same type. Those are the only values that can form cycles.
// If one side is missing (reflect.Invalid) the key holds the present reference only, so expanding
// a missing cyclic structure terminates as well.
func getVisitKey(left, right reflect.Value) (visitKey, bool) {
	if left.Kind() == reflect.Invalid {
		return oneSidedVisitKey(right, false)
	}
	if right.Kind() == reflect.Invalid {
		return oneSidedVisitKey(left, true)
	}

	if left.Kind() != right.Kind() || left.Type() != right.Type() || !isReference(left) {
		return visitKey{}, false
	}
	if left.IsNil() || right.IsNil() {
		return visitKey{}, false
	}

	return visitKey{
		typ:      left.Type(),
		left:     left.Pointer(),
		right:    right.Pointer(),
		leftLen:  refLen(left),
		rightLen: refLen(right),
	}, true
}

// oneSidedVisitKey returns the visit key for a reference whose counterpart is missing
func oneSidedVisitKey(v reflect.Value, isLeft bool) (visitKey, bool) {
	if !isReference(v) || v.IsNil() {
		return visitKey{}, false
	}

	key := visitKey{typ: v.Type()}
	if isLeft {
		key.left, key.leftLen = v.Pointer(), refLen(v)
	} else {
		key.right, key.rightLen = v.Pointer(), refLen(v)
	}
	return key, true
}

// isReference reports if v is a pointer, map or slice
func isReference(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
		return true
	default:
		return false
	}
}

// refLen returns the length of a slice, 0 for other kinds
func refLen(v reflect.Value) int {
	if v.Kind() == reflect.Slice {
		return v.Len()
	}
	return 0
}

// isValid returns true if left and right are of same Kind OR if left OR right are of reflect.Invalid
func isValid(left, right reflect.Value) bool {
	if left.Kind() == right.Kind() {
		return true
	}

	if left.Kind() == reflect.Invalid || right.Kind() == reflect.Invalid {
		return true
	}

	return false
}

// copyAppend copies src to a new slice and appends elems to it
func copyAppend(src []string, elems ...string) []string {
	dst := make([]string, len(src)+len(elems))
	copy(dst, src)
	for i := len(src); i < len(src)+len(elems); i++ {
		dst[i] = elems[i-len(src)]
	}
	return dst
}

// getTagName return the 'value' of the tag of a struct field. The value of the `cmp` tag
func getTagName(tag string, f reflect.StructField) string {
	t := f.Tag.Get(tag)

	return strings.Split(t, ",")[0]
}

// getIdentifier returns the identifier for a struct or nil if the struct has none.
// A single identifier field is returned as its value. Several identifier fields are rendered
// through a template: either the one given in the tag or all field names joined by the configured separator.
// The template data holds every identifier value by its Go field name and by its tag name.
func (c *Comparer) getIdentifier(v reflect.Value) (any, error) {
	if v.Kind() != reflect.Struct {
		return nil, nil
	}

	var (
		templateText string
		fieldNames   []string
		data         = make(map[string]any)
		single       any
	)

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		hto, toValue := hasTagOption(c.config.tagName, field, "identifier")
		if !hto {
			continue
		}

		if !v.Field(i).CanInterface() && !unexportedAccessOK {
			return nil, ErrUnexportedField
		}

		single = getAsAny(v.Field(i))
		fieldNames = append(fieldNames, field.Name)
		data[field.Name] = single
		if tName := getTagName(c.config.tagName, field); tName != "" && tName != "-" {
			data[tName] = single
		}

		if toValue != "" {
			if templateText == "" {
				templateText = toValue
			} else if templateText != toValue {
				return nil, fmt.Errorf("%w: identifier fields of %s use different templates %q and %q",
					ErrIdentifierTemplate, v.Type(), templateText, toValue)
			}
		}
	}

	switch len(fieldNames) {
	case 0:
		return nil, nil
	case 1:
		return single, nil
	}

	if templateText == "" {
		slices.Sort(fieldNames)
		for i, name := range fieldNames {
			fieldNames[i] = "{{." + name + "}}"
		}
		templateText = strings.Join(fieldNames, string(c.config.combinedIdentifierJoinSep))
	}

	tmpl, err := c.identifierTemplate(templateText)
	if err != nil {
		return nil, fmt.Errorf("%w: %q of %s: %v", ErrIdentifierTemplate, templateText, v.Type(), err)
	}

	out := bytes.NewBuffer(nil)
	if err := tmpl.Execute(out, data); err != nil {
		return nil, fmt.Errorf("%w: %q of %s: %v", ErrIdentifierTemplate, templateText, v.Type(), err)
	}

	return out.String(), nil
}

// identifierTemplate parses an identifier template once per comparison run
func (c *Comparer) identifierTemplate(text string) (*template.Template, error) {
	if tmpl, ok := c.templates[text]; ok {
		return tmpl, nil
	}

	tmpl, err := template.New("identifier").Option("missingkey=error").Parse(text)
	if err != nil {
		return nil, err
	}

	c.templates[text] = tmpl
	return tmpl, nil
}

// hasTagOption checks if a struct field has a given tag option
// return true/false and the option value if true
func hasTagOption(tag string, f reflect.StructField, opt string) (bool, string) {
	parts := strings.Split(f.Tag.Get(tag), ",")
	if len(parts) < 2 {
		return false, ""
	}

	for _, option := range parts[1:] {
		// split on the first ':' only - the option value (e.g. a template) may contain ':'
		tagOption := strings.SplitN(option, ":", 2)
		if len(tagOption) == 0 || tagOption[0] != opt {
			continue
		}
		if len(tagOption) == 2 {
			return true, tagOption[1]
		}
		return true, ""
	}

	return false, ""
}

// sortedMapKeys returns the keys of map m in a stable order so Differences are deterministic.
// Integer keys sort numerically, all other keys by their string representation.
func sortedMapKeys(m reflect.Value) []reflect.Value {
	keys := m.MapKeys()

	sort.SliceStable(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		switch {
		case areKind(a, b, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64):
			return a.Int() < b.Int()
		case areKind(a, b, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64):
			return a.Uint() < b.Uint()
		case areKind(a, b, reflect.String):
			return a.String() < b.String()
		default:
			return fmt.Sprint(getAsAny(a)) < fmt.Sprint(getAsAny(b))
		}
	})

	return keys
}

// getFinalValue dereferences v to final reflect.Value
func getFinalValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		return getFinalValue(v.Elem())
	case reflect.Ptr:
		return getFinalValue(reflect.Indirect(v))
	default:
		return v
	}
}

// hasAtSameIndex checks if val is at idx in slice
func hasAtSameIndex(slice, val reflect.Value, idx int) bool {
	if idx < slice.Len() {
		x := slice.Index(idx)
		return reflect.DeepEqual(getAsAny(x), getAsAny(val))
	}
	return false
}

// Compare returns a changelog of all mutated values from both
func Compare(left, right any, opts ...CompareOptsFunc) (Differences, error) {
	c, err := NewComparer(opts...)
	if err != nil {
		return nil, err
	}
	return c.Compare(left, right)
}

// getID returns the path element for a map key or slice identifier.
// Strings, integers, floats and bools are rendered as text. Other values are rendered with fmt.Sprint,
// or - if useComplex is set - encoded with gob and base64 so they survive as a path element.
func getID(val any, useComplex bool) (string, error) {
	if idx, ok := val.(indexKey); ok {
		return strconv.Itoa(int(idx)), nil
	}

	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.String:
		return rv.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), nil
	}

	if !useComplex {
		return fmt.Sprint(val), nil
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(val); err != nil {
		return "", fmt.Errorf("cannot encode map key of type %T: %w", val, err)
	}
	return base64.RawStdEncoding.EncodeToString(buf.Bytes()), nil
}
