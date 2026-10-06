package gompare

import "reflect"

// processComparableList processes all keys in a ComparableList and compare's both elements
func (c *Comparer) processComparableList(path []string, cmpList *ComparableList) error {
	for _, k := range cmpList.keys {
		id := getID(k, c.config.structMapKeys)

		// set LEFT / RIGHT in List to nil if nil
		nv := reflect.ValueOf(nil)

		if cmpList.elems[k].LEFT == nil {
			cmpList.elems[k].LEFT = &nv
		}

		if cmpList.elems[k].RIGHT == nil {
			cmpList.elems[k].RIGHT = &nv
		}

		inListPath := copyAppend(path, id)

		err := c.compare(inListPath, *cmpList.elems[k].LEFT, *cmpList.elems[k].RIGHT)
		if err != nil {
			return err
		}
	}

	return nil
}

// isComparable checks if left and right contain identifiable objects and can be compared by identifier.
// Only the first element of each side is inspected.
func (c *Comparer) isComparable(path []string, left, right reflect.Value) (bool, error) {
	for _, side := range []reflect.Value{left, right} {
		if side.Len() == 0 {
			continue
		}

		val := getFinalValue(side.Index(0))
		if val.Kind() != reflect.Struct {
			continue
		}

		id, err := c.getIdentifier(val)
		if err != nil {
			return false, pathError(err, path)
		}
		if id != nil {
			return true, nil
		}
	}

	return false, nil
}
