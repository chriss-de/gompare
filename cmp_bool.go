package gompare

import "reflect"

// cmpBool compares tow objects as bool
func (c *Comparer) cmpBool(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Bool() != right.Bool() {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
