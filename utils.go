package gompare

import (
	"bytes"
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
var unexportedAccessOK = checkUnexportedAccess()

// checkUnexportedAccess verifies at startup that getAsAny can read an unexported field
func checkUnexportedAccess() (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()

	type probe struct{ v int }
	field := reflect.ValueOf(probe{v: 42}).Field(0)
	if field.CanInterface() {
		return true
	}

	val, isInt := forceExport(field).Interface().(int)
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

// getVisitKey returns a key identifying the pair left/right if both are non-nil references
// (pointer, map or slice) of the same type. Those are the only values that can form cycles.
func getVisitKey(left, right reflect.Value) (visitKey, bool) {
	if left.Kind() != right.Kind() || left.Type() != right.Type() {
		return visitKey{}, false
	}

	switch left.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
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
	default:
		return visitKey{}, false
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

	parts := strings.Split(t, ",")
	if len(parts) < 1 {
		return "-"
	}

	return parts[0]
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

// getID returns an ID for a given value
// basically converts value according to its type or - if configured - encode it with gob
func getID(val any, useComplex bool) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case indexKey:
		return strconv.Itoa(int(v))
	default:
		if useComplex {
			bWriter := new(bytes.Buffer)
			if err := gob.NewEncoder(bWriter).Encode(v); err != nil {
				panic(err)
			}
			return bWriter.String()
		} else {
			return fmt.Sprint(v)
		}
	}
}
