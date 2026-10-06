package gompare

import (
	"fmt"
	"reflect"
)

// cmpSlice compares two slices , if the items of the slices are identifiable and comparable we go with cmpSliceComparable
// otherwise we use  cmpSliceGeneric
func (c *Comparer) cmpSlice(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	comparable, err := c.isComparable(path, left, right)
	if err != nil {
		return err
	}
	if comparable {
		return c.cmpSliceComparable(path, left, right)
	}

	return c.cmpSliceGeneric(path, left, right)
}

// cmpSliceGeneric uses sliceTracker to track every element in the slices and collects missing elements of left and right
func (c *Comparer) cmpSliceGeneric(path []string, left, right reflect.Value) error {
	missing := NewComparableList()

	rightSlice := newSliceTracker(right, c)
	for i := 0; i < left.Len(); i++ {
		leftElem := left.Index(i)

		// if config.sliceOrdering is enabled and its not at the same index we add it to the missing objects
		// OR if config.sliceOrdering is disabled and the sliceTracker for RIGHT does not have it
		found, err := c.sliceHas(right, rightSlice, leftElem, i)
		if err != nil {
			return err
		}
		if !found {
			missing.addLeft(i, &leftElem)
		}
	}

	leftSlice := newSliceTracker(left, c)
	for i := 0; i < right.Len(); i++ {
		rightElem := right.Index(i)

		found, err := c.sliceHas(left, leftSlice, rightElem, i)
		if err != nil {
			return err
		}
		if !found {
			missing.addRight(i, &rightElem)
		}
	}

	// fallback to comparing based on order in slice if item is missing
	if len(missing.keys) == 0 {
		return nil
	}

	// process the ComparableList of missing objects of LEFT/RIGHT
	return c.processComparableList(path, missing)
}

// sliceHas checks if elem is in slice: at index idx if slices are ordered, anywhere otherwise
func (c *Comparer) sliceHas(slice reflect.Value, tracker *sliceTracker, elem reflect.Value, idx int) (bool, error) {
	if c.config.sliceOrdering {
		return hasAtSameIndex(slice, elem, idx), nil
	}
	return tracker.has(elem)
}

// cmpSliceComparable compare's two slices if they have identifiable entries.
// Elements without an identifier (e.g. nil pointers) are keyed by their index instead.
// An identifier that appears more than once on one side results in ErrDuplicateIdentifier.
func (c *Comparer) cmpSliceComparable(path []string, left, right reflect.Value) error {
	cmpList := NewComparableList()

	for i := 0; i < left.Len(); i++ {
		leftElem := left.Index(i)
		leftID, err := c.sliceElemKey(path, leftElem, i)
		if err != nil {
			return err
		}

		if !cmpList.addLeft(leftID, &leftElem) {
			return pathError(fmt.Errorf("%w: %q on left side", ErrDuplicateIdentifier, getID(leftID, false)), path)
		}
	}

	for i := 0; i < right.Len(); i++ {
		rightElem := right.Index(i)
		rightID, err := c.sliceElemKey(path, rightElem, i)
		if err != nil {
			return err
		}

		if !cmpList.addRight(rightID, &rightElem) {
			return pathError(fmt.Errorf("%w: %q on right side", ErrDuplicateIdentifier, getID(rightID, false)), path)
		}
	}

	return c.processComparableList(path, cmpList)
}

// sliceElemKey returns the identifier of a slice element or its index if it has none
func (c *Comparer) sliceElemKey(path []string, elem reflect.Value, idx int) (any, error) {
	id, err := c.getIdentifier(getFinalValue(elem))
	if err != nil {
		return nil, pathError(err, path)
	}
	if id != nil {
		return id, nil
	}
	return indexKey(idx), nil
}
