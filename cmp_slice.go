package gompare

import "reflect"

// cmpSlice compares two slices , if the items of the slices are identifiable and comparable we go with cmpSliceComparable
// otherwise we use  cmpSliceGeneric
func (c *Comparer) cmpSlice(path []string, left, right reflect.Value) error {
	if changed, err := c.cmpDefault(path, left, right); err != nil || changed {
		return err
	}

	if c.isComparable(left, right) {
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
		// OR if config.sliceOrdering is disabled and the sliceTracker for LEFT does not has
		if (c.config.sliceOrdering && !hasAtSameIndex(right, leftElem, i)) || (!c.config.sliceOrdering && !rightSlice.has(leftElem)) {
			missing.addLeft(i, &leftElem)
		}
	}

	leftSlice := newSliceTracker(left, c)
	for i := 0; i < right.Len(); i++ {
		rightElem := right.Index(i)

		if (c.config.sliceOrdering && !hasAtSameIndex(left, rightElem, i)) || (!c.config.sliceOrdering && !leftSlice.has(rightElem)) {
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

// cmpSliceComparable compare's two slices if they have identifiable entries
func (c *Comparer) cmpSliceComparable(path []string, left, right reflect.Value) error {
	cmpList := NewComparableList()

	for i := 0; i < left.Len(); i++ {
		leftElem := left.Index(i)
		leftVal := getFinalValue(leftElem)

		leftID := getIdentifier(c.config.tagName, leftVal, string(c.config.combinedIdentifierJoinSep))
		if leftID != nil {
			cmpList.addLeft(leftID, &leftElem)
		}
	}

	for i := 0; i < right.Len(); i++ {
		rightElem := right.Index(i)
		rightVal := getFinalValue(rightElem)

		leftID := getIdentifier(c.config.tagName, rightVal, string(c.config.combinedIdentifierJoinSep))
		if leftID != nil {
			cmpList.addRight(leftID, &rightElem)
		}
	}

	return c.processComparableList(path, cmpList)
}
