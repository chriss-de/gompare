package gompare

import "reflect"

// cmpInterface compares two interfaces. If the interfaces are not nil we compare the underlying value of the interfaces
func (c *Comparer) cmpInterface(path []string, left, right reflect.Value) error {
	// one side is missing: compare against the value inside the interface so containers get expanded
	if left.Kind() == reflect.Invalid && !right.IsNil() {
		return c.compare(path, left, right.Elem())
	}
	if right.Kind() == reflect.Invalid && !left.IsNil() {
		return c.compare(path, left.Elem(), right)
	}

	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if left.IsNil() && right.IsNil() {
		return nil
	}

	if left.IsNil() {
		return c.cmpFromNil(path, reflect.Value{}, right.Elem())
	}

	if right.IsNil() {
		return c.cmpFromNil(path, left.Elem(), reflect.Value{})
	}

	return c.compare(path, left.Elem(), right.Elem())
}
