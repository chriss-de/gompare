package gompare

import "reflect"

// cmpUint compares two items of type uint and other unsigned integers
func (c *Comparer) cmpUint(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Uint() != right.Uint() {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
