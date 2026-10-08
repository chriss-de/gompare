package gompare

import (
	"fmt"
	"reflect"
)

// cmpMap compare's two maps
// If one is empty/invalid we add all keys as ADDED/REMOVED
// otherwise compare each key. An identifier template on the struct field holding the map identifies
// the entries by their value instead of their key.
func (c *Comparer) cmpMap(path []string, left, right reflect.Value) error {
	elemIdentifier := c.takeElemIdentifier()

	if left.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(ADDED, path, right, elemIdentifier)
	}
	if right.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(REMOVED, path, left, elemIdentifier)
	}

	// with summarizeMissing an empty map on one side is one entry holding the other side
	if c.config.summarizeMissing {
		if left.Len() == 0 && right.Len() > 0 {
			c.differences.add(ADDED, path, nil, getAsAny(right))
			return nil
		}
		if right.Len() == 0 && left.Len() > 0 {
			c.differences.add(REMOVED, path, getAsAny(left), nil)
			return nil
		}
	}

	cmpList := newComparableList()

	for _, k := range sortedMapKeys(left) {
		leftElem := left.MapIndex(k)
		key, err := c.mapElemKey(path, k, leftElem, elemIdentifier)
		if err != nil {
			return err
		}
		if !cmpList.addLeft(key, &leftElem) {
			return pathError(fmt.Errorf("%w: %v on left side", ErrDuplicateIdentifier, key), path)
		}
	}

	for _, k := range sortedMapKeys(right) {
		rightElem := right.MapIndex(k)
		key, err := c.mapElemKey(path, k, rightElem, elemIdentifier)
		if err != nil {
			return err
		}
		if !cmpList.addRight(key, &rightElem) {
			return pathError(fmt.Errorf("%w: %v on right side", ErrDuplicateIdentifier, key), path)
		}
	}

	return c.processComparableList(path, cmpList)
}

// mapElemKey returns the key a map entry is matched and listed by: the rendered identifier template if
// the holding field has one and the entry's value has an identifier, the map key otherwise
func (c *Comparer) mapElemKey(path []string, key, elem reflect.Value, elemIdentifier string) (any, error) {
	if elemIdentifier == "" {
		return getAsAny(key), nil
	}

	id, err := c.getElemIdentifier(elemIdentifier, elem)
	if err != nil {
		return nil, pathError(err, path)
	}
	if id != nil {
		return id, nil
	}
	return getAsAny(key), nil
}

// cmpMapValuesForInvalid is used by cmpMap for maps that are empty/invalid to note all Differences
func (c *Comparer) cmpMapValuesForInvalid(dt DiffType, path []string, val reflect.Value, elemIdentifier string) error {
	// compare on a clone so only the Differences of this map get patched below
	var nc *Comparer = c.clone()
	missing := reflect.Value{}

	for _, k := range sortedMapKeys(val) {
		elemKey, err := c.mapElemKey(path, k, val.MapIndex(k), elemIdentifier)
		if err != nil {
			return err
		}
		key, err := getID(elemKey, c.config.structMapKeys)
		if err != nil {
			return pathError(err, path)
		}

		if err := nc.compare(copyAppend(path, key), missing, val.MapIndex(k)); err != nil {
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
