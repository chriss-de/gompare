package gompare

import "reflect"

// cmpInterface compares two interfaces. If the interfaces are not nil we compare the underlying value of the interfaces
func (c *Comparer) cmpInterface(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
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

	return c.compare(path, left.Elem(), right.Elem())
}
