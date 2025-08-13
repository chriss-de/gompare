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

// isComparable checks if left and right contains identifiable objects and can be compared
func (c *Comparer) isComparable(left, right reflect.Value) bool {
	if left.Len() > 0 {
		leftElem := left.Index(0)
		leftVal := getFinalValue(leftElem)

		if leftVal.Kind() == reflect.Struct {
			if getIdentifier(c.config.tagName, leftVal, string(c.config.combinedIdentifierJoinSep)) != nil {
				return true
			}
		}
	}

	if right.Len() > 0 {
		rightElem := right.Index(0)
		rightVal := getFinalValue(rightElem)

		if rightVal.Kind() == reflect.Struct {
			if getIdentifier(c.config.tagName, rightVal, string(c.config.combinedIdentifierJoinSep)) != nil {
				return true
			}
		}
	}

	return false
}
