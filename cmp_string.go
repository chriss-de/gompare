package gompare

import "reflect"

// cmpString compares two objects as string values
func (c *Comparer) cmpString(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.String() != right.String() {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
