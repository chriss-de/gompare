package gompare

import (
	"iter"
)

// Differences is a list of elements of Difference items
type Differences []Difference

// DiffType specifies the in what way it is different
type DiffType string

const (
	// ADDED represents when an element has been added
	ADDED DiffType = "added"
	// CHANGED represents when an element has been updated
	CHANGED DiffType = "changed"
	// REMOVED represents when an element has been removed
	REMOVED DiffType = "removed"
)

// DifferenceFilterFunc is a function to filter for Difference`s in Differences
type DifferenceFilterFunc func(Difference) bool

// Difference stores information about a changed item
type Difference struct {
	Type  DiffType `json:"type"`
	Path  []string `json:"path"`
	Left  any      `json:"left"`
	Right any      `json:"right"`
}

// add adds Difference's to Differences with all information about
func (d *Differences) add(dt DiffType, path []string, left any, right any) {
	diff := Difference{
		Type:  dt,
		Path:  path,
		Left:  left,
		Right: right,
	}
	*d = append(*d, diff)
}

// patchLeftAndRight is used by cmpMap and cmpStruct to correct LEFT and RIGHT after comparing a MAP or STRUCT
// When LEFT/RIGHT are reflect.Invalid we have to update DiffType and LEFT/RIGHT to get the correct result
func (d *Difference) patchLeftAndRight(dt DiffType) {
	d.Type = dt

	switch dt {
	case ADDED:
		d.Left = nil
	case REMOVED:
		d.Left = d.Right
		d.Right = nil
	}
}

// GetDifferences returns Difference's as iter.Seq filtered by filter functions
func (d *Differences) GetDifferences(filterFunc ...DifferenceFilterFunc) iter.Seq[Difference] {
	return func(yield func(diff Difference) bool) {
		for _, k := range *d {
			var yieldIt = true
			for _, ff := range filterFunc {
				// filterFunc are AND
				if !ff(k) {
					yieldIt = false
				}
			}
			if yieldIt && !yield(k) {
				return
			}
		}
	}
}

// HasDifferences returns true/false if Difference´s exists with applied filters
func (d *Differences) HasDifferences(filterFunc ...DifferenceFilterFunc) bool {
	for _, k := range *d {
		for _, ff := range filterFunc {
			// filterFunc are AND
			if !ff(k) {
				return false
			}
		}
	}
	return true
}
