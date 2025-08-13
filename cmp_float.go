package gompare

import "reflect"

// cmpFloat compares two objects a floats
func (c *Comparer) cmpFloat(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.Float() != right.Float() {
		if left.CanInterface() {
			c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
		} else {
			c.differences.add(CHANGED, path, left.Float(), right.Float())
		}
	}

	return nil
}
