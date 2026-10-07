package gompare

import "reflect"

// cmpInt compares to objects as integers
func (c *Comparer) cmpInt(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Int() != right.Int() {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
