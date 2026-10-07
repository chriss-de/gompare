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
	// index maps element values to their (not yet matched) indexes. It is only built for
	// elements of basic kind, where Go equality is the same as a full compare.
	index map[any][]int
}

// newSliceTracker creates a new sliceTracker for slice
func newSliceTracker(slice reflect.Value, cmp *Comparer) *sliceTracker {
	st := &sliceTracker{
		slice:   slice,
		cmp:     cmp,
		matches: make([]bool, slice.Len()),
	}

	if isBasicKind(slice.Type().Elem().Kind()) {
		st.index = make(map[any][]int, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			key := getAsAny(slice.Index(i))
			st.index[key] = append(st.index[key], i)
		}
	}

	return st
}

// isBasicKind reports if values of kind k are fully compared by Go's == operator
func isBasicKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// has checks if the slice has an element equal to val and marks it as matched.
// A type mismatch between two elements just means they are not equal, any other error is returned.
func (st *sliceTracker) has(val reflect.Value) (bool, error) {
	// fast path for basic kinds: look the value up instead of scanning the slice
	if st.index != nil {
		key := getAsAny(val)
		idxs := st.index[key]
		if len(idxs) == 0 {
			return false, nil
		}
		st.index[key] = idxs[1:]
		st.matches[idxs[0]] = true
		return true, nil
	}

	for i := 0; i < st.slice.Len(); i++ {
		// skip already matched elements
		if st.matches[i] {
			continue
		}

		x := st.slice.Index(i)

		// deeply equal values have no differences - skip the full compare
		if reflect.DeepEqual(getAsAny(x), getAsAny(val)) {
			st.matches[i] = true
			return true, nil
		}

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
