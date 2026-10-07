package gompare

import "reflect"

// cmpPtr compare's two entities from pointer. If both pointers are not nil we compare the underlying value.
// compare guarantees that left and right are both pointers or that one of them is missing (reflect.Invalid).
func (c *Comparer) cmpPtr(path []string, left, right reflect.Value) error {
	if left.Kind() == reflect.Invalid {
		if !right.IsNil() {
			return c.compare(path, reflect.Value{}, reflect.Indirect(right))
		}

		c.differences.add(ADDED, path, nil, getAsAny(right))
		return nil
	}

	if right.Kind() == reflect.Invalid {
		if !left.IsNil() {
			return c.compare(path, reflect.Indirect(left), reflect.Value{})
		}

		c.differences.add(REMOVED, path, getAsAny(left), nil)
		return nil
	}

	if left.IsNil() && right.IsNil() {
		return nil
	}

	if left.IsNil() {
		return c.cmpFromNil(path, reflect.Value{}, right)
	}

	if right.IsNil() {
		return c.cmpFromNil(path, left, reflect.Value{})
	}

	return c.compare(path, reflect.Indirect(left), reflect.Indirect(right))
}
