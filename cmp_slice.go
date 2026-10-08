package gompare

import (
	"fmt"
	"reflect"
)

// cmpSlice compares two slices , if the items of the slices are identifiable and comparable we go with cmpSliceComparable
// otherwise we use  cmpSliceGeneric
func (c *Comparer) cmpSlice(path []string, left, right reflect.Value) error {
	// an identifier template on the struct field holding this slice identifies its elements
	elemIdentifier := c.takeElemIdentifier()

	// a missing slice is compared as an empty one so every element gets listed
	if left.Kind() == reflect.Invalid {
		left = reflect.Zero(right.Type())
	}
	if right.Kind() == reflect.Invalid {
		right = reflect.Zero(left.Type())
	}

	// with summarizeMissing an empty slice on one side is one entry holding the other side
	if c.config.summarizeMissing {
		if left.Len() == 0 && right.Len() > 0 {
			c.differences.add(ADDED, path, nil, getAsAny(right))
			return nil
		}
		if right.Len() == 0 && left.Len() > 0 {
			c.differences.add(REMOVED, path, getAsAny(left), nil)
			return nil
		}
	}

	if elemIdentifier != "" {
		return c.cmpSliceComparable(path, left, right, elemIdentifier)
	}

	comparable, err := c.isComparable(path, left, right)
	if err != nil {
		return err
	}
	if comparable {
		return c.cmpSliceComparable(path, left, right, "")
	}

	return c.cmpSliceGeneric(path, left, right)
}

// cmpSliceGeneric collects the elements missing on either side and compares those
func (c *Comparer) cmpSliceGeneric(path []string, left, right reflect.Value) error {
	missing := newComparableList()

	leftMissing, err := c.missingElems(left, right)
	if err != nil {
		return err
	}
	for _, i := range leftMissing {
		elem := left.Index(i)
		missing.addLeft(i, &elem)
	}

	rightMissing, err := c.missingElems(right, left)
	if err != nil {
		return err
	}
	for _, i := range rightMissing {
		elem := right.Index(i)
		missing.addRight(i, &elem)
	}

	if len(missing.keys) == 0 {
		return nil
	}

	// elements left over on both sides are paired by their index and compared against each other
	return c.processComparableList(path, missing)
}

// missingElems returns the indexes of the elements of slice that have no counterpart in other:
// at the same index if slices are ordered, anywhere otherwise
func (c *Comparer) missingElems(slice, other reflect.Value) ([]int, error) {
	var (
		tracker *sliceTracker
		missing []int
	)
	if !c.config.sliceOrdering {
		tracker = newSliceTracker(other, c)
	}

	for i := 0; i < slice.Len(); i++ {
		elem := slice.Index(i)

		var found bool
		if c.config.sliceOrdering {
			found = hasAtSameIndex(other, elem, i)
		} else {
			var err error
			if found, err = tracker.has(elem); err != nil {
				return nil, err
			}
		}

		if !found {
			missing = append(missing, i)
		}
	}

	return missing, nil
}

// cmpSliceComparable compare's two slices if they have identifiable entries. With elemIdentifier set the
// elements are identified by that template instead of their own identifier fields.
// Elements without an identifier (e.g. nil pointers) are keyed by their index instead.
// An identifier that appears more than once on one side results in ErrDuplicateIdentifier.
func (c *Comparer) cmpSliceComparable(path []string, left, right reflect.Value, elemIdentifier string) error {
	cmpList := newComparableList()

	for i := 0; i < left.Len(); i++ {
		leftElem := left.Index(i)
		leftID, err := c.sliceElemKey(path, leftElem, i, elemIdentifier)
		if err != nil {
			return err
		}

		if !cmpList.addLeft(leftID, &leftElem) {
			return pathError(fmt.Errorf("%w: %v on left side", ErrDuplicateIdentifier, leftID), path)
		}
	}

	for i := 0; i < right.Len(); i++ {
		rightElem := right.Index(i)
		rightID, err := c.sliceElemKey(path, rightElem, i, elemIdentifier)
		if err != nil {
			return err
		}

		if !cmpList.addRight(rightID, &rightElem) {
			return pathError(fmt.Errorf("%w: %v on right side", ErrDuplicateIdentifier, rightID), path)
		}
	}

	return c.processComparableList(path, cmpList)
}

// sliceElemKey returns the identifier of a slice element or its index if it has none.
// With elemIdentifier set the identifier is rendered through that template.
func (c *Comparer) sliceElemKey(path []string, elem reflect.Value, idx int, elemIdentifier string) (any, error) {
	var (
		id  any
		err error
	)
	if elemIdentifier != "" {
		id, err = c.getElemIdentifier(elemIdentifier, elem)
	} else {
		id, err = c.getIdentifier(getFinalValue(elem))
	}
	if err != nil {
		return nil, pathError(err, path)
	}
	if id != nil {
		return id, nil
	}
	return indexKey(idx), nil
}
