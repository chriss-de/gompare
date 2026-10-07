package gompare

import "reflect"

// cmpMap compare's two maps
// If one is empty/invalid we add all keys as ADDED/REMOVED
// otherwise compare each key
func (c *Comparer) cmpMap(path []string, left, right reflect.Value) error {
	if left.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(ADDED, path, right)
	}
	if right.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(REMOVED, path, left)
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
		cmpList.addLeft(getAsAny(k), &leftElem)
	}

	for _, k := range sortedMapKeys(right) {
		rightElem := right.MapIndex(k)
		cmpList.addRight(getAsAny(k), &rightElem)
	}

	return c.processComparableList(path, cmpList)
}

// cmpMapValuesForInvalid is used by cmpMap for maps that are empty/invalid to note all Differences
func (c *Comparer) cmpMapValuesForInvalid(dt DiffType, path []string, val reflect.Value) error {
	// compare on a clone so only the Differences of this map get patched below
	var nc *Comparer = c.clone()
	missing := reflect.Value{}

	for _, k := range sortedMapKeys(val) {
		key, err := getID(getAsAny(k), c.config.structMapKeys)
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
