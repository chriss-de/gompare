package gompare

import "reflect"

// cmpPtr compare's two entities from pointer. If both pointers are not nil we compare the underlying value.
func (c *Comparer) cmpPtr(path []string, left, right reflect.Value) error {
	if left.Kind() != right.Kind() {
		if left.Kind() == reflect.Invalid {
			if !right.IsNil() {
				return c.compare(path, reflect.ValueOf(nil), reflect.Indirect(right))
			}

			c.differences.add(ADDED, path, nil, getAsAny(right))
			return nil
		}

		if right.Kind() == reflect.Invalid {
			if !left.IsNil() {
				return c.compare(path, reflect.Indirect(left), reflect.ValueOf(nil))
			}

			c.differences.add(REMOVED, path, getAsAny(left), nil)
			return nil
		}

		return ErrTypeMismatch
	}

	if left.IsNil() && right.IsNil() {
		return nil
	}

	if left.IsNil() {
		c.differences.add(CHANGED, path, nil, getAsAny(right))
		return nil
	}

	if right.IsNil() {
		c.differences.add(CHANGED, path, getAsAny(left), nil)
		return nil
	}

	return c.compare(path, reflect.Indirect(left), reflect.Indirect(right))
}
