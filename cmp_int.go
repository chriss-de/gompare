package gompare

import "reflect"

// cmpInt compares to objects as integers
func (c *Comparer) cmpInt(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Int() != right.Int() {
		if left.CanInterface() {
			c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
		} else {
			c.differences.add(CHANGED, path, left.Int(), right.Int())
		}
	}

	return nil
}
