package gompare

import "reflect"

// cmpFloat compares two objects a floats
func (c *Comparer) cmpFloat(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Float() != right.Float() {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
