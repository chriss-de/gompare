package gompare

import "reflect"

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

// Has checks if the slice has a val inside it and compare's it
func (st *sliceTracker) has(val reflect.Value) bool {

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
			continue
		}

		if len(nc.differences) == 0 {
			st.matches[i] = true
			return true
		}
	}

	return false
}
