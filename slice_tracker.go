package gompare

import (
	"errors"
	"reflect"
)

// sliceTracker holds a slice and a list of matches
// you can check if slice contains an element
type sliceTracker struct {
	matches []bool
	cmp     *Comparer
	slice   reflect.Value
}

// newSliceTracker creates a new sliceTracker for slice
func newSliceTracker(slice reflect.Value, cmp *Comparer) *sliceTracker {
	st := &sliceTracker{
		slice:   slice,
		cmp:     cmp,
		matches: make([]bool, slice.Len()),
	}

	return st
}

// has checks if the slice has an element equal to val and marks it as matched.
// A type mismatch between two elements just means they are not equal, any other error is returned.
func (st *sliceTracker) has(val reflect.Value) (bool, error) {
	for i := 0; i < st.slice.Len(); i++ {
		// skip already matched elements
		if st.matches[i] {
			continue
		}

		x := st.slice.Index(i)

		var nc *Comparer = st.cmp.clone()

		// compare and check if the elements are identical
		err := nc.compare([]string{}, x, val)
		if err != nil {
			if errors.Is(err, ErrTypeMismatch) {
				continue
			}
			return false, err
		}

		if len(nc.differences) == 0 {
			st.matches[i] = true
			return true, nil
		}
	}

	return false, nil
}
