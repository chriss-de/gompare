package gompare

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"unsafe"
)

// isExportFlag flag value to 'export' a struct field even if private
var isExportFlag uintptr = (1 << 5) | (1 << 6)

// getAsAny returns v's current value as any. It is equivalent to:
// var i any = (v's underlying value)
func getAsAny(v reflect.Value) any {
	// check if we can access the field
	// fake export it if it is unexported
	if !v.CanInterface() {
		flagTmp := (*uintptr)(unsafe.Pointer(uintptr(unsafe.Pointer(&v)) + 2*unsafe.Sizeof(uintptr(0))))
		*flagTmp = (*flagTmp) & (^isExportFlag)
	}
	return v.Interface()
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

// areType checks if left and right are of reflect.Type of the types listed
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

	return leftMatch && rightMatch
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

// getIdentifier returns the identifier for a struct
func getIdentifier(tag string, v reflect.Value, joinSep string) any {
	if v.Kind() != reflect.Struct {
		return nil
	}

	var combinedIdentifierTemplate string
	var combinedIdentifier map[string]reflect.Value = make(map[string]reflect.Value)

	for i := 0; i < v.NumField(); i++ {
		if hto, toValue := hasTagOption(tag, v.Type().Field(i), "identifier"); hto {
			combinedIdentifier[v.Type().Field(i).Name] = v.Field(i)
			if toValue != "" {
				if combinedIdentifierTemplate == "" {
					combinedIdentifierTemplate = toValue
				} else if combinedIdentifierTemplate != toValue {
					panic("identifier name must be identical")
				}
			}
		}
	}

	switch len(combinedIdentifier) {
	case 0:
		return nil
	case 1:
		for identifier := range maps.Values(combinedIdentifier) {
			return identifier.Interface()
		}
		return nil
	default:
		if combinedIdentifierTemplate == "" {
			var combinedIdentifierTemplatePrepare []string
			for _, k := range slices.Sorted(maps.Keys(combinedIdentifier)) {
				combinedIdentifierTemplatePrepare = append(combinedIdentifierTemplatePrepare, "{{."+k+"}}")
			}
			combinedIdentifierTemplate = strings.Join(combinedIdentifierTemplatePrepare, joinSep)
		}
		templatedIdentifier := template.Must(template.New("id").Parse(combinedIdentifierTemplate))
		templatedIdentifierOutput := bytes.NewBuffer(nil)

		if err := templatedIdentifier.Execute(templatedIdentifierOutput, combinedIdentifier); err != nil {
			panic("failed to execute template: " + err.Error())
		}
		return templatedIdentifierOutput.String()

	}
}

// hasTagOption checks if a struct field has a given tag option
// return true/false and the option value if true
func hasTagOption(tag string, f reflect.StructField, opt string) (bool, string) {
	parts := strings.Split(f.Tag.Get(tag), ",")
	if len(parts) < 2 {
		return false, ""
	}

	for _, option := range parts[1:] {
		tagOption := strings.Split(option, ":")
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
	default:
		if useComplex {
			bWriter := new(bytes.Buffer)
			if err := gob.NewEncoder(bWriter).Encode(v); err != nil {
				panic(err)
			}
			return string(bWriter.Bytes())
		} else {
			return fmt.Sprint(v)
		}
	}
}

// isPartOfPath checks if pathTest is a sub path of pathOrig
func isPartOfPath(pathOrig, pathTest []string) bool {
	if len(pathOrig) == 0 {
		return true
	}
	if strings.HasPrefix(strings.Join(pathTest, "|"), strings.Join(pathOrig, "|")) {
		return true
	}
	return false
}
