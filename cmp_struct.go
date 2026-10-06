package gompare

import "reflect"

// cmpStruct compare's two structs
func (c *Comparer) cmpStruct(path []string, left, right reflect.Value) error {
	if left.Kind() == reflect.Invalid {
		// if summarizeMissingStructs note Difference as the whole struct
		// otherwise note every field
		if c.config.summarizeMissingStructs {
			c.differences.add(ADDED, path, nil, getAsAny(right))
			return nil
		} else {
			return c.cmpStructValuesForInvalid(ADDED, path, right)
		}
	}

	if right.Kind() == reflect.Invalid {
		if c.config.summarizeMissingStructs {
			c.differences.add(REMOVED, path, getAsAny(left), nil)
			return nil
		} else {
			return c.cmpStructValuesForInvalid(REMOVED, path, left)
		}
	}

	for i := 0; i < left.NumField(); i++ {
		field := left.Type().Field(i)
		tName := getTagName(c.config.tagName, field)

		// if tag is `-` we skip it
		if tName == "-" {
			continue
		}

		if tName == "" {
			tName = field.Name
		}

		leftField := left.Field(i)
		rightFieldName := right.FieldByName(field.Name)

		fieldPath := path

		// always add path - except when field is embedded/anonymous AND embeddedStructsAsFields is false (default)
		if !(field.Anonymous && !c.config.embeddedStructsAsFields) {
			fieldPath = copyAppend(fieldPath, tName)
		}

		// unexported fields are compared as well - getAsAny takes care of accessing them
		if !leftField.CanInterface() && !unexportedAccessOK {
			return pathError(ErrUnexportedField, fieldPath)
		}

		if err := c.compare(fieldPath, leftField, rightFieldName); err != nil {
			return err
		}
	}

	return nil
}

// missingCounterpart returns the value a field of a missing struct is compared against.
// Containers (slice, array, map, pointer, interface) are compared against their zero value so their
// elements get listed one by one as everywhere else. Every other field is compared against
// reflect.Invalid so it is reported even if it holds its zero value.
func missingCounterpart(field reflect.Value) reflect.Value {
	switch field.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Ptr, reflect.Interface:
		return reflect.Zero(field.Type())
	default:
		return reflect.Value{}
	}
}

// cmpStructValuesForInvalid is used when one struct of cmpStruct was empty/invalid
// we use this to add all fields to the Differences
func (c *Comparer) cmpStructValuesForInvalid(dt DiffType, path []string, val reflect.Value) error {
	var nc *Comparer = c.clone()

	if dt != ADDED && dt != REMOVED {
		return ErrInvalidChangeType
	}

	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}

	if val.Kind() != reflect.Struct {
		return ErrTypeMismatch
	}

	for idx := 0; idx < val.NumField(); idx++ {
		field := val.Type().Field(idx)
		tName := getTagName(c.config.tagName, field)

		// if tag is `-` we skip it
		if tName == "-" {
			continue
		}

		if tName == "" {
			tName = field.Name
		}

		valField := val.Field(idx)
		missing := missingCounterpart(valField)

		fieldPath := path

		// always add path - except when field is embedded/anonymous AND embeddedStructsAsFields is false (default)
		if !(field.Anonymous && !c.config.embeddedStructsAsFields) {
			fieldPath = copyAppend(fieldPath, tName)
		}

		// unexported fields are compared as well - getAsAny takes care of accessing them
		if !valField.CanInterface() && !unexportedAccessOK {
			return pathError(ErrUnexportedField, fieldPath)
		}

		// Differences are always produced as "missing on the left" and patched to dt below
		if err := nc.compare(fieldPath, missing, valField); err != nil {
			return err
		}
	}

	// we have to patch Differences to get the right Difference result
	for i := 0; i < len(nc.differences); i++ {
		diff := nc.differences[i]
		diff.patchLeftAndRight(dt)
		c.differences = append(c.differences, diff)
	}

	return nil
}
