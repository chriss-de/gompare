package gompare

import "reflect"

// cmpString compares two objects as string values
func (c *Comparer) cmpString(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.String() != right.String() {
		if left.CanInterface() {
			// if we cant access it as string - use any
			c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
		} else {
			c.differences.add(CHANGED, path, left.String(), right.String())
		}
	}

	return nil
}
