package gompare

import "reflect"

// processComparableList processes all keys in a comparableList and compare's both elements
func (c *Comparer) processComparableList(path []string, cmpList *comparableList) error {
	for _, k := range cmpList.keys {
		id, err := getID(k, c.config.structMapKeys)
		if err != nil {
			return pathError(err, path)
		}

		// set LEFT / RIGHT in List to nil if nil
		nv := reflect.ValueOf(nil)

		if cmpList.elems[k].LEFT == nil {
			cmpList.elems[k].LEFT = &nv
		}

		if cmpList.elems[k].RIGHT == nil {
			cmpList.elems[k].RIGHT = &nv
		}

		inListPath := copyAppend(path, id)

		if err := c.compare(inListPath, *cmpList.elems[k].LEFT, *cmpList.elems[k].RIGHT); err != nil {
			return err
		}
	}

	return nil
}

// isComparable checks if left and right contain identifiable objects and can be compared by identifier.
// The first element that is a struct (on either side, nil pointers are skipped) decides.
func (c *Comparer) isComparable(path []string, left, right reflect.Value) (bool, error) {
	for _, side := range []reflect.Value{left, right} {
		for i := 0; i < side.Len(); i++ {
			val := getFinalValue(side.Index(i))
			if val.Kind() != reflect.Struct {
				continue
			}

			id, err := c.getIdentifier(val)
			if err != nil {
				return false, pathError(err, path)
			}
			return id != nil, nil
		}
	}

	return false, nil
}
