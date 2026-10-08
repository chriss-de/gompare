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

	// structs of different types are a type mismatch unless the caller allows it
	if left.Type() != right.Type() && !c.config.allowDifferentStructs {
		if c.config.allowTypeMismatch {
			c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
			return nil
		}
		return pathError(ErrTypeMismatch, path)
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

		elemIdentifier, err := fieldElemIdentifier(c.config.tagName, field)
		if err != nil {
			return pathError(err, fieldPath)
		}

		if err := c.compareField(fieldPath, leftField, rightFieldName, elemIdentifier); err != nil {
			return err
		}
	}

	return nil
}

// fieldElemIdentifier returns the identifier template a struct field hands down to its elements, if any
func fieldElemIdentifier(tagName string, field reflect.StructField) (string, error) {
	opt, err := fieldIdentifier(tagName, field)
	if err != nil {
		return "", err
	}
	return opt.elemTemplate, nil
}

// missingCounterpart returns the value a field of a missing struct is compared against.
// Pointers and interfaces are compared against nil so a nil field produces no entry and a set one is
// expanded. Every other field is compared against reflect.Invalid so scalars are reported even if they
// hold their zero value, while slices, maps and structs list their elements/fields.
func missingCounterpart(field reflect.Value) reflect.Value {
	switch field.Kind() {
	case reflect.Ptr, reflect.Interface:
		return reflect.Zero(field.Type())
	default:
		return reflect.Value{}
	}
}

// cmpStructValuesForInvalid is used when one struct of cmpStruct was empty/invalid
// we use this to add all fields to the Differences
func (c *Comparer) cmpStructValuesForInvalid(dt DiffType, path []string, val reflect.Value) error {
	var nc *Comparer = c.clone()

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

		elemIdentifier, err := fieldElemIdentifier(c.config.tagName, field)
		if err != nil {
			return pathError(err, fieldPath)
		}

		// Differences are always produced as "missing on the left" and patched to dt below
		if err := nc.compareField(fieldPath, missing, valField, elemIdentifier); err != nil {
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
