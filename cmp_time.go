package gompare

import (
	"reflect"
	"time"
)

// cmpTime compares two items of time objects
func (c *Comparer) cmpTime(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	leftNano := getAsAny(left).(time.Time).UnixNano()
	rightNano := getAsAny(right).(time.Time).UnixNano()

	if leftNano != rightNano {
		c.differences.add(CHANGED, path, getAsAny(left), getAsAny(right))
	}

	return nil
}
