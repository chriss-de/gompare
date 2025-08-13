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

		// skip private fields
		if !left.CanInterface() {
			continue
		}

		if err := c.compare(fieldPath, leftField, rightFieldName); err != nil {
			return err
		}
	}

	return nil
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

	valElem := reflect.New(val.Type()).Elem()

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
		valFieldName := valElem.FieldByName(field.Name)

		fieldPath := path

		// always add path - except when field is embedded/anonymous AND embeddedStructsAsFields is false (default)
		if !(field.Anonymous && !c.config.embeddedStructsAsFields) {
			fieldPath = copyAppend(fieldPath, tName)
		}

		// skip private fields
		if !val.CanInterface() {
			continue
		}

		err := nc.compare(fieldPath, valFieldName, valField)
		if err != nil {
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
